package io.flipt.client.models;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.databind.annotation.JsonDeserialize;
import lombok.Builder;
import lombok.Value;

@Value
@Builder
@JsonIgnoreProperties(ignoreUnknown = true)
@JsonDeserialize(builder = Flag.FlagBuilder.class)
public class Flag {
  @JsonProperty("key")
  String key;

  @JsonProperty("enabled")
  boolean enabled;

  @JsonProperty("type")
  String type;

  @lombok.experimental.SuperBuilder
  @JsonIgnoreProperties(ignoreUnknown = true)
  public static class FlagBuilder {}
}
