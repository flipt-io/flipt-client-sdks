package io.flipt.client.models
import kotlinx.serialization.*
import kotlinx.serialization.json.*
import kotlinx.serialization.encoding.*
import kotlinx.serialization.descriptors.*

@OptIn(ExperimentalSerializationApi::class)
@Serializable(with = AuthenticationStrategySerializer::class)
sealed class AuthenticationStrategy

@Serializable
data class ClientTokenAuthentication(
    @SerialName("client_token")
    val clientToken: String,
) : AuthenticationStrategy()

@Serializable
data class JWTAuthentication(
    @SerialName("jwt_token")
    val jwtToken: String,
) : AuthenticationStrategy()

object AuthenticationStrategySerializer : KSerializer<AuthenticationStrategy> {
    override val descriptor: SerialDescriptor = buildClassSerialDescriptor("AuthenticationStrategy") {
        element<String>("type", isOptional = true) // Optional type for identification if needed
        element<String>("client_token", isOptional = true)
        element<String>("jwt_token", isOptional = true)
    }

    override fun serialize(encoder: Encoder, value: AuthenticationStrategy) {
        val jsonEncoder = encoder as? JsonEncoder
            ?: throw SerializationException("This serializer only supports JSON encoding")

        val jsonObject = when (value) {
            is ClientTokenAuthentication -> JsonObject(mapOf("client_token" to JsonPrimitive(value.clientToken)))
            is JWTAuthentication -> JsonObject(mapOf("jwt_token" to JsonPrimitive(value.jwtToken)))
        }
        jsonEncoder.encodeJsonElement(jsonObject)
    }

    override fun deserialize(decoder: Decoder): AuthenticationStrategy {
        val jsonDecoder = decoder as? JsonDecoder
            ?: throw SerializationException("This serializer only supports JSON decoding")

        val jsonElement = jsonDecoder.decodeJsonElement() as? JsonObject
            ?: throw SerializationException("Expected JSON object")

        return when {
            "client_token" in jsonElement -> ClientTokenAuthentication(
                clientToken = jsonElement["client_token"]!!.jsonPrimitive.content
            )
            "jwt_token" in jsonElement -> JWTAuthentication(
                jwtToken = jsonElement["jwt_token"]!!.jsonPrimitive.content
            )
            else -> throw SerializationException("Unknown authentication strategy")
        }
    }
}
