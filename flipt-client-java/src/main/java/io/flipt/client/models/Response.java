package io.flipt.client.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.util.Optional;

public class Response {
  private final String type;

  private final Optional<VariantEvaluationResponse> variantEvaluationResponse;

  private final Optional<BooleanEvaluationResponse> booleanEvaluationResponse;

  private final Optional<ErrorEvaluationResponse> errorEvaluationResponse;

  public Response(
      @JsonProperty("type") String type,
      @JsonProperty("variant_evaluation_response")
          Optional<VariantEvaluationResponse> variantEvaluationResponse,
      @JsonProperty("boolean_evaluation_response")
          Optional<BooleanEvaluationResponse> booleanEvaluationResponse,
      @JsonProperty("error_evaluation_response")
          Optional<ErrorEvaluationResponse> errorEvaluationResponse) {
    this.type = type;
    this.variantEvaluationResponse = variantEvaluationResponse;
    this.booleanEvaluationResponse = booleanEvaluationResponse;
    this.errorEvaluationResponse = errorEvaluationResponse;
  }

  @JsonProperty("type")
  public String getType() {
    return type;
  }

  @JsonProperty("variant_evaluation_response")
  public Optional<VariantEvaluationResponse> getVariantEvaluationResponse() {
    return variantEvaluationResponse;
  }

  @JsonProperty("boolean_evaluation_response")
  public Optional<BooleanEvaluationResponse> getBooleanEvaluationResponse() {
    return booleanEvaluationResponse;
  }

  @JsonProperty("error_evaluation_response")
  public Optional<ErrorEvaluationResponse> getErrorEvaluationResponse() {
    return errorEvaluationResponse;
  }
}
