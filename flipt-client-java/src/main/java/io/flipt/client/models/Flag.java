package io.flipt.client.models;

import com.fasterxml.jackson.annotation.JsonCreator;
import com.fasterxml.jackson.annotation.JsonProperty;

public class Flag {
  private final String key;
  private final boolean enabled;

  @JsonCreator(mode = JsonCreator.Mode.PROPERTIES)
  public Flag(@JsonProperty("key") String key, @JsonProperty("enabled") boolean enabled) {
    this.key = key;
    this.enabled = enabled;
  }

  public String getKey() {
    return key;
  }

  public boolean isEnabled() {
    return enabled;
  }
}
