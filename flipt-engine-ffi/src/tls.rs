use crate::Error;
use serde::Deserialize;

#[derive(Deserialize, Debug, PartialEq, Default)]
pub struct TlsConfig {
    /// Path to custom CA certificate file (PEM format)
    ca_cert_file: Option<String>,
    /// Raw CA certificate content (PEM format)
    ca_cert_data: Option<String>,
    /// Skip certificate verification (insecure - for development only)
    insecure_skip_verify: Option<bool>,
    /// Skip hostname verification while maintaining certificate validation (insecure - for development only)
    insecure_skip_hostname_verify: Option<bool>,
    /// Client certificate file for mutual TLS (PEM format)
    client_cert_file: Option<String>,
    /// Client key file for mutual TLS (PEM format)
    client_key_file: Option<String>,
    /// Raw client certificate content (PEM format)
    client_cert_data: Option<String>,
    /// Raw client key content (PEM format)
    client_key_data: Option<String>,
}

impl TlsConfig {
    /// Create a TLS config that skips all certificate verification (insecure)
    pub fn insecure() -> Self {
        Self {
            insecure_skip_verify: Some(true),
            ..Default::default()
        }
    }

    /// Create a TLS config that skips hostname verification only
    pub fn skip_hostname_verify() -> Self {
        Self {
            insecure_skip_hostname_verify: Some(true),
            ..Default::default()
        }
    }
}

/// Configures TLS settings for an HTTP client.
///
/// This function handles various TLS configuration scenarios:
/// - Insecure mode (skip all verification)
/// - Hostname verification skip (keep certificate validation)
/// - Custom CA certificates
/// - Client certificates for mutual TLS
///
/// # Arguments
/// * `builder` - The HTTP client builder to configure
/// * `tls_config` - TLS configuration options
///
/// # Returns
/// * `Ok(ClientBuilder)` - Configured client builder
/// * `Err(Error)` - Configuration error
pub fn configure_tls(
    mut builder: reqwest::ClientBuilder,
    tls_config: &TlsConfig,
) -> Result<reqwest::ClientBuilder, Error> {
    // Always use rustls for consistency
    builder = builder.use_rustls_tls();

    // Handle insecure mode (skip all validation)
    if tls_config.insecure_skip_verify.unwrap_or(false) {
        builder = builder.danger_accept_invalid_certs(true);
        return Ok(builder);
    }

    // Handle custom CA certificates using reqwest's built-in method
    let ca_cert_data = load_ca_certificate_data(tls_config)?;
    if let Some(cert_bytes) = ca_cert_data {
        let cert = reqwest::Certificate::from_pem(&cert_bytes)
            .map_err(|e| Error::Internal(format!("failed to parse CA certificate: {e}")))?;
        builder = builder.add_root_certificate(cert);
    }

    // Handle hostname verification skip
    if tls_config.insecure_skip_hostname_verify.unwrap_or(false) {
        builder = builder.danger_accept_invalid_hostnames(true);
    }

    // Handle client certificates (if needed)
    let client_cert_data = load_client_certificate_data(tls_config)?;
    if let Some((cert_bytes, key_bytes)) = client_cert_data {
        // Combine cert and key for reqwest identity
        let mut pem_data = Vec::new();
        pem_data.extend_from_slice(&cert_bytes);
        pem_data.extend_from_slice(&key_bytes);

        let identity = reqwest::Identity::from_pem(&pem_data).map_err(|e| {
            Error::Internal(format!(
                "failed to create identity from client cert/key: {e}"
            ))
        })?;
        builder = builder.identity(identity);
    }

    Ok(builder)
}

/// Loads CA certificate data from file or direct data.
///
/// # Arguments
/// * `tls_config` - TLS configuration containing CA certificate options
///
/// # Returns
/// * `Ok(Some(Vec<u8>))` - Certificate data if provided
/// * `Ok(None)` - No CA certificate configured
/// * `Err(Error)` - File read error
#[allow(clippy::type_complexity)]
fn load_ca_certificate_data(tls_config: &TlsConfig) -> Result<Option<Vec<u8>>, Error> {
    if let Some(ca_cert_data) = &tls_config.ca_cert_data {
        Ok(Some(ca_cert_data.as_bytes().to_vec()))
    } else if let Some(ca_cert_file) = &tls_config.ca_cert_file {
        match std::fs::read(ca_cert_file) {
            Ok(data) => Ok(Some(data)),
            Err(e) => {
                log::error!("failed to read CA cert file {ca_cert_file}: {e}");
                Err(Error::Internal(format!(
                    "failed to read CA cert file {ca_cert_file}: {e}"
                )))
            }
        }
    } else {
        Ok(None)
    }
}

