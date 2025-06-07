package io.flipt.client.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.databind.annotation.JsonDeserialize;
import java.util.Optional;
import lombok.Builder;
import lombok.Value;

@Value
@Builder
@JsonDeserialize(builder = Response.ResponseBuilder.class)
public class Response {
  @JsonProperty("type")
  String type;

  @JsonProperty("variant_evaluation_response")
  Optional<VariantEvaluationResponse> variantEvaluationResponse;

  @JsonProperty("boolean_evaluation_response")
  Optional<BooleanEvaluationResponse> booleanEvaluationResponse;

  @JsonProperty("error_evaluation_response")
  Optional<ErrorEvaluationResponse> errorEvaluationResponse;

  @lombok.experimental.SuperBuilder
  public static class ResponseBuilder {}
}
