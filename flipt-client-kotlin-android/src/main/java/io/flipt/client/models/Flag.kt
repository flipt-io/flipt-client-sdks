package io.flipt.client.models

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class Flag(
    @SerialName("key") val key: String,
    @SerialName("enabled") val enabled: Boolean,
    @SerialName("type") val type: String
)