/// Loads client certificate and key data from files or direct data.
///
/// # Arguments
/// * `tls_config` - TLS configuration containing client certificate options
///
/// # Returns
/// * `Ok(Some((cert_bytes, key_bytes)))` - Certificate and key data if provided
/// * `Ok(None)` - No client certificates configured
/// * `Err(Error)` - File read error
#[allow(clippy::type_complexity)]
fn load_client_certificate_data(
    tls_config: &TlsConfig,
) -> Result<Option<(Vec<u8>, Vec<u8>)>, Error> {
    if let (Some(cert_data), Some(key_data)) =
        (&tls_config.client_cert_data, &tls_config.client_key_data)
    {
        Ok(Some((
            cert_data.as_bytes().to_vec(),
            key_data.as_bytes().to_vec(),
        )))
    } else if let (Some(cert_file), Some(key_file)) =
        (&tls_config.client_cert_file, &tls_config.client_key_file)
    {
        let cert_bytes = std::fs::read(cert_file).map_err(|e| {
            log::warn!("failed to read client cert file {cert_file}: {e}");
            Error::Internal(format!("failed to read client cert file: {e}"))
        })?;
        let key_bytes = std::fs::read(key_file).map_err(|e| {
            log::warn!("failed to read client key file {key_file}: {e}");
            Error::Internal(format!("failed to read client key file: {e}"))
        })?;
        Ok(Some((cert_bytes, key_bytes)))
    } else {
        Ok(None)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_tls_config_insecure_skip_verify() {
        let tls_config = TlsConfig::insecure();
        let builder = reqwest::Client::builder();
        let result = configure_tls(builder, &tls_config);
        assert!(result.is_ok());
    }

    #[test]
    fn test_tls_config_insecure_skip_hostname_verify() {
        let tls_config = TlsConfig::skip_hostname_verify();
        let builder = reqwest::Client::builder();
        let result = configure_tls(builder, &tls_config);
        assert!(result.is_ok());
    }

    #[test]
    fn test_tls_config_invalid_ca_cert_file() {
        let tls_config = TlsConfig {
            ca_cert_file: Some("nonexistent.crt".to_string()),
            ..Default::default()
        };

        let builder = reqwest::Client::builder();
        let result = configure_tls(builder, &tls_config);
        assert!(result.is_err());
        assert!(result
            .unwrap_err()
            .to_string()
            .contains("failed to read CA cert file"));
    }

    #[test]
    fn test_tls_config_combined_options() {
        let tls_config = TlsConfig {
            ca_cert_file: Some("src/testdata/localhost.crt".to_string()),
            insecure_skip_verify: Some(true),
            ..Default::default()
        };

        let builder = reqwest::Client::builder();
        let result = configure_tls(builder, &tls_config);
        // insecure_skip_verify should take precedence
        assert!(result.is_ok());
    }

    #[test]
    fn test_tls_config_empty() {
        let tls_config = TlsConfig::default();

        let builder = reqwest::Client::builder();
        let result = configure_tls(builder, &tls_config);
        assert!(result.is_ok());
    }

    #[tokio::test]
    async fn test_tls_self_signed_certificate_integration() {
        use crate::http::HTTPFetcherBuilder;

        // Test that we can build HTTP fetchers with various TLS configurations
        // without actually connecting (which would require a real HTTPS server)

        // Test 1: With insecure_skip_verify
        let tls_config_insecure = TlsConfig::insecure();
        let fetcher_result = HTTPFetcherBuilder::new("https://localhost:8443")
            .tls_config(tls_config_insecure)
            .build();
        assert!(fetcher_result.is_ok());

        // Test 2: With hostname verification skip
        let tls_config_hostname_skip = TlsConfig::skip_hostname_verify();
        let fetcher_hostname_result = HTTPFetcherBuilder::new("https://localhost:8443")
            .tls_config(tls_config_hostname_skip)
            .build();
        assert!(fetcher_hostname_result.is_ok());

        // Test 3: Combined CA certificate with hostname skip
        let cert_pem = include_str!("testdata/localhost.crt");
        let tls_config_combined = TlsConfig {
            ca_cert_data: Some(cert_pem.to_string()),
            insecure_skip_hostname_verify: Some(true),
            ..Default::default()
        };

        let fetcher_combined_result = HTTPFetcherBuilder::new("https://localhost:8443")
            .tls_config(tls_config_combined)
            .build();

        assert!(fetcher_combined_result.is_ok());
    }
}
