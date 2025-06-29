using System.Text.Json.Serialization;

namespace FliptClient.Models
{
    /// <summary>
    /// Configuration for TLS connections to Flipt servers.
    /// Provides options for custom CA certificates, client certificates for mutual TLS,
    /// and insecure mode for development.
    /// </summary>
    public class TlsConfig
    {
        /// <summary>
        /// Gets or sets the path to the CA certificate file (PEM format).
        /// </summary>
        [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
        [JsonPropertyName("ca_cert_file")]
        public string? CaCertFile { get; set; }

        /// <summary>
        /// Gets or sets the CA certificate content (PEM format).
        /// Takes precedence over CaCertFile if both are provided.
        /// </summary>
        [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
        [JsonPropertyName("ca_cert_data")]
        public string? CaCertData { get; set; }

        /// <summary>
        /// Gets or sets whether to skip certificate verification.
        /// WARNING: Only use this in development environments!
        /// </summary>
        [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
        [JsonPropertyName("insecure_skip_verify")]
        public bool? InsecureSkipVerify { get; set; }

        /// <summary>
        /// Gets or sets the path to the client certificate file for mutual TLS (PEM format).
        /// </summary>
        [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
        [JsonPropertyName("client_cert_file")]
        public string? ClientCertFile { get; set; }

        /// <summary>
        /// Gets or sets the path to the client private key file for mutual TLS (PEM format).
        /// </summary>
        [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
        [JsonPropertyName("client_key_file")]
        public string? ClientKeyFile { get; set; }

        /// <summary>
        /// Gets or sets the client certificate content for mutual TLS (PEM format).
        /// Takes precedence over ClientCertFile if both are provided.
        /// </summary>
        [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
        [JsonPropertyName("client_cert_data")]
        public string? ClientCertData { get; set; }

        /// <summary>
        /// Gets or sets the client private key content for mutual TLS (PEM format).
        /// Takes precedence over ClientKeyFile if both are provided.
        /// </summary>
        [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
        [JsonPropertyName("client_key_data")]
        public string? ClientKeyData { get; set; }

        /// <summary>
        /// Initializes a new instance of the <see cref="TlsConfig"/> class.
        /// </summary>
        public TlsConfig()
        {
        }

        /// <summary>
        /// Creates a TLS configuration for insecure connections (development only).
        /// WARNING: Only use this in development environments!
        /// </summary>
        /// <returns>A TLS configuration with certificate verification disabled.</returns>
        public static TlsConfig Insecure()
        {
            return new TlsConfig
            {
                InsecureSkipVerify = true
            };
        }

        /// <summary>
        /// Creates a TLS configuration with a custom CA certificate file.
        /// </summary>
        /// <param name="caCertFile">Path to the CA certificate file (PEM format).</param>
        /// <returns>A TLS configuration with the specified CA certificate.</returns>
        /// <exception cref="ValidationException">Thrown when the CA certificate file does not exist.</exception>
        public static TlsConfig WithCaCertFile(string caCertFile)
        {
            if (string.IsNullOrWhiteSpace(caCertFile))
            {
                throw new ValidationException("CA certificate file path cannot be null or empty");
            }

            if (!File.Exists(caCertFile))
            {
                throw new ValidationException($"CA certificate file does not exist: {caCertFile}");
            }

            return new TlsConfig
            {
                CaCertFile = caCertFile
            };
        }

        /// <summary>
        /// Creates a TLS configuration with CA certificate data.
        /// </summary>
        /// <param name="caCertData">CA certificate content in PEM format.</param>
        /// <returns>A TLS configuration with the specified CA certificate data.</returns>
        /// <exception cref="ValidationException">Thrown when the CA certificate data is null or empty.</exception>
        public static TlsConfig WithCaCertData(string caCertData)
        {
            if (string.IsNullOrWhiteSpace(caCertData))
            {
                throw new ValidationException("CA certificate data cannot be null or empty");
            }

            return new TlsConfig
            {
                CaCertData = caCertData
            };
        }

        /// <summary>
        /// Creates a TLS configuration for mutual TLS using certificate files.
        /// </summary>
        /// <param name="clientCertFile">Path to the client certificate file (PEM format).</param>
        /// <param name="clientKeyFile">Path to the client private key file (PEM format).</param>
        /// <returns>A TLS configuration for mutual TLS.</returns>
        /// <exception cref="ValidationException">Thrown when certificate files are invalid or do not exist.</exception>
        public static TlsConfig WithMutualTls(string clientCertFile, string clientKeyFile)
        {
            if (string.IsNullOrWhiteSpace(clientCertFile))
            {
                throw new ValidationException("Client certificate file path cannot be null or empty");
            }

            if (string.IsNullOrWhiteSpace(clientKeyFile))
            {
                throw new ValidationException("Client key file path cannot be null or empty");
            }

            if (!File.Exists(clientCertFile))
            {
                throw new ValidationException($"Client certificate file does not exist: {clientCertFile}");
            }

            if (!File.Exists(clientKeyFile))
            {
                throw new ValidationException($"Client key file does not exist: {clientKeyFile}");
            }

            return new TlsConfig
            {
                ClientCertFile = clientCertFile,
                ClientKeyFile = clientKeyFile
            };
        }

        /// <summary>
        /// Creates a TLS configuration for mutual TLS using certificate data.
        /// </summary>
        /// <param name="clientCertData">Client certificate content in PEM format.</param>
        /// <param name="clientKeyData">Client private key content in PEM format.</param>
        /// <returns>A TLS configuration for mutual TLS.</returns>
        /// <exception cref="ValidationException">Thrown when certificate data is null or empty.</exception>
        public static TlsConfig WithMutualTlsData(string clientCertData, string clientKeyData)
        {
            if (string.IsNullOrWhiteSpace(clientCertData))
            {
                throw new ValidationException("Client certificate data cannot be null or empty");
            }

            if (string.IsNullOrWhiteSpace(clientKeyData))
            {
                throw new ValidationException("Client key data cannot be null or empty");
            }

            return new TlsConfig
            {
                ClientCertData = clientCertData,
                ClientKeyData = clientKeyData
            };
        }
    }
}