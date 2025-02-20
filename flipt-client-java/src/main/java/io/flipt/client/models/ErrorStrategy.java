package io.flipt.client.models;

import com.fasterxml.jackson.annotation.JsonProperty;

public enum ErrorStrategy {
  @JsonProperty("fail")
  FAIL,
  @JsonProperty("fallback")
  FALLBACK
}
