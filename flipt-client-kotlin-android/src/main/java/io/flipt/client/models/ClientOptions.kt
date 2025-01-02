package io.flipt.client.models

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlin.time.Duration

@Serializable
data class ClientOptions(
    @SerialName("url")private val url: String?,
    @SerialName("update_interval")private val updateInterval: Long?,
    @SerialName("authentication")private val authentication: AuthenticationStrategy?,
    @SerialName("reference")private val reference: String?,
    @SerialName("fetch_mode")private val fetchMode: FetchMode?
)