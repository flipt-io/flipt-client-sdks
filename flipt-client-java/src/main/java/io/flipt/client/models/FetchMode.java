package io.flipt.client.models;

import com.fasterxml.jackson.annotation.JsonProperty;

public enum FetchMode {
    @JsonProperty("polling")
    POLLING,
    @JsonProperty("streaming")
    STREAMING
}