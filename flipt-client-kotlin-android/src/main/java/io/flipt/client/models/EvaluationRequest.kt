package io.flipt.client.models

import kotlinx.serialization.Serializable

@Serializable
data class EvaluationRequest(
    val flagKey: String,
    val entityId: String,
    val context: Map<String, String>
)