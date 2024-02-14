package io.flipt.client.models;

import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.annotation.JsonProperty;

@JsonInclude(JsonInclude.Include.NON_EMPTY)
public class BooleanEvaluationResponse {
  private final boolean enabled;

  private final String flagKey;

  private final String reason;

  private final float requestDurationMillis;

  private final String timestamp;

  public BooleanEvaluationResponse(
      @JsonProperty("enabled") boolean enabled,
      @JsonProperty("flag_key") String flagKey,
      @JsonProperty("reason") String reason,
      @JsonProperty("request_duration_millis") float requestDurationMillis,
      @JsonProperty("timestamp") String timestamp) {
    this.enabled = enabled;
    this.flagKey = flagKey;
    this.requestDurationMillis = requestDurationMillis;
    this.reason = reason;
    this.timestamp = timestamp;
  }

  @JsonProperty("enabled")
  public boolean isEnabled() {
    return enabled;
  }

  @JsonProperty("flag_key")
  public String getFlagKey() {
    return flagKey;
  }

  @JsonProperty("reason")
  public String getReason() {
    return reason;
  }

  @JsonProperty("request_duration_millis")
  public float getRequestDurationMillis() {
    return requestDurationMillis;
  }

  @JsonProperty("timestamp")
  public String getTimestamp() {
    return timestamp;
  }
}
