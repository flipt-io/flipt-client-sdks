package io.flipt.client.models;

import com.fasterxml.jackson.annotation.JsonCreator;
import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;

@JsonIgnoreProperties(ignoreUnknown = true)
public class Flag {
  private final String key;
  private final boolean enabled;
  private final String type;

  @JsonCreator(mode = JsonCreator.Mode.PROPERTIES)
  public Flag(
      @JsonProperty("key") String key,
      @JsonProperty("enabled") boolean enabled,
      @JsonProperty("type") String type) {
    this.key = key;
    this.enabled = enabled;
    this.type = type;
  }

  public String getKey() {
    return key;
  }

  public boolean isEnabled() {
    return enabled;
  }

  public String getType() {
    return type;
  }
}
