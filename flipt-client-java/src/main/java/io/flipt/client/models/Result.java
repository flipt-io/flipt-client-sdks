package io.flipt.client.models;

import com.fasterxml.jackson.annotation.JsonCreator;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.util.Optional;

public class Result<T> {
  private final String status;

  private final Optional<T> result;

  private final Optional<String> errorMessage;

  @JsonCreator(mode = JsonCreator.Mode.PROPERTIES)
  public Result(
      @JsonProperty("status") String status,
      @JsonProperty("result") Optional<T> result,
      @JsonProperty("error_message") Optional<String> errorMessage) {
    this.status = status;
    this.result = result;
    this.errorMessage = errorMessage;
  }

  public String getStatus() {
    return status;
  }

  public Optional<T> getResult() {
    return result;
  }

  @JsonProperty("error_message")
  public Optional<String> getErrorMessage() {
    return errorMessage;
  }
}
