import ctypes
import os
import platform

from .models import (
    BooleanResult,
    EngineOpts,
    EvaluationRequest,
    VariantResult,
)


class FliptEvaluationClient:
    def __init__(self, namespace: str = "default", engine_opts: EngineOpts = {}):
        # first try to load the engine library from FLIPT_ENGINE_LIB_PATH env var if it exists
        engine_library_path = os.environ.get("FLIPT_ENGINE_LIB_PATH")

        if engine_library_path is None:
            # get dynamic library extension for the current platform

            if platform.system() == "Windows":
                libfile = "fliptengine.dll"
            elif platform.system() == "Darwin":
                libfile = "libfliptengine.dylib"
            else:
                libfile = "libfliptengine.so"

            # if not set, get the absolute path to the engine library from the ../ext directory
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
        self.ffi_core.evaluate_variant.restype = ctypes.c_char_p

        self.ffi_core.evaluate_boolean.argtypes = [ctypes.c_void_p, ctypes.c_char_p]
        self.ffi_core.evaluate_boolean.restype = ctypes.c_char_p

        namespace_list = [namespace]

        ns = (ctypes.c_char_p * len(namespace_list))()
        ns[:] = [s.encode("utf-8") for s in namespace_list]

        engine_opts_serialized = engine_opts.model_dump_json().encode("utf-8")

        self.engine = self.ffi_core.initialize_engine(ns, engine_opts_serialized)

    def __del__(self):
        if hasattr(self, "engine") and self.engine is not None:
            self.ffi_core.destroy_engine(self.engine)

    def evaluate_variant(
        self, flag_key: str, entity_id: str, context: dict
    ) -> VariantResult:
        response = self.ffi_core.evaluate_variant(
            self.engine,
            serialize_evaluation_request(
                self.namespace_key, flag_key, entity_id, context
            ),
        )

        bytes_returned = ctypes.c_char_p(response).value

        variant_result = VariantResult.model_validate_json(bytes_returned)

        return variant_result

    def evaluate_boolean(
        self, flag_key: str, entity_id: str, context: dict
    ) -> BooleanResult:
        response = self.ffi_core.evaluate_boolean(
            self.engine,
            serialize_evaluation_request(
                self.namespace_key, flag_key, entity_id, context
            ),
        )

        bytes_returned = ctypes.c_char_p(response).value

        boolean_result = BooleanResult.model_validate_json(bytes_returned)

        return boolean_result


def serialize_evaluation_request(
    namespace_key: str, flag_key: str, entity_id: str, context: dict
) -> str:
    evaluation_request = EvaluationRequest(
        namespace_key=namespace_key,
        flag_key=flag_key,
        entity_id=entity_id,
        context=context,
    )

    return evaluation_request.model_dump_json().encode("utf-8")
