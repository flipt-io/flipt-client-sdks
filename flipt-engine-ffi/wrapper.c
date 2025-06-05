#include <stdlib.h>
#include "flipt_engine.h"

// Declare the Rust functions we're wrapping
extern void* initialize_engine_ffi(const char* options);
extern const char* evaluate_boolean_ffi(void* engine, const char* request);
extern const char* evaluate_variant_ffi(void* engine, const char* request);
extern const char* evaluate_batch_ffi(void* engine, const char* request);
extern const char* list_flags_ffi(void* engine);
extern const char* get_snapshot_ffi(void* engine);
extern void destroy_engine_ffi(void* engine);
extern void destroy_string_ffi(char* str);

// Wrapper functions that will be exported in our .so
void* initialize_engine(const char* options) {
    return initialize_engine_ffi(options);
}

const char* evaluate_boolean(void* engine, const char* request) {
    return evaluate_boolean_ffi(engine, request);
}

const char* evaluate_variant(void* engine, const char* request) {
    return evaluate_variant_ffi(engine, request);
}

const char* evaluate_batch(void* engine, const char* request) {
    return evaluate_batch_ffi(engine, request);
}

const char* list_flags(void* engine) {
    return list_flags_ffi(engine);
}

const char* get_snapshot(void* engine) {
    return get_snapshot_ffi(engine);
}

void destroy_engine(void* engine) {
    destroy_engine_ffi(engine);
}

void destroy_string(char* str) {
    destroy_string_ffi(str);
} 