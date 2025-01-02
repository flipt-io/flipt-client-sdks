package io.flipt.client.models

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class Response(
    @SerialName("type")
    val type: String,

    @SerialName("variant_evaluation_response")
    val variantEvaluationResponse: VariantEvaluationResponse?,

    @SerialName("boolean_evaluation_response")
    val booleanEvaluationResponse: BooleanEvaluationResponse?,

    @SerialName("error_evaluation_response")
    val errorEvaluationResponse: ErrorEvaluationResponse?
)