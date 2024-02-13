package io.flipt.client.models;

import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.annotation.JsonProperty;

@JsonInclude(JsonInclude.Include.NON_EMPTY)
public class ErrorEvaluationResponse {
  private final String flagKey;

  private final String namespaceKey;

  private final String reason;

  public ErrorEvaluationResponse(
      @JsonProperty("flag_key") String flagKey,
      @JsonProperty("namespace_key") String namespaceKey,
      @JsonProperty("reason") String reason) {
    this.flagKey = flagKey;
    this.namespaceKey = namespaceKey;
    this.reason = reason;
  }

  @JsonProperty("flag_key")
  public String getFlagKey() {
    return flagKey;
  }

  @JsonProperty("namespace_key")
  public String getNamespaceKey() {
    return namespaceKey;
  }

  @JsonProperty("reason")
  public String getReason() {
    return reason;
  }
}
