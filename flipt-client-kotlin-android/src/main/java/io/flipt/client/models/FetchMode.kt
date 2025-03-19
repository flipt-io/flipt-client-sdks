package io.flipt.client.models
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
enum class FetchMode {
    @SerialName("polling")
    POLLING,

    @SerialName("streaming")
    STREAMING
}