package io.flipt.client.models;

import com.fasterxml.jackson.annotation.JsonCreator;
import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.annotation.JsonProperty;

@JsonInclude(JsonInclude.Include.NON_EMPTY)
public class VariantEvaluationResponse {
  private final boolean match;

  private final String[] segmentKeys;

  private final String reason;

  private final String flagKey;

  private final String variantKey;

  private final String variantAttachment;

  private final float requestDurationMillis;

  private final String timestamp;

  @JsonCreator(mode = JsonCreator.Mode.PROPERTIES)
  public VariantEvaluationResponse(
      @JsonProperty("match") boolean match,
      @JsonProperty("segment_keys") String[] segmentKeys,
      @JsonProperty("reason") String reason,
      @JsonProperty("flag_key") String flagKey,
      @JsonProperty("variant_key") String variantKey,
      @JsonProperty("variant_attachment") String variantAttachment,
      @JsonProperty("request_duration_millis") float requestDurationMillis,
      @JsonProperty("timestamp") String timestamp) {
    this.match = match;
    this.segmentKeys = segmentKeys;
    this.reason = reason;
    this.flagKey = flagKey;
    this.variantKey = variantKey;
    this.variantAttachment = variantAttachment;
    this.requestDurationMillis = requestDurationMillis;
    this.timestamp = timestamp;
  }

  public boolean isMatch() {
    return match;
  }

  @JsonProperty("segment_keys")
  public String[] getSegmentKeys() {
    return segmentKeys;
  }

  public String getReason() {
    return reason;
  }

  @JsonProperty("flag_key")
  public String getFlagKey() {
    return flagKey;
  }

  @JsonProperty("variant_key")
  public String getVariantKey() {
    return variantKey;
  }

  @JsonProperty("variant_attachment")
  public String getVariantAttachment() {
    return variantAttachment;
  }

  @JsonProperty("request_duration_millis")
  public float getRequestDurationMillis() {
    return requestDurationMillis;
  }

  public String getTimestamp() {
    return timestamp;
  }
}
