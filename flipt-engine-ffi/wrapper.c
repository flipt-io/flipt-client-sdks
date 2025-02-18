#include <stdlib.h>
#include "flipt_engine.h"

// Declare the Rust functions we're wrapping
extern void* initialize_engine_rust(const char* namespace, const char* options);
extern const char* evaluate_boolean_rust(void* engine, const char* request);
extern const char* evaluate_variant_rust(void* engine, const char* request);
extern const char* evaluate_batch_rust(void* engine, const char* request);
extern const char* list_flags_rust(void* engine);
extern void destroy_engine_rust(void* engine);
extern void destroy_string_rust(char* str);

// Wrapper functions that will be exported in our .so
void* initialize_engine(const char* namespace, const char* options) {
    return initialize_engine_rust(namespace, options);
}

const char* evaluate_boolean(void* engine, const char* request) {
    return evaluate_boolean_rust(engine, request);
}

const char* evaluate_variant(void* engine, const char* request) {
    return evaluate_variant_rust(engine, request);
}

const char* evaluate_batch(void* engine, const char* request) {
    return evaluate_batch_rust(engine, request);
}

const char* list_flags(void* engine) {
    return list_flags_rust(engine);
}

void destroy_engine(void* engine) {
    destroy_engine_rust(engine);
}

void destroy_string(char* str) {
    destroy_string_rust(str);
} 