package io.flipt.client.models;

import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.databind.annotation.JsonDeserialize;
import java.util.List;
import lombok.Builder;
import lombok.Singular;
import lombok.Value;
import com.fasterxml.jackson.annotation.JsonIgnoreProperties;

@Value
@Builder
@JsonInclude(JsonInclude.Include.NON_EMPTY)
@JsonDeserialize(builder = VariantEvaluationResponse.VariantEvaluationResponseBuilder.class)
@JsonIgnoreProperties(ignoreUnknown = true)
public class VariantEvaluationResponse {
  @JsonProperty("match")
  boolean match;

  @JsonProperty("segment_keys")
  @Singular
  List<String> segmentKeys;

  @JsonProperty("reason")
  String reason;

  @JsonProperty("flag_key")
  String flagKey;

  @JsonProperty("variant_key")
  String variantKey;

  @JsonProperty("variant_attachment")
  String variantAttachment;

  @JsonProperty("request_duration_millis")
  float requestDurationMillis;

  @JsonProperty("timestamp")
  String timestamp;

  @lombok.experimental.SuperBuilder
  @JsonIgnoreProperties(ignoreUnknown = true)
  public static class VariantEvaluationResponseBuilder {}
}
