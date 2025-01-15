package io.flipt.client.models

import kotlinx.serialization.ExperimentalSerializationApi
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonClassDiscriminator

@OptIn(ExperimentalSerializationApi::class)
@Serializable
sealed class AuthenticationStrategy

@Serializable
data class ClientTokenAuthentication(
    @SerialName("client_token")
    val clientToken: String
) : AuthenticationStrategy()

@Serializable
data class JWTAuthentication(
    @SerialName("jwt_token")
    val jwtToken: String
) : AuthenticationStrategy()
