package io.flipt.client.models;

import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.nio.file.Files;
import java.nio.file.Paths;
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

  /**
   * @deprecated Use {@code builder().insecureSkipVerify(true)} instead. Creates a TlsConfig for
   *     development with insecure certificate verification disabled. <strong>WARNING:</strong> Only
   *     use this in development environments.
   * @return TlsConfig with insecure skip verify enabled
   */
  public static TlsConfig insecure() {
    return TlsConfig.builder().insecureSkipVerify(true).build();
  }

  /**
   * @deprecated Use {@code builder().caCertFile(caCertFile)} instead. Creates a TlsConfig with a
   *     custom CA certificate from a file.
   * @param caCertFile path to the CA certificate file in PEM format
   * @return TlsConfig with custom CA certificate
   * @throws IllegalArgumentException if the certificate file does not exist
   */
  public static TlsConfig withCaCertFile(String caCertFile) {
    if (caCertFile == null || caCertFile.trim().isEmpty()) {
      throw new IllegalArgumentException("CA certificate file path cannot be null or empty");
    }

    if (!Files.exists(Paths.get(caCertFile))) {
      throw new IllegalArgumentException("CA certificate file does not exist: " + caCertFile);
    }

    return TlsConfig.builder().caCertFile(caCertFile).build();
  }

  /**
   * @deprecated Use {@code builder().caCertData(caCertData)} instead. Creates a TlsConfig with a
   *     custom CA certificate from string data.
   * @param caCertData CA certificate content in PEM format
   * @return TlsConfig with custom CA certificate
   */
  public static TlsConfig withCaCertData(String caCertData) {
    return TlsConfig.builder().caCertData(caCertData).build();
  }

  /**
   * @deprecated Use {@code builder().clientCertFile(clientCertFile).clientKeyFile(clientKeyFile)}
   *     instead. Creates a TlsConfig for mutual TLS with client certificate and key files.
   * @param clientCertFile path to client certificate file in PEM format
   * @param clientKeyFile path to client private key file in PEM format
   * @return TlsConfig with mutual TLS configuration
   * @throws IllegalArgumentException if either certificate file or key file does not exist
   */
  public static TlsConfig withMutualTls(String clientCertFile, String clientKeyFile) {
    if (clientCertFile == null || clientCertFile.trim().isEmpty()) {
      throw new IllegalArgumentException("Client certificate file path cannot be null or empty");
    }

    if (clientKeyFile == null || clientKeyFile.trim().isEmpty()) {
      throw new IllegalArgumentException("Client key file path cannot be null or empty");
    }

    if (!Files.exists(Paths.get(clientCertFile))) {
      throw new IllegalArgumentException(
          "Client certificate file does not exist: " + clientCertFile);
    }

    if (!Files.exists(Paths.get(clientKeyFile))) {
      throw new IllegalArgumentException("Client key file does not exist: " + clientKeyFile);
    }

    return TlsConfig.builder().clientCertFile(clientCertFile).clientKeyFile(clientKeyFile).build();
  }

  /**
   * @deprecated Use {@code builder().clientCertData(clientCertData).clientKeyData(clientKeyData)}
   *     instead. Creates a TlsConfig for mutual TLS with client certificate and key data.
   * @param clientCertData client certificate content in PEM format
   * @param clientKeyData client private key content in PEM format
   * @return TlsConfig with mutual TLS configuration
   */
  public static TlsConfig withMutualTlsData(String clientCertData, String clientKeyData) {
    return TlsConfig.builder().clientCertData(clientCertData).clientKeyData(clientKeyData).build();
  }
}
