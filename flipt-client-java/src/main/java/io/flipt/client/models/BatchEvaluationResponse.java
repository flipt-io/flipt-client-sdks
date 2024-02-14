package io.flipt.client.models;

import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.annotation.JsonProperty;

@JsonInclude(JsonInclude.Include.NON_EMPTY)
public class BatchEvaluationResponse {
  private final Response[] responses;

  private final float requestDurationMillis;

  public BatchEvaluationResponse(
      @JsonProperty("responses") Response[] responses,
      @JsonProperty("request_duration_millis") float requestDurationMillis) {
    this.responses = responses;
    this.requestDurationMillis = requestDurationMillis;
  }

  @JsonProperty("responses")
  public Response[] getResponses() {
    return responses;
  }

  @JsonProperty("request_duration_millis")
  public float getRequestDurationMillis() {
    return requestDurationMillis;
  }
}
