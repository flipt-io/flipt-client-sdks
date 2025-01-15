#include <stdarg.h>
#include <stdbool.h>
#include <stdint.h>
#include <stdlib.h>

/**
 * # Safety
 *
 * This function will initialize an Engine and return a pointer back to the caller.
 */
void *initialize_engine(const char *namespace_, const char *opts);

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
const char *evaluate_boolean(void *engine_ptr, const char *evaluation_request);

/**
 * # Safety
 *
 * This function will take in a pointer to the engine and return a batch evaluation response.
 */
const char *evaluate_batch(void *engine_ptr, const char *batch_evaluation_request);

/**
 * # Safety
 *
 * This function will take in a pointer to the engine and return a list of flags for the given namespace.
 */
const char *list_flags(void *engine_ptr);

/**
 * # Safety
 *
 * This function will free the memory occupied by the engine.
 */
void destroy_engine(void *engine_ptr);

/**
 * # Safety
 *
 * This function will take in a pointer to the string and free the memory.
 * See Rust the safety section in CString::from_raw.
 */
void destroy_string(char *ptr);
