package io.flipt.client.models

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class BooleanEvaluationResponse(
    @SerialName("enabled") val enabled: Boolean,
    @SerialName("flag_key") val flagKey: String,
    @SerialName("reason") val reason: String,
    @SerialName("request_duration_millis") val requestDurationMillis: Float,
    @SerialName("timestamp") val timestamp: String
)