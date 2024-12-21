package io.flipt.client.models

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class BatchEvaluationResponse(
    @SerialName("responses") val responses: Array<Response>,
    @SerialName("request_duration_millis") val requestDurationMillis: Float
)