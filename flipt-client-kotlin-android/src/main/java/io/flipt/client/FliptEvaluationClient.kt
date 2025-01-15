package io.flipt.client

import io.flipt.client.models.AuthenticationStrategy
import io.flipt.client.models.BatchEvaluationResponse
import io.flipt.client.models.BooleanEvaluationResponse
import io.flipt.client.models.ClientOptions
import io.flipt.client.models.EvaluationRequest
import io.flipt.client.models.FetchMode
import io.flipt.client.models.Flag
import io.flipt.client.models.Result
import io.flipt.client.models.VariantEvaluationResponse
import kotlinx.serialization.DeserializationStrategy
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlin.time.Duration

class FliptEvaluationClient(namespace: String, options: ClientOptions) {
    private var engine: Long = 0
    private val json = Json {
        encodeDefaults = false
        ignoreUnknownKeys = true
    }

    init {
        val clientOptionsSerialized = json.encodeToString(options)
        engine = CLibrary.INSTANCE.initializeEngine(namespace, clientOptionsSerialized)
    }

    class FliptEvaluationClientBuilder {
        private var namespace = "default"
        private var url = "http://localhost:8080"
        private var authentication: AuthenticationStrategy? = null
        private var reference: String? = null
        private var updateInterval: Long? = null
        private var fetchMode = FetchMode.polling

        /**
         * url sets the URL for the Flipt server.
         *
         * @param url the URL for the Flipt server
         * @return the FliptEvaluationClientBuilder
         */
        fun url(url: String): FliptEvaluationClientBuilder {
            this.url = url
            return this
        }

        /**
         * namespace sets the namespace for the Flipt server.
         *
         * @param namespace the namespace for the Flipt server
         * @return the FliptEvaluationClientBuilder
         */
        fun namespace(namespace: String): FliptEvaluationClientBuilder {
            this.namespace = namespace
            return this
        }

        /**
         * authentication sets the authentication strategy for the Flipt server.
         *
         * @param authentication the authentication strategy for the Flipt server
         * @return the FliptEvaluationClientBuilder
         */
        fun authentication(authentication: AuthenticationStrategy?): FliptEvaluationClientBuilder {
            this.authentication = authentication
            return this
        }

        /**
         * updateInterval sets the update interval for the Flipt server.
         *
         * @param updateInterval the update interval for the Flipt server
         * @return the FliptEvaluationClientBuilder
         */
        fun updateInterval(updateInterval: Long?): FliptEvaluationClientBuilder {
            this.updateInterval = updateInterval
            return this
        }

        /**
         * reference sets the reference for the Flipt server.
         *
         * @param reference the reference for the Flipt server
         * @return the FliptEvaluationClientBuilder
         */
        fun reference(reference: String?): FliptEvaluationClientBuilder {
            this.reference = reference
            return this
        }

        /**
         * fetchMode sets the fetch mode for the Flipt server. Note: Streaming is currently only
         * supported when using the SDK with Flipt Cloud (https://flipt.io/cloud).
         *
         * @param fetchMode the fetch mode for the Flipt server
         * @return the FliptEvaluationClientBuilder
         */
        fun fetchMode(fetchMode: FetchMode): FliptEvaluationClientBuilder {
            this.fetchMode = fetchMode
            return this
        }

        /**
         * build builds a new FliptEvaluationClient.
         *
         * @return the FliptEvaluationClient
         * @throws EvaluationException if the FliptEvaluationClient could not be built
         */
        @Throws(EvaluationException::class)
        fun build(): FliptEvaluationClient {
            return FliptEvaluationClient(
                namespace,
                ClientOptions(
                    url,
                    updateInterval,
                    authentication,
                    reference,
                    fetchMode
                )
            )
        }
    }


    @Serializable
    internal data class InternalEvaluationRequest(
        @SerialName("flag_key") val flagKey: String,
        @SerialName("entity_id") val entityId: String,
        @SerialName("context") val context: Map<String, String>
    )

    fun evaluateVariant(
        flagKey: String,
        entityId: String,
        context: Map<String, String>
    ): VariantEvaluationResponse {
        val evaluationRequest = InternalEvaluationRequest(flagKey, entityId, context)
        val evaluationRequestSerialized = json.encodeToString(evaluationRequest)

        val value = CLibrary.INSTANCE.evaluateVariant(engine, evaluationRequestSerialized)

        val resp: Result<VariantEvaluationResponse> =
            json.decodeFromString(Result.serializer(VariantEvaluationResponse.serializer()), value)

        return resp.result ?: throw EvaluationException(resp.errorMessage ?: "Unknown Error")
    }

    fun evaluateBoolean(
        flagKey: String,
        entityId: String,
        context: Map<String, String>
    ): BooleanEvaluationResponse? {
        val evaluationRequest = InternalEvaluationRequest(flagKey, entityId, context)
        val evaluationRequestSerialized = json.encodeToString(evaluationRequest)

        val value = CLibrary.INSTANCE.evaluateBoolean(engine, evaluationRequestSerialized)

        val resp: Result<BooleanEvaluationResponse> =
            json.decodeFromString(Result.serializer(BooleanEvaluationResponse.serializer()), value)

        return resp.result ?: throw EvaluationException(resp.errorMessage ?: "Unknown Error")
    }

    fun evaluateBatch(batchEvaluationRequest: Array<EvaluationRequest>): BatchEvaluationResponse? {
        val evaluationrequests = mutableListOf<InternalEvaluationRequest>()

        batchEvaluationRequest.map {
            evaluationrequests.add(InternalEvaluationRequest(it.flagKey, it.entityId, it.context))
        }

        val batchEvaluationRequestSerialized = json.encodeToString(evaluationrequests)
        val value = CLibrary.INSTANCE.evaluateBatch(engine, batchEvaluationRequestSerialized)

        val resp: Result<BatchEvaluationResponse> = json.decodeFromString(
            Result.serializer(BatchEvaluationResponse.serializer()),
            value
        )

        return resp.result ?: throw EvaluationException(resp.errorMessage ?: "Unknown Error")
    }

    fun listFlags(): ArrayList<Flag>? {
        val value = CLibrary.INSTANCE.listFlags(engine)
        val resp = json.decodeFromString<Result<ArrayList<Flag>>>(value)
        return resp.result ?: throw EvaluationException(resp.errorMessage ?: "Unknown Error")
    }

    fun close() {
        CLibrary.INSTANCE.destroyEngine(engine)
    }

    companion object {
        fun builder(): FliptEvaluationClientBuilder {
            return FliptEvaluationClientBuilder()
        }
    }
}
