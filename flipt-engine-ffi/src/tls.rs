use crate::{Error, TlsConfig};

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
    // Handle insecure mode (skip all validation)
    if tls_config.insecure_skip_verify.unwrap_or(false) {
        builder = builder.use_rustls_tls().danger_accept_invalid_certs(true);
        log::debug!("TLS configured with insecure_skip_verify");
        return Ok(builder);
    }

    // Handle custom certificates (CA and/or client certificates)
    let ca_cert_data = load_ca_certificate_data(tls_config)?;
    let client_cert_data = load_client_certificate_data(tls_config)?;

    if ca_cert_data.is_some() || client_cert_data.is_some() {
        // Ensure crypto provider is installed for rustls
        let _ = rustls::crypto::ring::default_provider().install_default();

        let root_store = create_root_store(ca_cert_data)?;
        let rustls_config = create_rustls_config(root_store, client_cert_data)?;

        builder = builder.use_preconfigured_tls(rustls_config);
        log::debug!("successfully configured custom TLS");
    } else {
        // Use default rustls configuration
        builder = builder.use_rustls_tls();
    }

    // Apply hostname verification skip AFTER setting up TLS configuration
    if tls_config.insecure_skip_hostname_verify.unwrap_or(false) {
        builder = builder.danger_accept_invalid_hostnames(true);
        log::debug!("hostname verification disabled");
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
        log::debug!("loading CA certificate from provided data");
        Ok(Some(ca_cert_data.as_bytes().to_vec()))
    } else if let Some(ca_cert_file) = &tls_config.ca_cert_file {
        log::debug!("loading CA certificate from file: {ca_cert_file}");
        std::fs::read(ca_cert_file)
            .map_err(|e| {
                log::warn!("failed to read CA cert file {ca_cert_file}: {e}");
                Error::Internal(format!("failed to read CA cert file: {e}"))
            })
            .map(Some)
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
        log::debug!("loading client certificate and key from provided data");
        Ok(Some((
            cert_data.as_bytes().to_vec(),
            key_data.as_bytes().to_vec(),
        )))
    } else if let (Some(cert_file), Some(key_file)) =
        (&tls_config.client_cert_file, &tls_config.client_key_file)
    {
        log::debug!("loading client certificate from files: cert={cert_file}, key={key_file}");
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

/// Creates a certificate root store from CA certificate data.
///
/// # Arguments
/// * `ca_cert_data` - Optional CA certificate data
///
/// # Returns
/// * `Ok(RootCertStore)` - Configured certificate store
/// * `Err(Error)` - Certificate parsing error
#[allow(clippy::type_complexity)]
fn create_root_store(ca_cert_data: Option<Vec<u8>>) -> Result<rustls::RootCertStore, Error> {
    use rustls_pemfile::certs;
    use std::io::Cursor;

    if let Some(cert_bytes) = ca_cert_data {
        let mut cert_reader = Cursor::new(&cert_bytes);
        let ca_certs: Vec<rustls::pki_types::CertificateDer> = certs(&mut cert_reader)
            .collect::<Result<Vec<_>, _>>()
            .map_err(|e| {
                log::warn!("failed to parse CA certificate: {e}");
                Error::Internal(format!("invalid CA certificate: {e}"))
            })?;

        if ca_certs.is_empty() {
            log::warn!("no CA certificates found in provided data");
            return Err(Error::Internal("no CA certificates found".to_string()));
        }

        // Create a custom certificate store with only our CA
        let mut root_store = rustls::RootCertStore::empty();
        for cert in ca_certs {
            root_store.add(cert).map_err(|e| {
                log::warn!("failed to add CA certificate to root store: {e}");
                Error::Internal(format!("failed to add CA certificate: {e}"))
            })?;
        }
        log::debug!("successfully configured custom CA certificate");
        Ok(root_store)
    } else {
        // Use default root certificates
        Ok(rustls::RootCertStore {
            roots: webpki_roots::TLS_SERVER_ROOTS.to_vec(),
        })
    }
}

/// Creates a rustls client configuration with optional client certificates.
///
/// # Arguments
/// * `root_store` - Certificate root store
/// * `client_cert_data` - Optional client certificate and key data
///
/// # Returns
/// * `Ok(ClientConfig)` - Configured rustls client config
/// * `Err(Error)` - Configuration error
#[allow(clippy::type_complexity)]
fn create_rustls_config(
    root_store: rustls::RootCertStore,
    client_cert_data: Option<(Vec<u8>, Vec<u8>)>,
) -> Result<rustls::ClientConfig, Error> {
    use rustls_pemfile::{certs, private_key};
    use std::io::Cursor;

    if let Some((cert_bytes, key_bytes)) = client_cert_data {
        let mut cert_reader = Cursor::new(&cert_bytes);
        let mut key_reader = Cursor::new(&key_bytes);

        let client_certs: Vec<rustls::pki_types::CertificateDer> = certs(&mut cert_reader)
            .collect::<Result<Vec<_>, _>>()
            .map_err(|e| {
                log::warn!("failed to parse client certificate: {e}");
                Error::Internal(format!("invalid client certificate: {e}"))
            })?;

        let client_key = private_key(&mut key_reader)
            .map_err(|e| {
                log::warn!("failed to parse client key: {e}");
                Error::Internal(format!("invalid client key: {e}"))
            })?
            .ok_or_else(|| {
                log::warn!("no client key found");
                Error::Internal("no client key found".to_string())
            })?;

        rustls::ClientConfig::builder()
            .with_root_certificates(root_store)
            .with_client_auth_cert(client_certs, client_key)
            .map_err(|e| {
                log::warn!("failed to create client config with client auth: {e}");
                Error::Internal(format!("failed to create client config: {e}"))
            })
    } else {
        Ok(rustls::ClientConfig::builder()
            .with_root_certificates(root_store)
            .with_no_client_auth())
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
    fn test_tls_config_hostname_skip_with_custom_ca() {
        // Install crypto provider for rustls
        let _ = rustls::crypto::ring::default_provider().install_default();

        // Use the existing localhost.crt for testing
        let cert_pem = include_str!("testdata/localhost.crt");
        let tls_config = TlsConfig {
            ca_cert_data: Some(cert_pem.to_string()),
            insecure_skip_hostname_verify: Some(true),
            ..Default::default()
        };

        let builder = reqwest::Client::builder();
        let result = configure_tls(builder, &tls_config);
        assert!(result.is_ok());
    }

    #[test]
    fn test_tls_config_custom_ca_cert_data() {
        // Install crypto provider for rustls
        let _ = rustls::crypto::ring::default_provider().install_default();

        // Use the existing localhost.crt for testing
        let cert_pem = include_str!("testdata/localhost.crt");

        let tls_config = TlsConfig::with_ca_data(cert_pem);

        let builder = reqwest::Client::builder();
        let result = configure_tls(builder, &tls_config);
        assert!(result.is_ok());
    }

    #[test]
    fn test_tls_config_custom_ca_cert_file() {
        // Install crypto provider for rustls
        let _ = rustls::crypto::ring::default_provider().install_default();

        let tls_config = TlsConfig::with_ca_file("src/testdata/localhost.crt");

        let builder = reqwest::Client::builder();
        let result = configure_tls(builder, &tls_config);
        assert!(result.is_ok());
    }

    #[test]
    fn test_tls_config_client_certificates_data() {
        // Install crypto provider for rustls
        let _ = rustls::crypto::ring::default_provider().install_default();

        let cert_pem = include_str!("testdata/localhost.crt");
        let key_pem = include_str!("testdata/localhost.key");

        let tls_config = TlsConfig {
            client_cert_data: Some(cert_pem.to_string()),
            client_key_data: Some(key_pem.to_string()),
            ..Default::default()
        };

        let builder = reqwest::Client::builder();
        let result = configure_tls(builder, &tls_config);
        assert!(result.is_ok());
    }

    #[test]
    fn test_tls_config_client_certificates_files() {
        // Install crypto provider for rustls
        let _ = rustls::crypto::ring::default_provider().install_default();

        let tls_config = TlsConfig {
            client_cert_file: Some("src/testdata/localhost.crt".to_string()),
            client_key_file: Some("src/testdata/localhost.key".to_string()),
            ..Default::default()
        };

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

    #[test]
    fn test_tls_config_hostname_skip_order_of_operations() {
        // Test that hostname verification skip works in both scenarios:
        // 1. Only hostname skip (no custom certificates)
        // 2. Hostname skip + custom CA certificates

        // Scenario 1: Only hostname verification skip
        let tls_config_hostname_only = TlsConfig {
            insecure_skip_hostname_verify: Some(true),
            ..Default::default()
        };

        let builder1 = reqwest::ClientBuilder::new();
        let result1 = configure_tls(builder1, &tls_config_hostname_only);
        assert!(
            result1.is_ok(),
            "Should configure hostname skip without custom CA"
        );

        // Scenario 2: Hostname skip + custom CA data (the problematic combination)
        let cert_pem = include_str!("testdata/localhost.crt");
        let tls_config_combined = TlsConfig {
            ca_cert_data: Some(cert_pem.to_string()),
            insecure_skip_hostname_verify: Some(true),
            ..Default::default()
        };

        let builder2 = reqwest::ClientBuilder::new();
        let result2 = configure_tls(builder2, &tls_config_combined);
        assert!(
            result2.is_ok(),
            "Should configure hostname skip WITH custom CA"
        );

        // Scenario 3: Hostname skip + custom CA file path
        let tls_config_file = TlsConfig {
            ca_cert_file: Some("src/testdata/localhost.crt".to_string()),
            insecure_skip_hostname_verify: Some(true),
            ..Default::default()
        };

        let builder3 = reqwest::ClientBuilder::new();
        let result3 = configure_tls(builder3, &tls_config_file);
        assert!(
            result3.is_ok(),
            "Should configure hostname skip WITH custom CA file"
        );
    }

    #[test]
    fn test_load_ca_certificate_data_functions() {
        // Test the individual helper functions

        // Test with data
        let tls_config_data = TlsConfig::with_ca_data("test-cert-data");
        let result = load_ca_certificate_data(&tls_config_data);
        assert!(result.is_ok());
        assert!(result.unwrap().is_some());

        // Test with file (nonexistent)
        let tls_config_file = TlsConfig::with_ca_file("nonexistent.crt");
        let result = load_ca_certificate_data(&tls_config_file);
        assert!(result.is_err());

        // Test with neither
        let tls_config_empty = TlsConfig::default();
        let result = load_ca_certificate_data(&tls_config_empty);
        assert!(result.is_ok());
        assert!(result.unwrap().is_none());
    }

    #[test]
    fn test_load_client_certificate_data_functions() {
        // Test the client certificate helper functions

        // Test with data
        let tls_config = TlsConfig {
            client_cert_data: Some("cert-data".to_string()),
            client_key_data: Some("key-data".to_string()),
            ..Default::default()
        };
        let result = load_client_certificate_data(&tls_config);
        assert!(result.is_ok());
        assert!(result.unwrap().is_some());

        // Test with missing key data
        let tls_config_incomplete = TlsConfig {
            client_cert_data: Some("cert-data".to_string()),
            ..Default::default()
        };
        let result = load_client_certificate_data(&tls_config_incomplete);
        assert!(result.is_ok());
        assert!(result.unwrap().is_none());

        // Test with neither
        let tls_config_empty = TlsConfig::default();
        let result = load_client_certificate_data(&tls_config_empty);
        assert!(result.is_ok());
        assert!(result.unwrap().is_none());
    }

    #[tokio::test]
    async fn test_tls_self_signed_certificate_integration() {
        use crate::http::HTTPFetcherBuilder;

        // Test that we can build HTTP fetchers with various TLS configurations
        // without actually connecting (which would require a real HTTPS server)

        // Test 1: With insecure_skip_verify (should succeed)
        println!("Testing with insecure_skip_verify...");
        let tls_config_insecure = TlsConfig::insecure();

        let fetcher_result = HTTPFetcherBuilder::new("https://localhost:8443")
            .tls_config(tls_config_insecure)
            .build();

        assert!(
            fetcher_result.is_ok(),
            "Failed to build fetcher with insecure TLS: {:?}",
            fetcher_result.err()
        );

        // Test 2: With hostname verification skip
        println!("Testing with hostname verification skip...");
        let tls_config_hostname_skip = TlsConfig::skip_hostname_verify();

        let fetcher_hostname_result = HTTPFetcherBuilder::new("https://localhost:8443")
            .tls_config(tls_config_hostname_skip)
            .build();

        assert!(
            fetcher_hostname_result.is_ok(),
            "Failed to build fetcher with hostname verification skip: {:?}",
            fetcher_hostname_result.err()
        );

        // Test 3: With CA certificate file (using test cert)
        println!("Testing with CA certificate file...");
        let tls_config_ca_file = TlsConfig::with_ca_file("src/testdata/localhost.crt");

        let fetcher_ca_result = HTTPFetcherBuilder::new("https://localhost:8443")
            .tls_config(tls_config_ca_file)
            .build();

        assert!(
            fetcher_ca_result.is_ok(),
            "Failed to build fetcher with CA file: {:?}",
            fetcher_ca_result.err()
        );

        // Test 4: With CA certificate data
        println!("Testing with CA certificate data...");
        let cert_pem = include_str!("testdata/localhost.crt");
        let tls_config_ca_data = TlsConfig::with_ca_data(cert_pem);

        let fetcher_data_result = HTTPFetcherBuilder::new("https://localhost:8443")
            .tls_config(tls_config_ca_data)
            .build();

        assert!(
            fetcher_data_result.is_ok(),
            "Failed to build fetcher with CA data: {:?}",
            fetcher_data_result.err()
        );

        // Test 5: Combined CA certificate with hostname skip
        println!("Testing with CA cert and hostname verification skip...");
        let tls_config_combined =
            TlsConfig::with_ca_file_skip_hostname("src/testdata/localhost.crt");

        let fetcher_combined_result = HTTPFetcherBuilder::new("https://localhost:8443")
            .tls_config(tls_config_combined)
            .build();

        assert!(
            fetcher_combined_result.is_ok(),
            "Failed to build fetcher with combined config: {:?}",
            fetcher_combined_result.err()
        );

        println!("All TLS configuration integration tests passed!");
    }
}
