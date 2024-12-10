package io.flipt.client


class CLibrary {
    init {
        System.loadLibrary("fliptengine_wrapper")
    }

    external fun initializeEngine(namespace: String, opts: String): Long
    external fun evaluateVariant(enginePtr: Long, evaluationRequest: String): String
    external fun evaluateBoolean(enginePtr: Long, evaluationRequest: String): String
    external fun evaluateBatch(enginePtr: Long, batchRequest: String): String
    external fun listFlags(enginePtr: Long): String
    external fun destroyEngine(enginePtr: Long)
    external fun destroyString(ptr: String)

    companion object {
        val INSTANCE = CLibrary()
    }
}