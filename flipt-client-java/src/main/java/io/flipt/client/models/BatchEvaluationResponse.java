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
@JsonDeserialize(builder = BatchEvaluationResponse.BatchEvaluationResponseBuilder.class)
@JsonIgnoreProperties(ignoreUnknown = true)
public class BatchEvaluationResponse {
  @JsonProperty("responses")
  @Singular
  List<Response> responses;

  @JsonProperty("request_duration_millis")
  float requestDurationMillis;

  @lombok.experimental.SuperBuilder
  @JsonIgnoreProperties(ignoreUnknown = true)
  public static class BatchEvaluationResponseBuilder {}
}
