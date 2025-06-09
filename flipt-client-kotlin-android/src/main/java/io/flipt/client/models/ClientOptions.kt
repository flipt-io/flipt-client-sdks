package io.flipt.client.models

import kotlinx.serialization.KSerializer
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.descriptors.PrimitiveKind
import kotlinx.serialization.descriptors.PrimitiveSerialDescriptor
import kotlinx.serialization.descriptors.SerialDescriptor
import kotlinx.serialization.encoding.Decoder
import kotlinx.serialization.encoding.Encoder
import kotlin.time.Duration
import kotlin.time.Duration.Companion.seconds

/**
 * Serializes a [Duration] as a number of seconds (Long) for JSON.
 */
@kotlinx.serialization.ExperimentalSerializationApi
object DurationSecondsSerializer : KSerializer<Duration?> {
    override val descriptor: SerialDescriptor = PrimitiveSerialDescriptor("DurationSeconds", PrimitiveKind.LONG)

    override fun serialize(
        encoder: Encoder,
        value: Duration?,
    ) {
        if (value == null) {
            encoder.encodeNull()
        } else {
            encoder.encodeLong(value.inWholeSeconds)
        }
    }

    override fun deserialize(decoder: Decoder): Duration? {
        return try {
            val seconds = decoder.decodeLong()
            seconds.seconds
        } catch (e: Exception) {
            null
        }
    }
}

/**
 * Configuration options for FliptClient.
 * @property environment The environment for the Flipt server (optional).
 * @property namespace The namespace for the Flipt server (optional).
 * @property url The URL for the Flipt server (optional).
 * @property requestTimeout The request timeout (optional, as Duration).
 * @property updateInterval The update interval (optional, as Duration).
 * @property authentication The authentication strategy (optional).
 * @property reference The reference for the Flipt server (optional).
 * @property fetchMode The fetch mode for the Flipt server (optional).
 * @property errorStrategy The error strategy (optional).
 * @property snapshot The initial snapshot for evaluation (optional).
 */
@Serializable
data class ClientOptions(
    @SerialName("environment") val environment: String? = null,
    @SerialName("namespace") val namespace: String? = null,
    @SerialName("url") val url: String? = null,
    @SerialName("request_timeout") @Serializable(with = DurationSecondsSerializer::class) val requestTimeout: Duration? = null,
    @SerialName("update_interval") @Serializable(with = DurationSecondsSerializer::class) val updateInterval: Duration? = null,
    @SerialName("authentication") val authentication: AuthenticationStrategy? = null,
    @SerialName("reference") val reference: String? = null,
    @SerialName("fetch_mode") val fetchMode: FetchMode? = null,
    @SerialName("error_strategy") val errorStrategy: ErrorStrategy? = null,
    @SerialName("snapshot") val snapshot: String? = null,
)
