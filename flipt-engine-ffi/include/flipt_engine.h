#include <stdarg.h>
#include <stdbool.h>
#include <stdint.h>
#include <stdlib.h>

/**
 * Called by the engine after it applies new flag state. `user_data` is the
 * pointer the host passed at registration, returned untouched.
 * Nullable in C: passing NULL clears the registered callback.
 */
typedef void (*StateUpdateCallback)(void *user_data);

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
 * Registers `callback` to be called, with `user_data`, after the engine applies
 * changed flag state. Pass a null callback to clear it. Callbacks run on a
 * single engine-owned thread; bursts of updates are coalesced into one call.
 * Initial state and transitions into an error state do not fire; recovering
 * from an error does.
 *
 * A callback may evaluate flags and may call `destroy_engine`, but changing
 * the callback from inside one returns an error. `user_data` must stay valid
 * until the callback is cleared or `destroy_engine` returns. The callback
 * must not unwind: a panic across this boundary aborts the process.
 */
const char *set_state_update_callback_ffi(void *engine_ptr,
                                          StateUpdateCallback callback,
                                          void *user_data);

/**
 * # Safety
 *
 * Registers `callback` to be called, with `user_data`, after the engine applies
 * changed flag state. Pass a null callback to clear it. Callbacks run on a
 * single engine-owned thread; bursts of updates are coalesced into one call.
 * Initial state and transitions into an error state do not fire; recovering
 * from an error does.
 *
 * A callback may evaluate flags and may call `destroy_engine`, but changing
 * the callback from inside one returns an error. `user_data` must stay valid
 * until the callback is cleared or `destroy_engine` returns. The callback
 * must not unwind: a panic across this boundary aborts the process.
 */
const char *set_state_update_callback(void *engine_ptr,
                                      StateUpdateCallback callback,
                                      void *user_data);

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
