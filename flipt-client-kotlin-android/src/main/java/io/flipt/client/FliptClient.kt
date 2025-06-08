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
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlin.time.Duration

/**
 * Thrown when a validation error occurs in FliptClient.
 */
class ValidationException(message: String) : IllegalArgumentException(message)

class FliptClient(
    options: ClientOptions,
) {
    private var engine: Long = 0
    private val json =
        Json {
            encodeDefaults = false
            ignoreUnknownKeys = true
        }

    init {
        val clientOptionsSerialized = json.encodeToString(options)
        engine =
            CLibrary.INSTANCE.initializeEngine(
                clientOptionsSerialized,
            )
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
         * Builds a new FliptClient.
         * @return the FliptClient
         * @throws EvaluationException if the FliptClient could not be built
         */
        @Throws(EvaluationException::class)
        fun build(): FliptClient {
            val requestTimeout = this.requestTimeout?.inWholeSeconds
            val updateInterval = this.updateInterval?.inWholeSeconds
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
            return FliptClient(options)
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
        var value: String? = null
        try {
            value = CLibrary.INSTANCE.evaluateVariant(engine, evaluationRequestSerialized)
            val resp: Result<VariantEvaluationResponse> =
                json.decodeFromString(Result.serializer(VariantEvaluationResponse.serializer()), value)
            if (resp.status != "success") {
                throw EvaluationException(resp.errorMessage ?: "Unknown error")
            }
            return resp.result ?: throw EvaluationException(resp.errorMessage ?: "No result returned from engine")
        } finally {
            if (value != null) {
                CLibrary.INSTANCE.destroyString(value)
            }
        }
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
        var value: String? = null
        try {
            value = CLibrary.INSTANCE.evaluateBoolean(engine, evaluationRequestSerialized)
            val resp: Result<BooleanEvaluationResponse> =
                json.decodeFromString(Result.serializer(BooleanEvaluationResponse.serializer()), value)
            if (resp.status != "success") {
                throw EvaluationException(resp.errorMessage ?: "Unknown error")
            }
            return resp.result ?: throw EvaluationException(resp.errorMessage ?: "No result returned from engine")
        } finally {
            if (value != null) {
                CLibrary.INSTANCE.destroyString(value)
            }
        }
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
        var value: String? = null
        try {
            value = CLibrary.INSTANCE.evaluateBatch(engine, batchEvaluationRequestSerialized)
            val resp: Result<BatchEvaluationResponse> =
                json.decodeFromString(Result.serializer(BatchEvaluationResponse.serializer()), value)
            if (resp.status != "success") {
                throw EvaluationException(resp.errorMessage ?: "Unknown error")
            }
            return resp.result ?: throw EvaluationException(resp.errorMessage ?: "No result returned from engine")
        } finally {
            if (value != null) {
                CLibrary.INSTANCE.destroyString(value)
            }
        }
    }

    /**
     * Lists all flags in the namespace.
     * @return the list of flags
     * @throws EvaluationException if listing fails
     */
    fun listFlags(): List<Flag> {
        var value: String? = null
        try {
            value = CLibrary.INSTANCE.listFlags(engine)
            val resp = json.decodeFromString<Result<ArrayList<Flag>>>(value)
            if (resp.status != "success") {
                throw EvaluationException(resp.errorMessage ?: "Unknown error")
            }
            return resp.result ?: throw EvaluationException(resp.errorMessage ?: "No result returned from engine")
        } finally {
            if (value != null) {
                CLibrary.INSTANCE.destroyString(value)
            }
        }
    }

    /**
     * Gets the snapshot from the Flipt client.
     * @return the snapshot
     */
    fun getSnapshot(): String {
        var value: String? = null
        try {
            value = CLibrary.INSTANCE.getSnapshot(engine)
            return value
        } finally {
            if (value != null) {
                CLibrary.INSTANCE.destroyString(value)
            }
        }
    }

    /**
     * Closes the Flipt client and releases resources.
     */
    fun close() {
        CLibrary.INSTANCE.destroyEngine(engine)
    }

    companion object {
        /**
         * Returns a new builder for FliptClient.
         */
        fun builder(): FliptClientBuilder = FliptClientBuilder()
    }
}
