package io.flipt.client.models;

import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.util.Optional;

@JsonInclude(JsonInclude.Include.NON_EMPTY)
public class EngineOpts {
  private final Optional<String> url;
  private final Optional<Integer> updateInterval;
  private final Optional<AuthenticationStrategy> authentication;
  private final Optional<String> reference;

  public EngineOpts(
      Optional<String> url,
      Optional<Integer> updateInterval,
      Optional<AuthenticationStrategy> authentication,
      Optional<String> reference) {
    this.url = url;
    this.updateInterval = updateInterval;
    this.authentication = authentication;
    this.reference = reference;
  }

  @JsonProperty("url")
  public Optional<String> getUrl() {
    return url;
  }

  @JsonProperty("update_interval")
  public Optional<Integer> getUpdateInterval() {
    return updateInterval;
  }

  @JsonProperty("authentication")
  public Optional<AuthenticationStrategy> getAuthentication() {
    return authentication;
  }

  @JsonProperty("reference")
  public Optional<String> getReference() {
    return reference;
  }
}
