package io.flipt.client.models;

import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.time.Duration;
import java.util.Optional;

@JsonInclude(JsonInclude.Include.NON_EMPTY)
public class ClientOptions {
  private final Optional<String> environment;
  private final Optional<String> namespace;
  private final Optional<String> url;
  private final Optional<Long> requestTimeout;
  private final Optional<Long> updateInterval;
  private final Optional<AuthenticationStrategy> authentication;
  private final Optional<String> reference;
  private final Optional<FetchMode> fetchMode;
  private final Optional<ErrorStrategy> errorStrategy;
  private final Optional<String> snapshot;
  private final Optional<TlsConfig> tlsConfig;

  public ClientOptions(
      Optional<String> environment,
      Optional<String> namespace,
      Optional<String> url,
      Optional<Duration> requestTimeout,
      Optional<Duration> updateInterval,
      Optional<AuthenticationStrategy> authentication,
      Optional<String> reference,
      Optional<FetchMode> fetchMode,
      Optional<ErrorStrategy> errorStrategy,
      Optional<String> snapshot,
      Optional<TlsConfig> tlsConfig) {
    this.environment = environment;
    this.namespace = namespace;
    this.url = url;
    this.authentication = authentication;
    this.reference = reference;

    Optional<Long> setRequestTimeout = Optional.empty();
    if (requestTimeout.isPresent()) {
      setRequestTimeout = Optional.of(requestTimeout.get().getSeconds());
    }

    Optional<Long> setUpdateInterval = Optional.empty();
    if (updateInterval.isPresent()) {
      setUpdateInterval = Optional.of(updateInterval.get().getSeconds());
    }

    this.requestTimeout = setRequestTimeout;
    this.updateInterval = setUpdateInterval;
    this.fetchMode = fetchMode;
    this.errorStrategy = errorStrategy;
    this.snapshot = snapshot;
    this.tlsConfig = tlsConfig;
  }

  @JsonProperty("environment")
  public Optional<String> getEnvironment() {
    return environment;
  }

  @JsonProperty("namespace")
  public Optional<String> getNamespace() {
    return namespace;
  }

  @JsonProperty("url")
  public Optional<String> getUrl() {
    return url;
  }

  @JsonProperty("request_timeout")
  public Optional<Long> getRequestTimeout() {
    return requestTimeout;
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

  @JsonProperty("error_strategy")
  public Optional<ErrorStrategy> getErrorStrategy() {
    return errorStrategy;
  }

  @JsonProperty("snapshot")
  public Optional<String> getSnapshot() {
    return snapshot;
  }

  @JsonProperty("tls_config")
  public Optional<TlsConfig> getTlsConfig() {
    return tlsConfig;
  }
}
