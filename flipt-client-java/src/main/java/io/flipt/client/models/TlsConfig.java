package io.flipt.client.models;

import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.util.Optional;
import lombok.Builder;

/**
 * TLS configuration for connecting to Flipt servers with custom certificates.
 *
 * <p>This class provides comprehensive TLS configuration options including:
 *
 * <ul>
 *   <li>Custom CA certificates for self-signed or private CAs
 *   <li>Client certificates for mutual TLS authentication
 *   <li>Certificate data as strings or file paths
 *   <li>Insecure mode for development (skip certificate verification)
 * </ul>
 *
 * <p>Certificate data fields take precedence over file path fields when both are provided.
 */
@JsonInclude(JsonInclude.Include.NON_EMPTY)
@Builder
public class TlsConfig {

  @Builder.Default private final String caCertFile = null;
  @Builder.Default private final String caCertData = null;
  @Builder.Default private final Boolean insecureSkipVerify = null;
  @Builder.Default private final Boolean insecureSkipHostnameVerify = null;
  @Builder.Default private final String clientCertFile = null;
  @Builder.Default private final String clientKeyFile = null;
  @Builder.Default private final String clientCertData = null;
  @Builder.Default private final String clientKeyData = null;

  /**
   * Path to custom CA certificate file in PEM format. Used to verify server certificates signed by
   * custom or self-signed CAs.
   */
  @JsonProperty("ca_cert_file")
  public Optional<String> getCaCertFile() {
    return Optional.ofNullable(caCertFile);
  }

  /**
   * Raw CA certificate content in PEM format. Used to verify server certificates signed by custom
   * or self-signed CAs. Takes precedence over caCertFile when both are provided.
   */
  @JsonProperty("ca_cert_data")
  public Optional<String> getCaCertData() {
    return Optional.ofNullable(caCertData);
  }

  /**
   * Skip certificate verification entirely. <strong>WARNING:</strong> This should only be used in
   * development environments. Setting this to true makes connections vulnerable to
   * man-in-the-middle attacks.
   */
  @JsonProperty("insecure_skip_verify")
  public Optional<Boolean> getInsecureSkipVerify() {
    return Optional.ofNullable(insecureSkipVerify);
  }

  /**
   * Skip hostname verification while maintaining certificate validation. <strong>WARNING:</strong>
   * This should only be used when you know what you are doing. This still validates the certificate
   * chain.
   */
  @JsonProperty("insecure_skip_hostname_verify")
  public Optional<Boolean> getInsecureSkipHostnameVerify() {
    return Optional.ofNullable(insecureSkipHostnameVerify);
  }

  /**
   * Path to client certificate file in PEM format. Used for mutual TLS authentication where the
   * server requires client certificates.
   */
  @JsonProperty("client_cert_file")
  public Optional<String> getClientCertFile() {
    return Optional.ofNullable(clientCertFile);
  }

  /**
   * Path to client private key file in PEM format. Used for mutual TLS authentication where the
   * server requires client certificates. Must correspond to the clientCertFile.
   */
  @JsonProperty("client_key_file")
  public Optional<String> getClientKeyFile() {
    return Optional.ofNullable(clientKeyFile);
  }

  /**
   * Raw client certificate content in PEM format. Used for mutual TLS authentication where the
   * server requires client certificates. Takes precedence over clientCertFile when both are
   * provided.
   */
  @JsonProperty("client_cert_data")
  public Optional<String> getClientCertData() {
    return Optional.ofNullable(clientCertData);
  }

  /**
   * Raw client private key content in PEM format. Used for mutual TLS authentication where the
   * server requires client certificates. Must correspond to the clientCertData. Takes precedence
   * over clientKeyFile when both are provided.
   */
  @JsonProperty("client_key_data")
  public Optional<String> getClientKeyData() {
    return Optional.ofNullable(clientKeyData);
  }
}
