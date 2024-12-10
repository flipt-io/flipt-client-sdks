package io.flipt.client.models

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class VariantEvaluationResponse(
    @SerialName("match")val match: Boolean? = null,
    @SerialName("segment_keys")val segmentKeys: List<String>? = null,
    @SerialName("reason")val reason: String? = null,
    @SerialName("flag_key")val flagKey: String? = null,
    @SerialName("variant_key")val variantKey: String? = null,
    @SerialName("variant_attachment")val variantAttachment: String? = null,
    @SerialName("request_duration_millis")val requestDurationMillis: Float? = null,
    @SerialName("timestamp")val timestamp: String? = null
)