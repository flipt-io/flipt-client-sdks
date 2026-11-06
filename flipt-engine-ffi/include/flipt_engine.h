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
 * This function will update the authentication for the engine.
 */
const char *update_authentication_ffi(void *engine_ptr, const char *auth_json);

/**
 * # Safety
 *
 * This function will update the authentication for the engine.
 */
const char *update_authentication(void *engine_ptr, const char *auth_json);

/**
 * # Safety
 *
 * Subscribes to flag state changes applied after this call returns (a change
 * applied just before may also be reported once), and returns a subscription
 * handle, or null if `engine_ptr` is null. Pass the
 * handle to `wait_for_state_update` from a thread the host owns.
 *
 * The handle holds its own reference to the engine's update state, so it
 * stays valid after `destroy_engine`; waiting on it then reports `closed`.
 * Free it with `destroy_state_subscription`.
 */
void *subscribe_state_updates_ffi(void *engine_ptr);

/**
 * # Safety
 *
 * Subscribes to flag state changes applied after this call returns (a change
 * applied just before may also be reported once), and returns a subscription
 * handle, or null if `engine_ptr` is null. Pass the
 * handle to `wait_for_state_update` from a thread the host owns.
 *
 * The handle holds its own reference to the engine's update state, so it
 * stays valid after `destroy_engine`; waiting on it then reports `closed`.
 * Free it with `destroy_state_subscription`.
 */
void *subscribe_state_updates(void *engine_ptr);

/**
 * # Safety
 *
 * Blocks until flag state changes, or until the subscription or engine is
 * closed. Returns a JSON response whose `result` is `"changed"` or
 * `"closed"`. Changes that land while the host is busy are coalesced into
 * one `"changed"`. Initial state and transitions into an error state are
 * not reported; recovering from an error is. Once a wait returns `"closed"`,
 * every later wait does too, without blocking.
 */
const char *wait_for_state_update_ffi(void *subscription_ptr);

/**
 * # Safety
 *
 * Blocks until flag state changes, or until the subscription or engine is
 * closed. Returns a JSON response whose `result` is `"changed"` or
 * `"closed"`. Changes that land while the host is busy are coalesced into
 * one `"changed"`. Initial state and transitions into an error state are
 * not reported; recovering from an error is. Once a wait returns `"closed"`,
 * every later wait does too, without blocking.
 */
const char *wait_for_state_update(void *subscription_ptr);

/**
 * # Safety
 *
 * Wakes any thread blocked in `wait_for_state_update` on this subscription
 * and makes every later wait return `"closed"`. Safe to call from any thread,
 * and more than once. Does not free the handle.
 */
void close_state_subscription_ffi(void *subscription_ptr);

/**
 * # Safety
 *
 * Wakes any thread blocked in `wait_for_state_update` on this subscription
 * and makes every later wait return `"closed"`. Safe to call from any thread,
 * and more than once. Does not free the handle.
 */
void close_state_subscription(void *subscription_ptr);

/**
 * # Safety
 *
 * Closes and frees a subscription handle. No other thread may be using the
 * handle, including waiting on it, when this is called.
 */
void destroy_state_subscription_ffi(void *subscription_ptr);

/**
 * # Safety
 *
 * Closes and frees a subscription handle. No other thread may be using the
 * handle, including waiting on it, when this is called.
 */
void destroy_state_subscription(void *subscription_ptr);

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
