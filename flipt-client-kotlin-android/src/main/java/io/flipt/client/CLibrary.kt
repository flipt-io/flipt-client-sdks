package io.flipt.client

class CLibrary {
    init {
        System.loadLibrary("fliptengine_wrapper")
    }

    external fun initializeEngine(options: String): Long

    external fun evaluateVariant(
        enginePtr: Long,
        evaluationRequest: String,
    ): String

    external fun evaluateBoolean(
        enginePtr: Long,
        evaluationRequest: String,
    ): String

    external fun evaluateBatch(
        enginePtr: Long,
        batchRequest: String,
    ): String

    external fun listFlags(enginePtr: Long): String

    external fun getSnapshot(enginePtr: Long): String

    external fun destroyEngine(enginePtr: Long)

    /** Returns a subscription handle, or 0 if the engine pointer is invalid. */
    external fun subscribeStateUpdates(enginePtr: Long): Long

    /** Blocks until flag state changes or the subscription is closed. */
    external fun waitForStateUpdate(subscriptionPtr: Long): String

    /** Wakes a blocked [waitForStateUpdate]. Safe to call from any thread. */
    external fun closeStateSubscription(subscriptionPtr: Long)

    /** Frees a subscription. No thread may be waiting on it. */
    external fun destroyStateSubscription(subscriptionPtr: Long)

    companion object {
        val INSTANCE = CLibrary()
    }
}
