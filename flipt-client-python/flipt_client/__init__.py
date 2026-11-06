from pydantic import BaseModel
import ctypes
import json
import logging
import os
import platform
import threading
import warnings
from collections.abc import Callable
from typing import Optional
from .errors import FliptError, ValidationError, EvaluationError

from .models import (
    BatchEvaluationResponse,
    BatchResult,
    BooleanEvaluationResponse,
    BooleanResult,
    ClientOptions,
    EvaluationRequest,
    FlagList,
    ListFlagsResult,
    VariantEvaluationResponse,
    VariantResult,
    model_to_json,
    model_from_json,
)

from typing import List

logger = logging.getLogger(__name__)


class InternalEvaluationRequest(BaseModel):
    namespace_key: str
    flag_key: str
    entity_id: str
    context: dict


class Subscription:
    """A registered state update listener. Closing it unsubscribes; closing twice is a no-op."""

    def __init__(self, unsubscribe: Callable[[], None]):
        self._unsubscribe = unsubscribe

    def close(self) -> None:
        self._unsubscribe()

    def __enter__(self) -> "Subscription":
        return self

    def __exit__(self, *exc) -> None:
        self.close()


class FliptClient:
    """Main client for interacting with Flipt feature flag engine."""

    def __init__(
        self,
        opts: ClientOptions = ClientOptions(),
        on_state_update: Callable[[], None] | None = None,
    ):
        self.engine = None
        self._closed = False
        # Guards the fields below and _closed. Never held while running listeners, joining the
        # update thread or destroying the engine.
        self._state_update_lock = threading.Lock()
        # Replaced, never mutated, so the update thread can iterate a snapshot without the lock.
        self._state_update_listeners: list[Callable[[], None]] = []
        # Started on the first listener. It owns the native subscription and frees it on exit.
        self._state_update_thread: threading.Thread | None = None
        self._state_subscription = None

        namespace = opts.namespace or "default"
        # Mapping of platform-architecture combinations to their respective library file paths
        lib_files = {
            "Darwin-x86_64": "darwin_x86_64/libfliptengine.dylib",
            "Darwin-arm64": "darwin_aarch64/libfliptengine.dylib",
            "Darwin-aarch64": "darwin_aarch64/libfliptengine.dylib",
            "Linux-x86_64": "linux_x86_64/libfliptengine.so",
            "Linux-arm64": "linux_aarch64/libfliptengine.so",
            "Linux-aarch64": "linux_aarch64/libfliptengine.so",
            "Windows-x86_64": "windows_x86_64/fliptengine.dll",
        }

        platform_name = platform.system()
        arch = platform.machine()
        key = f"{platform_name}-{arch}"

        libfile = lib_files.get(key)

        if not libfile:
            raise FliptError(f"Unsupported platform/processor: {platform_name}/{arch}")

        # Get the absolute path to the engine library from the ../ext directory
        engine_library_path = os.path.join(
            os.path.dirname(os.path.abspath(__file__)), f"../ext/{libfile}"
        )

        if not os.path.exists(engine_library_path):
            raise FliptError(
                f"The engine library could not be found at the path: {engine_library_path}"
            )

        self.namespace_key = namespace

        self.ffi_core = ctypes.CDLL(engine_library_path)

        self.ffi_core.initialize_engine.restype = ctypes.c_void_p
        self.ffi_core.destroy_engine.argtypes = [ctypes.c_void_p]

        self.ffi_core.evaluate_variant.argtypes = [ctypes.c_void_p, ctypes.c_char_p]
        self.ffi_core.evaluate_variant.restype = ctypes.POINTER(ctypes.c_char_p)

        self.ffi_core.evaluate_boolean.argtypes = [ctypes.c_void_p, ctypes.c_char_p]
        self.ffi_core.evaluate_boolean.restype = ctypes.POINTER(ctypes.c_char_p)

        self.ffi_core.evaluate_batch.argtypes = [ctypes.c_void_p, ctypes.c_char_p]
        self.ffi_core.evaluate_batch.restype = ctypes.POINTER(ctypes.c_char_p)

        self.ffi_core.list_flags.argtypes = [ctypes.c_void_p]
        self.ffi_core.list_flags.restype = ctypes.POINTER(ctypes.c_char_p)

        self.ffi_core.destroy_string.argtypes = [ctypes.POINTER(ctypes.c_char_p)]
        self.ffi_core.destroy_string.restype = ctypes.c_void_p

        self.ffi_core.get_snapshot.argtypes = [ctypes.c_void_p]
        self.ffi_core.get_snapshot.restype = ctypes.POINTER(ctypes.c_char_p)

        self.ffi_core.subscribe_state_updates.argtypes = [ctypes.c_void_p]
        self.ffi_core.subscribe_state_updates.restype = ctypes.c_void_p

        # Blocks until flag state changes or the subscription closes. ctypes releases the GIL
        # for the duration of the call.
        self.ffi_core.wait_for_state_update.argtypes = [ctypes.c_void_p]
        self.ffi_core.wait_for_state_update.restype = ctypes.POINTER(ctypes.c_char_p)

        self.ffi_core.close_state_subscription.argtypes = [ctypes.c_void_p]
        self.ffi_core.close_state_subscription.restype = None

        self.ffi_core.destroy_state_subscription.argtypes = [ctypes.c_void_p]
        self.ffi_core.destroy_state_subscription.restype = None

        client_opts_serialized = model_to_json(opts, exclude_none=True).encode("utf-8")

        self.engine = self.ffi_core.initialize_engine(client_opts_serialized)

        if on_state_update is not None:
            try:
                self.add_state_update_listener(on_state_update)
            except BaseException:
                # The caller never gets a client to close, so tear the engine down here.
                self.close()
                raise

    def close(self):
        """
        Closes the client and releases the engine. When called from a thread other than the
        state update thread, waits for a running listener to return first.
        """
        # __del__ may run on a partially constructed client.
        lock = getattr(self, "_state_update_lock", None)
        if lock is None:
            return
        with lock:
            if self._closed:
                return
            self._closed = True
            thread = self._state_update_thread
            if self._state_subscription is not None:
                # Wakes the update thread if it is waiting. It still owns and frees the handle.
                self.ffi_core.close_state_subscription(self._state_subscription)
        if thread is not None and thread is not threading.current_thread():
            # Let a running listener finish before the engine it may be evaluating against is freed.
            thread.join()
        if self.engine is not None:
            self.ffi_core.destroy_engine(self.engine)
            self.engine = None
        self._state_update_listeners = []

    def add_state_update_listener(self, listener: Callable[[], None]) -> Subscription:
        """
        Adds a listener that is called after each change to flag state. Listeners can be added at
        any time. Read the flags you care about once after adding a listener: it is called for
        changes applied after it was added, and may occasionally see one extra call.

        Listeners run in the order they were added, on a dedicated daemon thread named
        ``flipt-state-update``. Updates that arrive while they run are coalesced into a single
        follow-up call. A listener may evaluate flags, add or remove listeners, or call
        ``close()``. ``close()`` from another thread waits for a running listener to return, so a
        listener must not block on anything held by the thread that closes the client. An
        exception raised by one listener is logged and does not stop the others.

        The client must be closed to stop its update thread.

        :param listener: the callable to call, with no arguments
        :return: a subscription that removes the listener when closed
        :raises FliptError: if the client is closed
        :raises EvaluationError: if the engine cannot subscribe to updates
        """
        if listener is None or not callable(listener):
            raise ValidationError("listener must be callable")

        # A distinct wrapper per call, so adding the same callable twice yields two independent
        # subscriptions.
        def entry():
            listener()

        with self._state_update_lock:
            if self._closed:
                raise FliptError("client is closed")
            self._ensure_state_update_thread()
            self._state_update_listeners = self._state_update_listeners + [entry]

        def unsubscribe():
            with self._state_update_lock:
                self._state_update_listeners = [
                    existing
                    for existing in self._state_update_listeners
                    if existing is not entry
                ]

        return Subscription(unsubscribe)

    def _ensure_state_update_thread(self):
        """
        Starts the single thread that waits on the engine and fans out to all listeners, on first
        use. Called with _state_update_lock held. It runs until close(), so adding or removing
        listeners never calls into the engine, including from inside a listener.
        """
        if self._state_update_thread is not None:
            return
        subscription = self.ffi_core.subscribe_state_updates(self.engine)
        if not subscription:
            raise EvaluationError("Failed to subscribe to state updates")
        # Never keep the interpreter alive just to wait for flag changes.
        thread = threading.Thread(
            target=self._run_state_update_loop,
            args=(subscription,),
            name="flipt-state-update",
            daemon=True,
        )
        try:
            thread.start()
        except BaseException:
            self.ffi_core.destroy_state_subscription(subscription)
            raise
        # The thread can't exit before these are set: its cleanup takes the lock we hold.
        self._state_subscription = subscription
        self._state_update_thread = thread

    def _run_state_update_loop(self, subscription):
        try:
            while not self._closed:
                response = self.ffi_core.wait_for_state_update(subscription)
                bytes_returned = ctypes.cast(response, ctypes.c_char_p).value
                self.ffi_core.destroy_string(response)
                if bytes_returned is None:
                    logger.warning("Failed to wait for state update: empty response")
                    return
                result = json.loads(bytes_returned)
                if result.get("status") != "success":
                    logger.warning(
                        "Failed to wait for state update: %s",
                        result.get("error_message") or "Unknown error",
                    )
                    return
                if result.get("result") != "changed":
                    return
                for listener in self._state_update_listeners:
                    if self._closed:
                        return
                    try:
                        listener()
                    # BaseException, so a listener calling sys.exit() is logged and does not
                    # silently stop notifications for the others.
                    except BaseException:
                        logger.exception("State update listener raised")
        except BaseException:
            logger.exception("State update thread failed")
        finally:
            # This thread is the only one that waits on the subscription, so it frees it. Under
            # the lock, so close() never wakes a handle that is already freed.
            with self._state_update_lock:
                self._state_subscription = None
                self.ffi_core.destroy_state_subscription(subscription)
                # If this thread died early, the next listener starts a new one.
                if self._state_update_thread is threading.current_thread():
                    self._state_update_thread = None

    def evaluate_variant(
        self, flag_key: str, entity_id: str, context: Optional[dict] = None
    ) -> VariantEvaluationResponse:
        if context is None:
            context = {}
        if not flag_key or not flag_key.strip():
            raise ValidationError("flag_key cannot be empty or null")
        if not entity_id or not entity_id.strip():
            raise ValidationError("entity_id cannot be empty or null")

        response = self.ffi_core.evaluate_variant(
            self.engine,
            serialize_evaluation_request(
                self.namespace_key, flag_key, entity_id, context
            ),
        )

        bytes_returned = ctypes.cast(response, ctypes.c_char_p).value
        variant_result = model_from_json(VariantResult, bytes_returned)
        self.ffi_core.destroy_string(response)

        if variant_result.status != "success":
            raise EvaluationError(variant_result.error_message)

        return variant_result.result

    def evaluate_boolean(
        self, flag_key: str, entity_id: str, context: Optional[dict] = None
    ) -> BooleanEvaluationResponse:
        if context is None:
            context = {}
        if not flag_key or not flag_key.strip():
            raise ValidationError("flag_key cannot be empty or null")
        if not entity_id or not entity_id.strip():
            raise ValidationError("entity_id cannot be empty or null")

        response = self.ffi_core.evaluate_boolean(
            self.engine,
            serialize_evaluation_request(
                self.namespace_key, flag_key, entity_id, context
            ),
        )

        bytes_returned = ctypes.cast(response, ctypes.c_char_p).value
        boolean_result = model_from_json(BooleanResult, bytes_returned)
        self.ffi_core.destroy_string(response)

        if boolean_result.status != "success":
            raise EvaluationError(boolean_result.error_message)

        return boolean_result.result

    def evaluate_batch(
        self, requests: List[EvaluationRequest]
    ) -> BatchEvaluationResponse:
        evaluation_requests = []

        for r in requests:
            if not r.flag_key or not r.flag_key.strip():
                raise ValidationError("flag_key cannot be empty or null")
            if not r.entity_id or not r.entity_id.strip():
                raise ValidationError("entity_id cannot be empty or null")

            evaluation_requests.append(
                InternalEvaluationRequest(
                    namespace_key=self.namespace_key,
                    flag_key=r.flag_key,
                    entity_id=r.entity_id,
                    context=r.context,
                )
            )

        json_list = [
            model_to_json(evaluation_request)
            for evaluation_request in evaluation_requests
        ]
        json_string = "[" + ", ".join(json_list) + "]"

        response = self.ffi_core.evaluate_batch(
            self.engine, json_string.encode("utf-8")
        )

        bytes_returned = ctypes.cast(response, ctypes.c_char_p).value
        batch_result = model_from_json(BatchResult, bytes_returned)
        self.ffi_core.destroy_string(response)

        if batch_result.status != "success":
            raise EvaluationError(batch_result.error_message)

        return batch_result.result

    def list_flags(self) -> FlagList:
        response = self.ffi_core.list_flags(self.engine)

        bytes_returned = ctypes.cast(response, ctypes.c_char_p).value
        result = model_from_json(ListFlagsResult, bytes_returned)
        self.ffi_core.destroy_string(response)

        if result.status != "success":
            raise EvaluationError(result.error_message)

        return result.result

    def get_snapshot(self) -> str:
        """
        Returns a snapshot of the current engine state as a base64 encoded JSON string.
        """
        response = self.ffi_core.get_snapshot(self.engine)
        snapshot_bytes = ctypes.cast(response, ctypes.c_char_p).value
        if hasattr(self.ffi_core, "destroy_string"):
            self.ffi_core.destroy_string(response)
        if snapshot_bytes is None:
            raise EvaluationError("Failed to get snapshot")
        return snapshot_bytes.decode("utf-8")

    def __del__(self):
        self.close()


def serialize_evaluation_request(
    namespace_key: str, flag_key: str, entity_id: str, context: dict
) -> str:
    if not flag_key or not flag_key.strip():
        raise ValidationError("flag_key cannot be empty or null")
    if not entity_id or not entity_id.strip():
        raise ValidationError("entity_id cannot be empty or null")

    evaluation_request = InternalEvaluationRequest(
        namespace_key=namespace_key,
        flag_key=flag_key,
        entity_id=entity_id,
        context=context,
    )

    return model_to_json(evaluation_request).encode("utf-8")


# Deprecation alias
class FliptEvaluationClient(FliptClient):
    def __init__(self, *args, **kwargs):
        warnings.warn(
            "FliptEvaluationClient is deprecated and will be removed in a future release. Use FliptClient instead.",
            DeprecationWarning,
            stacklevel=2,
        )
        super().__init__(*args, **kwargs)
