package io.flipt.client.models

import kotlinx.serialization.ExperimentalSerializationApi
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonClassDiscriminator

@OptIn(ExperimentalSerializationApi::class)
@Serializable
@JsonClassDiscriminator("type")
sealed class AuthenticationStrategy

@Serializable
@SerialName("client_token")
data class ClientTokenAuthentication(
    @SerialName("client_token")
    val clientToken: String
) : AuthenticationStrategy()

@Serializable
@SerialName("jwt_token")
data class JWTAuthentication(
    @SerialName("jwt_token")
    val jwtToken: String
) : AuthenticationStrategy()
