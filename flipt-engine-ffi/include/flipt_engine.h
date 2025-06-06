#include <stdarg.h>
#include <stdbool.h>
#include <stdint.h>
#include <stdlib.h>

/**
 * # Safety
 *
 * This function will initialize an Engine and return a pointer back to the caller.
 */
void *initialize_engine_ffi(const char *opts);

/**
 * # Safety
 *
 * This function will initialize an Engine and return a pointer back to the caller.
 */
void *initialize_engine(const char *opts);

/**
 * # Safety
 *
 * This function will take in a pointer to the engine and return the current snapshot as a JSON string.
 */
const char *get_snapshot_ffi(void *engine_ptr);

/**
 * # Safety
 *
 * This function will take in a pointer to the engine and return the current snapshot as a JSON string.
 */
const char *get_snapshot(void *engine_ptr);

/**
 * # Safety
 *
 * This function will take in a pointer to the engine and return a variant evaluation response.
 */
const char *evaluate_variant_ffi(void *engine_ptr, const char *evaluation_request);

/**
 * # Safety
 *
 * This function will take in a pointer to the engine and return a variant evaluation response.
 */
const char *evaluate_variant(void *engine_ptr, const char *evaluation_request);

/**
 * # Safety
 *
 * This function will take in a pointer to the engine and return a boolean evaluation response.
 */
const char *evaluate_boolean_ffi(void *engine_ptr, const char *evaluation_request);

/**
 * # Safety
 *
 * This function will take in a pointer to the engine and return a boolean evaluation response.
 */
const char *evaluate_boolean(void *engine_ptr, const char *evaluation_request);

/**
 * # Safety
 *
 * This function will take in a pointer to the engine and return a batch evaluation response.
 */
const char *evaluate_batch_ffi(void *engine_ptr, const char *batch_evaluation_request);

/**
 * # Safety
 *
 * This function will take in a pointer to the engine and return a batch evaluation response.
 */
const char *evaluate_batch(void *engine_ptr, const char *batch_evaluation_request);

/**
 * # Safety
 *
 * This function will take in a pointer to the engine and return a list of flags.
 */
const char *list_flags_ffi(void *engine_ptr);

/**
 * # Safety
 *
 * This function will take in a pointer to the engine and return a list of flags.
 */
const char *list_flags(void *engine_ptr);

/**
 * # Safety
 *
 * This function will take in a pointer to the engine and destroy it.
 */
void destroy_engine_ffi(void *engine_ptr);

/**
 * # Safety
 *
 * This function will take in a pointer to the engine and destroy it.
 */
void destroy_engine(void *engine_ptr);

/**
 * # Safety
 *
 * This function will take in a pointer to a string and destroy it.
 */
void destroy_string_ffi(char *ptr);

/**
 * # Safety
 *
 * This function will take in a pointer to a string and destroy it.
 */
void destroy_string(char *ptr);
