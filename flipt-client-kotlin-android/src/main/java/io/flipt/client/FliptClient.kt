package io.flipt.client

import io.flipt.client.models.AuthenticationStrategy
import io.flipt.client.models.BatchEvaluationResponse
import io.flipt.client.models.BooleanEvaluationResponse
import io.flipt.client.models.ClientOptions
import io.flipt.client.models.ErrorStrategy
import io.flipt.client.models.EvaluationRequest
import io.flipt.client.models.FetchMode
import io.flipt.client.models.Flag
import io.flipt.client.models.Result
import io.flipt.client.models.VariantEvaluationResponse
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.builtins.serializer
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import java.util.concurrent.CopyOnWriteArrayList
import java.util.concurrent.atomic.AtomicBoolean
import java.util.logging.Level
import java.util.logging.Logger
import kotlin.time.Duration
import kotlin.time.Duration.Companion.seconds

/**
 * Thrown when a validation error occurs in FliptClient.
 */
class ValidationException(
    message: String,
) : IllegalArgumentException(message)

class FliptClient(
    options: ClientOptions,
) {
    private var engine: Long = 0
    private val json =
        Json {
            encodeDefaults = false
            ignoreUnknownKeys = true
        }

    private val closed = AtomicBoolean(false)

    // Each entry wraps one registration, so adding the same lambda twice gives two
    // independent subscriptions.
    private class ListenerEntry(
        val listener: () -> Unit,
    )

    private val stateUpdateListeners = CopyOnWriteArrayList<ListenerEntry>()

    // Guards stateUpdateThread and stateSubscription. Never held while joining the thread,
    // since a running listener may add another listener.
    private val stateUpdateLock = Any()
    private var stateUpdateThread: Thread? = null
    private var stateSubscription: Long = 0

    init {
        val clientOptionsSerialized = json.encodeToString(options)
        engine =
            CLibrary.INSTANCE.initializeEngine(
                clientOptionsSerialized,
            )
    }

    // Used by the builder, which can reach it as a nested class. Private, so Java callers don't see
    // a public (ClientOptions, Function0) overload. Runs after the primary constructor, so the
    // engine and listener state already exist.
    private constructor(options: ClientOptions, onStateUpdate: (() -> Unit)?) : this(options) {
        if (onStateUpdate != null) {
            try {
                addStateUpdateListener(onStateUpdate)
            } catch (t: Throwable) {
                // The caller never gets a client to close, so tear the engine down here.
                CLibrary.INSTANCE.destroyEngine(engine)
                throw t
            }
        }
    }

    class FliptClientBuilder {
        private var environment: String = "default"
        private var namespace: String = "default"
        private var url: String = "http://localhost:8080"
        private var authentication: AuthenticationStrategy? = null
        private var reference: String? = null
        private var requestTimeout: Duration? = null
        private var updateInterval: Duration? = 120.seconds
        private var fetchMode: FetchMode = FetchMode.POLLING
        private var errorStrategy: ErrorStrategy = ErrorStrategy.FAIL
        private var snapshot: String? = null
        private var onStateUpdate: (() -> Unit)? = null

        /**
         * Sets the environment for the Flipt server.
         * @param environment the environment for the Flipt server
         * @return the FliptClientBuilder
         */
        fun environment(environment: String): FliptClientBuilder {
            this.environment = environment
            return this
        }

        /**
         * Sets the namespace for the Flipt server.
         * @param namespace the namespace for the Flipt server
         * @return the FliptClientBuilder
         */
        fun namespace(namespace: String): FliptClientBuilder {
            this.namespace = namespace
            return this
        }

        /**
         * Sets the URL for the Flipt server.
         * @param url the URL for the Flipt server
         * @return the FliptClientBuilder
         */
        fun url(url: String): FliptClientBuilder {
            this.url = url
            return this
        }

        /**
         * Sets the authentication strategy for the Flipt server.
         * @param authentication the authentication strategy
         * @return the FliptClientBuilder
         */
        fun authentication(authentication: AuthenticationStrategy?): FliptClientBuilder {
            this.authentication = authentication
            return this
        }

        /**
         * Sets the request timeout for the Flipt server.
         * @param requestTimeout the request timeout
         * @return the FliptClientBuilder
         */
        fun requestTimeout(requestTimeout: Duration?): FliptClientBuilder {
            this.requestTimeout = requestTimeout
            return this
        }

        /**
         * Sets the update interval for the Flipt server.
         * @param updateInterval the update interval
         * @return the FliptClientBuilder
         */
        fun updateInterval(updateInterval: Duration?): FliptClientBuilder {
            this.updateInterval = updateInterval
            return this
        }

        /**
         * Sets the reference for the Flipt server.
         * @param reference the reference
         * @return the FliptClientBuilder
         */
        fun reference(reference: String?): FliptClientBuilder {
            this.reference = reference
            return this
        }

        /**
         * Sets the fetch mode for the Flipt server.
         * @param fetchMode the fetch mode
         * @return the FliptClientBuilder
         */
        fun fetchMode(fetchMode: FetchMode): FliptClientBuilder {
            this.fetchMode = fetchMode
            return this
        }

        /**
         * Sets the error strategy for the Flipt server.
         * @param errorStrategy the error strategy
         * @return the FliptClientBuilder
         */
        fun errorStrategy(errorStrategy: ErrorStrategy): FliptClientBuilder {
            this.errorStrategy = errorStrategy
            return this
        }

        /**
         * Sets the initial snapshot for evaluation.
         * @param snapshot the initial snapshot
         * @return the FliptClientBuilder
         */
        fun snapshot(snapshot: String?): FliptClientBuilder {
            this.snapshot = snapshot
            return this
        }

        /**
         * Sets a listener that is called after the client applies changed flag state.
         * See [FliptClient.addStateUpdateListener] for how listeners behave.
         * @param onStateUpdate the listener
         * @return the FliptClientBuilder
         */
        fun onStateUpdate(onStateUpdate: (() -> Unit)?): FliptClientBuilder {
            this.onStateUpdate = onStateUpdate
            return this
        }

        /**
         * Builds a new FliptClient.
         * @return the FliptClient
         * @throws EvaluationException if the FliptClient could not be built
         */
        @Throws(EvaluationException::class)
        fun build(): FliptClient {
            val options =
                ClientOptions(
                    environment = environment,
                    namespace = namespace,
                    url = url,
                    requestTimeout = requestTimeout,
                    updateInterval = updateInterval,
                    authentication = authentication,
                    reference = reference,
                    fetchMode = fetchMode,
                    errorStrategy = errorStrategy,
                    snapshot = snapshot,
                )
            return FliptClient(options, onStateUpdate)
        }
    }

    @Serializable
    internal data class InternalEvaluationRequest(
        @SerialName("flag_key") val flagKey: String,
        @SerialName("entity_id") val entityId: String,
        @SerialName("context") val context: Map<String, String>,
    )

    /**
     * Evaluates a variant flag.
     * @param flagKey the key for the flag to evaluate (must not be null or empty)
     * @param entityId the ID for the entity to evaluate (must not be null or empty)
     * @param context the context for the evaluation (must not be null)
     * @return the evaluation response
     * @throws ValidationException if parameters are invalid
     * @throws EvaluationException if evaluation fails
     */
    fun evaluateVariant(
        flagKey: String,
        entityId: String,
        context: Map<String, String>,
    ): VariantEvaluationResponse {
        if (flagKey.isEmpty()) throw ValidationException("flagKey is required")
        if (entityId.isEmpty()) throw ValidationException("entityId is required")
        val evaluationRequest = InternalEvaluationRequest(flagKey, entityId, context)
        val evaluationRequestSerialized = json.encodeToString(evaluationRequest)
        val value = CLibrary.INSTANCE.evaluateVariant(engine, evaluationRequestSerialized)
        val resp: Result<VariantEvaluationResponse> =
            json.decodeFromString(Result.serializer(VariantEvaluationResponse.serializer()), value)
        if (resp.status != "success") {
            throw EvaluationException(resp.errorMessage ?: "Unknown error")
        }
        return resp.result ?: throw EvaluationException(resp.errorMessage ?: "No result returned from engine")
    }

    /**
     * Evaluates a boolean flag.
     * @param flagKey the key for the flag to evaluate (must not be null or empty)
     * @param entityId the ID for the entity to evaluate (must not be null or empty)
     * @param context the context for the evaluation (must not be null)
     * @return the evaluation response
     * @throws ValidationException if parameters are invalid
     * @throws EvaluationException if evaluation fails
     */
    fun evaluateBoolean(
        flagKey: String,
        entityId: String,
        context: Map<String, String>,
    ): BooleanEvaluationResponse {
        if (flagKey.isEmpty()) throw ValidationException("flagKey is required")
        if (entityId.isEmpty()) throw ValidationException("entityId is required")
        val evaluationRequest = InternalEvaluationRequest(flagKey, entityId, context)
        val evaluationRequestSerialized = json.encodeToString(evaluationRequest)
        val value = CLibrary.INSTANCE.evaluateBoolean(engine, evaluationRequestSerialized)
        val resp: Result<BooleanEvaluationResponse> =
            json.decodeFromString(Result.serializer(BooleanEvaluationResponse.serializer()), value)
        if (resp.status != "success") {
            throw EvaluationException(resp.errorMessage ?: "Unknown error")
        }
        return resp.result ?: throw EvaluationException(resp.errorMessage ?: "No result returned from engine")
    }

    /**
     * Evaluates a batch of flags.
     * @param batchEvaluationRequest the batch of flags to evaluate (must not be null)
     * @return the evaluation response
     * @throws ValidationException if parameters are invalid
     * @throws EvaluationException if evaluation fails
     */
    fun evaluateBatch(batchEvaluationRequest: Array<EvaluationRequest>): BatchEvaluationResponse {
        if (batchEvaluationRequest.isEmpty()) throw ValidationException("batchEvaluationRequest must not be empty")
        val evaluationrequests =
            batchEvaluationRequest.map {
                InternalEvaluationRequest(it.flagKey, it.entityId, it.context)
            }
        val batchEvaluationRequestSerialized = json.encodeToString(evaluationrequests)
        val value = CLibrary.INSTANCE.evaluateBatch(engine, batchEvaluationRequestSerialized)
        val resp: Result<BatchEvaluationResponse> =
            json.decodeFromString(Result.serializer(BatchEvaluationResponse.serializer()), value)
        if (resp.status != "success") {
            throw EvaluationException(resp.errorMessage ?: "Unknown error")
        }
        return resp.result ?: throw EvaluationException(resp.errorMessage ?: "No result returned from engine")
    }

    /**
     * Lists all flags in the namespace.
     * @return the list of flags
     * @throws EvaluationException if listing fails
     */
    fun listFlags(): List<Flag> {
        val value = CLibrary.INSTANCE.listFlags(engine)
        val resp = json.decodeFromString<Result<ArrayList<Flag>>>(value)
        if (resp.status != "success") {
            throw EvaluationException(resp.errorMessage ?: "Unknown error")
        }
        return resp.result ?: throw EvaluationException(resp.errorMessage ?: "No result returned from engine")
    }

    /**
     * Gets the snapshot from the Flipt client.
     * @return the snapshot
     */
    fun getSnapshot(): String = CLibrary.INSTANCE.getSnapshot(engine)

    /** A registered state update listener. Closing it unsubscribes; closing twice is a no-op. */
    fun interface Subscription : AutoCloseable {
        override fun close()
    }

    /**
     * Adds a listener that is called after each change to flag state. Listeners can be added at
     * any time. Read the flags you care about once after adding a listener: it is called for
     * changes applied after it was added, and may occasionally see one extra call.
     *
     * Listeners run in the order they were added, on a dedicated daemon thread named
     * `flipt-state-update`. Updates that arrive while they run are coalesced into a single
     * follow-up call. A listener may evaluate flags, add or remove listeners, or call [close].
     * [close] from another thread waits for a running listener to return, so a listener must not
     * block on anything held by the thread that closes the client. An exception thrown by one
     * listener is logged and does not stop the others.
     *
     * The client must be closed to stop its update thread.
     *
     * @param listener the listener to call
     * @return a subscription that removes the listener when closed
     * @throws IllegalStateException if the client is closed
     * @throws EvaluationException if the engine cannot subscribe to updates
     */
    @Throws(EvaluationException::class)
    fun addStateUpdateListener(listener: () -> Unit): Subscription {
        check(!closed.get()) { "client is closed" }
        val entry = ListenerEntry(listener)
        stateUpdateListeners.add(entry)
        try {
            ensureStateUpdateThread()
        } catch (t: Throwable) {
            stateUpdateListeners.remove(entry)
            throw t
        }
        // close() may have finished between the checks above and the add.
        if (closed.get()) {
            stateUpdateListeners.remove(entry)
            error("client is closed")
        }
        return Subscription { stateUpdateListeners.remove(entry) }
    }

    /**
     * Starts the single thread that waits on the engine and fans out to all listeners, on first
     * use. It runs until [close], so adding or removing listeners never calls into the engine,
     * including from inside a listener.
     *
     * Why a plain [Thread] and not a coroutine: the thread spends its life inside
     * `waitForStateUpdate`, a blocking JNI call into the engine. A coroutine can't suspend there,
     * so it would hold a pool thread (e.g. from Dispatchers.IO) just the same, cancelling it
     * would not unblock the native wait (that still needs `closeStateSubscription`), and it would
     * add a kotlinx-coroutines dependency. A dedicated thread also mirrors the Java SDK one to one.
     */
    private fun ensureStateUpdateThread() {
        synchronized(stateUpdateLock) {
            if (stateUpdateThread != null) {
                return
            }
            check(!closed.get()) { "client is closed" }
            val subscription = CLibrary.INSTANCE.subscribeStateUpdates(engine)
            if (subscription == 0L) {
                throw EvaluationException("Failed to subscribe to state updates")
            }
            val thread = Thread({ runStateUpdateLoop(subscription) }, "flipt-state-update")
            // Never keep the process alive just to wait for flag changes.
            thread.isDaemon = true
            try {
                thread.start()
            } catch (t: Throwable) {
                CLibrary.INSTANCE.destroyStateSubscription(subscription)
                throw t
            }
            // The thread can't exit before these are set: its cleanup takes this lock.
            stateSubscription = subscription
            stateUpdateThread = thread
        }
    }

    private fun runStateUpdateLoop(subscription: Long) {
        try {
            while (!closed.get()) {
                val value = CLibrary.INSTANCE.waitForStateUpdate(subscription)
                val resp = json.decodeFromString(Result.serializer(String.serializer()), value)
                if (resp.status != "success") {
                    logger.warning("Failed to wait for state update: ${resp.errorMessage ?: "Unknown error"}")
                    return
                }
                if (resp.result != "changed") {
                    return
                }
                for (entry in stateUpdateListeners) {
                    if (closed.get()) {
                        return
                    }
                    try {
                        entry.listener()
                    } catch (t: Throwable) {
                        logger.log(Level.WARNING, "State update listener threw", t)
                    }
                }
            }
        } catch (t: Throwable) {
            logger.log(Level.WARNING, "State update thread failed", t)
        } finally {
            // This thread is the only one that waits on the subscription, so it frees it. Under
            // the lock, so close() never wakes a handle that is already freed.
            synchronized(stateUpdateLock) {
                stateSubscription = 0
                CLibrary.INSTANCE.destroyStateSubscription(subscription)
                // If this thread died early, the next listener starts a new one.
                if (stateUpdateThread === Thread.currentThread()) {
                    stateUpdateThread = null
                }
            }
        }
    }

    /**
     * Closes the Flipt client and releases resources. Calling it more than once is a no-op.
     *
     * If a state update listener is running on another thread, waits for it to return first, so
     * it never evaluates against a freed engine.
     */
    fun close() {
        // destroyEngine frees the engine, so a second call would free it twice.
        if (!closed.compareAndSet(false, true)) {
            return
        }
        // Wait out a thread start already in progress; later ones see closed and give up. The
        // lock is released before joining, since a running listener may add another listener.
        val thread: Thread?
        synchronized(stateUpdateLock) {
            thread = stateUpdateThread
            if (stateSubscription != 0L) {
                // Wakes the update thread if it is waiting. It still owns and frees the handle.
                CLibrary.INSTANCE.closeStateSubscription(stateSubscription)
            }
        }
        if (thread != null && thread !== Thread.currentThread()) {
            // Let a running listener finish before the engine it may be evaluating against is freed.
            var interrupted = false
            while (thread.isAlive) {
                try {
                    thread.join()
                } catch (e: InterruptedException) {
                    interrupted = true
                }
            }
            if (interrupted) {
                Thread.currentThread().interrupt()
            }
        }
        CLibrary.INSTANCE.destroyEngine(engine)
        stateUpdateListeners.clear()
    }

    companion object {
        private val logger: Logger = Logger.getLogger(FliptClient::class.java.name)

        /**
         * Returns a new builder for FliptClient.
         */
        fun builder(): FliptClientBuilder = FliptClientBuilder()
    }
}
