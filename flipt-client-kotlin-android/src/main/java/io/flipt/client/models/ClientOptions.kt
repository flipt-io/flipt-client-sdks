package io.flipt.client.models

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class ClientOptions(
    @SerialName("url")private val url: String?,
    @SerialName("request_timeout")private val requestTimeout: Long?,
    @SerialName("update_interval")private val updateInterval: Long?,
    @SerialName("authentication")private val authentication: AuthenticationStrategy?,
    @SerialName("reference")private val reference: String?,
    @SerialName("fetch_mode")private val fetchMode: FetchMode?,
    @SerialName("error_strategy")private val errorStrategy: ErrorStrategy?,
)
