package io.flipt.client.models;

import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.time.Duration;
import java.util.Optional;

@JsonInclude(JsonInclude.Include.NON_EMPTY)
public class ClientOptions {
  private final Optional<String> url;
  private final Optional<Long> updateInterval;
  private final Optional<AuthenticationStrategy> authentication;
  private final Optional<String> reference;
  private final Optional<FetchMode> fetchMode;
  public ClientOptions(
          Optional<String> url,
          Optional<Duration> updateInterval,
          Optional<AuthenticationStrategy> authentication,
          Optional<String> reference,
          Optional<FetchMode> fetchMode) {
    this.url = url;
    this.authentication = authentication;
    this.reference = reference;

    Optional<Long> setUpdateInterval = Optional.empty();
    if (updateInterval.isPresent()) {
      setUpdateInterval = Optional.of(updateInterval.get().getSeconds());
    }

    this.updateInterval = setUpdateInterval;
    this.fetchMode = fetchMode;
  }

  @JsonProperty("url")
  public Optional<String> getUrl() {
    return url;
  }

  @JsonProperty("update_interval")
  public Optional<Long> getUpdateInterval() {
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

  @JsonProperty("fetch_mode")
  public Optional<FetchMode> getFetchMode() {
    return fetchMode;
  }
}
