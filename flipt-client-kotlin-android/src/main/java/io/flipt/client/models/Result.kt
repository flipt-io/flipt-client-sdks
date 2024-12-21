package io.flipt.client.models

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class Result<T>(
    @SerialName("status") val status: String,
    @SerialName("result") val result: T?,
    @SerialName("error_message") val errorMessage: String?
)