package io.flipt.client.models;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.databind.annotation.JsonDeserialize;
import lombok.Builder;
import lombok.Value;

@Value
@Builder
@JsonInclude(JsonInclude.Include.NON_EMPTY)
@JsonDeserialize(builder = BooleanEvaluationResponse.BooleanEvaluationResponseBuilder.class)
public class BooleanEvaluationResponse {
  @JsonProperty("enabled")
  boolean enabled;

  @JsonProperty("flag_key")
  String flagKey;

  @JsonProperty("reason")
  String reason;

  @JsonProperty("request_duration_millis")
  float requestDurationMillis;

  @JsonProperty("timestamp")
  String timestamp;

  @lombok.experimental.SuperBuilder
  @JsonIgnoreProperties(ignoreUnknown = true)
  public static class BooleanEvaluationResponseBuilder {}
}
