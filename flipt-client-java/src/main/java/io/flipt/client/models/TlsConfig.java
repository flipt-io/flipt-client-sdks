package io.flipt.client.models;

import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.Builder;
import lombok.Value;

import java.util.Optional;

/**
 * TLS configuration for connecting to Flipt servers with custom certificates.
 * 
 * <p>This class provides comprehensive TLS configuration options including:
 * <ul>
 * <li>Custom CA certificates for self-signed or private CAs</li>
 * <li>Client certificates for mutual TLS authentication</li>
 * <li>Certificate data as strings or file paths</li>
 * <li>Insecure mode for development (skip certificate verification)</li>
 * </ul>
 * 
 * <p>Certificate data fields take precedence over file path fields when both are provided.
 */
@Value
@Builder
@JsonInclude(JsonInclude.Include.NON_EMPTY)
public class TlsConfig {

    /**
     * Path to custom CA certificate file in PEM format.
     * Used to verify server certificates signed by custom or self-signed CAs.
     */
    @JsonProperty("ca_cert_file")
    Optional<String> caCertFile;

    /**
     * Raw CA certificate content in PEM format.
     * Used to verify server certificates signed by custom or self-signed CAs.
     * Takes precedence over caCertFile when both are provided.
     */
    @JsonProperty("ca_cert_data")
    Optional<String> caCertData;

    /**
     * Skip certificate verification entirely.
     * <strong>WARNING:</strong> This should only be used in development environments.
     * Setting this to true makes connections vulnerable to man-in-the-middle attacks.
     */
    @JsonProperty("insecure_skip_verify")
    Optional<Boolean> insecureSkipVerify;

    /**
     * Path to client certificate file in PEM format.
     * Used for mutual TLS authentication where the server requires client certificates.
     */
    @JsonProperty("client_cert_file")
    Optional<String> clientCertFile;

    /**
     * Path to client private key file in PEM format.
     * Used for mutual TLS authentication where the server requires client certificates.
     * Must correspond to the clientCertFile.
     */
    @JsonProperty("client_key_file")
    Optional<String> clientKeyFile;

    /**
     * Raw client certificate content in PEM format.
     * Used for mutual TLS authentication where the server requires client certificates.
     * Takes precedence over clientCertFile when both are provided.
     */
    @JsonProperty("client_cert_data")
    Optional<String> clientCertData;

    /**
     * Raw client private key content in PEM format.
     * Used for mutual TLS authentication where the server requires client certificates.
     * Must correspond to the clientCertData.
     * Takes precedence over clientKeyFile when both are provided.
     */
    @JsonProperty("client_key_data")
    Optional<String> clientKeyData;

    /**
     * Creates a TlsConfig for development with insecure certificate verification disabled.
     * <strong>WARNING:</strong> Only use this in development environments.
     * 
     * @return TlsConfig with insecure skip verify enabled
     */
    public static TlsConfig insecure() {
        return TlsConfig.builder()
                .insecureSkipVerify(Optional.of(true))
                .build();
    }

    /**
     * Creates a TlsConfig with a custom CA certificate from a file.
     * 
     * @param caCertFile path to the CA certificate file in PEM format
     * @return TlsConfig with custom CA certificate
     */
    public static TlsConfig withCaCertFile(String caCertFile) {
        return TlsConfig.builder()
                .caCertFile(Optional.of(caCertFile))
                .build();
    }

    /**
     * Creates a TlsConfig with a custom CA certificate from string data.
     * 
     * @param caCertData CA certificate content in PEM format
     * @return TlsConfig with custom CA certificate
     */
    public static TlsConfig withCaCertData(String caCertData) {
        return TlsConfig.builder()
                .caCertData(Optional.of(caCertData))
                .build();
    }

    /**
     * Creates a TlsConfig for mutual TLS with client certificate and key files.
     * 
     * @param clientCertFile path to client certificate file in PEM format
     * @param clientKeyFile path to client private key file in PEM format
     * @return TlsConfig with mutual TLS configuration
     */
    public static TlsConfig withMutualTls(String clientCertFile, String clientKeyFile) {
        return TlsConfig.builder()
                .clientCertFile(Optional.of(clientCertFile))
                .clientKeyFile(Optional.of(clientKeyFile))
                .build();
    }

    /**
     * Creates a TlsConfig for mutual TLS with client certificate and key data.
     * 
     * @param clientCertData client certificate content in PEM format
     * @param clientKeyData client private key content in PEM format
     * @return TlsConfig with mutual TLS configuration
     */
    public static TlsConfig withMutualTlsData(String clientCertData, String clientKeyData) {
        return TlsConfig.builder()
                .clientCertData(Optional.of(clientCertData))
                .clientKeyData(Optional.of(clientKeyData))
                .build();
    }
}