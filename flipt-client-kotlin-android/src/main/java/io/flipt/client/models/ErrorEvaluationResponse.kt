package io.flipt.client.models

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
class ErrorEvaluationResponse(
    @SerialName("flag_key") val flagKey: String,
    @SerialName("namespace_key") val namespaceKey: String,
    @SerialName("reason") val reason: String)