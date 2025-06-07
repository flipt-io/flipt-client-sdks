package io.flipt.client.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.databind.annotation.JsonDeserialize;
import java.util.Optional;
import lombok.Builder;
import lombok.Value;

@Value
@Builder
@JsonDeserialize(builder = Result.ResultBuilder.class)
public class Result<T> {
  @JsonProperty("status")
  String status;

  @JsonProperty("result")
  Optional<T> result;

  @JsonProperty("error_message")
  Optional<String> errorMessage;

  @lombok.experimental.SuperBuilder
  public static class ResultBuilder<T> {}
}
