#include <jni.h>
#include "include/flipt_engine.h"
#include <stdlib.h>
#include <string.h>

jlong JNICALL Java_io_flipt_client_CLibrary_initializeEngine(
        JNIEnv *env, jobject obj, jstring namespace_, jstring opts) {
    const char *namespace_c = (*env)->GetStringUTFChars(env, namespace_, 0);
    const char *opts_c = (*env)->GetStringUTFChars(env, opts, 0);

    void *engine = initialize_engine(namespace_c, opts_c);

    (*env)->ReleaseStringUTFChars(env, namespace_, namespace_c);
    (*env)->ReleaseStringUTFChars(env, opts, opts_c);

    return (jlong) engine; // Cast to jlong to safely pass the pointer
}

jstring JNICALL Java_io_flipt_client_CLibrary_evaluateVariant(
        JNIEnv *env, jobject obj, jlong enginePtr, jstring evaluationRequest) {
    const char *eval_request_c = (*env)->GetStringUTFChars(env, evaluationRequest, 0);

    const char *result = evaluate_variant((void *) enginePtr, eval_request_c);

    (*env)->ReleaseStringUTFChars(env, evaluationRequest, eval_request_c);

    jstring result_j = (*env)->NewStringUTF(env, result);
    destroy_string((char *) result); // Free the result string

    return result_j;
}

jstring JNICALL Java_io_flipt_client_CLibrary_evaluateBoolean(
        JNIEnv *env, jobject obj, jlong enginePtr, jstring evaluationRequest) {
    const char *eval_request_c = (*env)->GetStringUTFChars(env, evaluationRequest, 0);

    const char *result = evaluate_boolean((void *) enginePtr, eval_request_c);

    (*env)->ReleaseStringUTFChars(env, evaluationRequest, eval_request_c);

    jstring result_j = (*env)->NewStringUTF(env, result);
    destroy_string((char *) result); // Free the result string

    return result_j;
}

jstring JNICALL Java_io_flipt_client_CLibrary_evaluateBatch(
        JNIEnv *env, jobject obj, jlong enginePtr, jstring batchEvaluationRequest) {
    const char *batch_request_c = (*env)->GetStringUTFChars(env, batchEvaluationRequest, 0);

    const char *result = evaluate_batch((void *) enginePtr, batch_request_c);

    (*env)->ReleaseStringUTFChars(env, batchEvaluationRequest, batch_request_c);

    jstring result_j = (*env)->NewStringUTF(env, result);
    destroy_string((char *) result); // Free the result string

    return result_j;
}

jstring JNICALL Java_io_flipt_client_CLibrary_listFlags(
        JNIEnv *env, jobject obj, jlong enginePtr) {
    const char *result = list_flags((void *) enginePtr);

    jstring result_j = (*env)->NewStringUTF(env, result);
    destroy_string((char *) result); // Free the result string

    return result_j;
}

void JNICALL Java_io_flipt_client_CLibrary_destroyEngine(
        JNIEnv *env, jobject obj, jlong enginePtr) {
    destroy_engine((void *) enginePtr);
}

void JNICALL Java_io_flipt_client_CLibrary_destroyString(
        JNIEnv *env, jobject obj, jstring ptr) {
    const char *str = (*env)->GetStringUTFChars(env, ptr, 0);
    destroy_string((char *) str);
    (*env)->ReleaseStringUTFChars(env, ptr, str);
}
