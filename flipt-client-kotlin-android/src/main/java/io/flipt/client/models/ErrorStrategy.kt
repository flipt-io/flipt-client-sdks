package io.flipt.client.models
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
enum class ErrorStrategy {
    @SerialName("fail")
    FAIL,

    @SerialName("fallback")
    FALLBACK,
}
