from pydantic import BaseModel
import ctypes
import os
import platform

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
)

from typing import List


class InternalEvaluationRequest(BaseModel):
    namespace_key: str
    flag_key: str
    entity_id: str
    context: dict


class FliptEvaluationClient:
    def __init__(
        self, namespace: str = "default", opts: ClientOptions = ClientOptions()
    ):
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
            raise Exception(f"Unsupported platform/processor: {platform_name}/{arch}")

        # Get the absolute path to the engine library from the ../ext directory
        engine_library_path = os.path.join(
            os.path.dirname(os.path.abspath(__file__)), f"../ext/{libfile}"
        )

        if not os.path.exists(engine_library_path):
            raise Exception(
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

        ns = namespace.encode("utf-8")

        client_opts_serialized = opts.model_dump_json(exclude_none=True).encode("utf-8")

        self.engine = self.ffi_core.initialize_engine(ns, client_opts_serialized)

    def __del__(self):
        if hasattr(self, "engine") and self.engine is not None:
            self.ffi_core.destroy_engine(self.engine)

    def evaluate_variant(
        self, flag_key: str, entity_id: str, context: dict = {}
    ) -> VariantEvaluationResponse:
        if not flag_key or not flag_key.strip():
            raise ValueError("flag_key cannot be empty or null")
        if not entity_id or not entity_id.strip():
            raise ValueError("entity_id cannot be empty or null")

        response = self.ffi_core.evaluate_variant(
            self.engine,
            serialize_evaluation_request(
                self.namespace_key, flag_key, entity_id, context
            ),
        )

        bytes_returned = ctypes.cast(response, ctypes.c_char_p).value
        variant_result = VariantResult.model_validate_json(bytes_returned)
        self.ffi_core.destroy_string(response)

        if variant_result.status != "success":
            raise Exception(variant_result.error_message)

        return variant_result.result

    def evaluate_boolean(
        self, flag_key: str, entity_id: str, context: dict = {}
    ) -> BooleanEvaluationResponse:
        if not flag_key or not flag_key.strip():
            raise ValueError("flag_key cannot be empty or null")
        if not entity_id or not entity_id.strip():
            raise ValueError("entity_id cannot be empty or null")

        response = self.ffi_core.evaluate_boolean(
            self.engine,
            serialize_evaluation_request(
                self.namespace_key, flag_key, entity_id, context
            ),
        )

        bytes_returned = ctypes.cast(response, ctypes.c_char_p).value
        boolean_result = BooleanResult.model_validate_json(bytes_returned)
        self.ffi_core.destroy_string(response)

        if boolean_result.status != "success":
            raise Exception(boolean_result.error_message)

        return boolean_result.result

    def evaluate_batch(
        self, requests: List[EvaluationRequest]
    ) -> BatchEvaluationResponse:
        evaluation_requests = []

        for r in requests:
            if not r.flag_key or not r.flag_key.strip():
                raise ValueError("flag_key cannot be empty or null")
            if not r.entity_id or not r.entity_id.strip():
                raise ValueError("entity_id cannot be empty or null")

            evaluation_requests.append(
                InternalEvaluationRequest(
                    namespace_key=self.namespace_key,
                    flag_key=r.flag_key,
                    entity_id=r.entity_id,
                    context=r.context,
                )
            )

        json_list = [
            evaluation_request.model_dump_json()
            for evaluation_request in evaluation_requests
        ]
        json_string = "[" + ", ".join(json_list) + "]"

        response = self.ffi_core.evaluate_batch(
            self.engine, json_string.encode("utf-8")
        )

        bytes_returned = ctypes.cast(response, ctypes.c_char_p).value
        batch_result = BatchResult.model_validate_json(bytes_returned)
        self.ffi_core.destroy_string(response)

        if batch_result.status != "success":
            raise Exception(batch_result.error_message)

        return batch_result.result

    def list_flags(self) -> FlagList:
        response = self.ffi_core.list_flags(self.engine)

        bytes_returned = ctypes.cast(response, ctypes.c_char_p).value
        result = ListFlagsResult.model_validate_json(bytes_returned)
        self.ffi_core.destroy_string(response)

        if result.status != "success":
            raise Exception(result.error_message)

        return result.result


def serialize_evaluation_request(
    namespace_key: str, flag_key: str, entity_id: str, context: dict
) -> str:
    if not flag_key or not flag_key.strip():
        raise ValueError("flag_key cannot be empty or null")
    if not entity_id or not entity_id.strip():
        raise ValueError("entity_id cannot be empty or null")

    evaluation_request = InternalEvaluationRequest(
        namespace_key=namespace_key,
        flag_key=flag_key,
        entity_id=entity_id,
        context=context,
    )

    return evaluation_request.model_dump_json().encode("utf-8")
