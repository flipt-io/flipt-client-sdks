use crate::Error;
use serde::Deserialize;
use std::io::Cursor;
use std::sync::Arc;

use rustls::client::danger::{HandshakeSignatureValid, ServerCertVerified, ServerCertVerifier};
use rustls::crypto::{verify_tls12_signature, verify_tls13_signature, WebPkiSupportedAlgorithms};
use rustls::pki_types::pem::PemObject;
use rustls::pki_types::{CertificateDer, PrivateKeyDer, ServerName, UnixTime};
use rustls::server::ParsedCertificate;
use rustls::{DigitallySignedStruct, Error as TlsError, RootCertStore, SignatureScheme};

#[derive(Deserialize, Debug, PartialEq, Default)]
pub struct TlsConfig {
    /// Path to custom CA certificate file (PEM format)
    ca_cert_file: Option<String>,
    /// Raw CA certificate content (PEM format)
    ca_cert_data: Option<String>,
    /// Skip certificate verification (insecure - for development only)
    insecure_skip_verify: Option<bool>,
    /// Skip hostname verification. The verifier keeps certificate validation (insecure - for development only)
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

/// Certificate verifier that checks the chain against the root store.
/// It ignores the hostname.
///
/// It follows the internal `IgnoreHostname` of reqwest.
/// It uses the native roots of the platform plus the custom CA.
/// The standard hostname flag does not accept system roots,
/// so the code keeps its own verifier for that case.
#[derive(Debug)]
struct NoHostnameVerifier {
    roots: RootCertStore,
    signature_algorithms: WebPkiSupportedAlgorithms,
}

impl NoHostnameVerifier {
    fn new(roots: RootCertStore, signature_algorithms: WebPkiSupportedAlgorithms) -> Self {
        Self {
            roots,
            signature_algorithms,
        }
    }
}

impl ServerCertVerifier for NoHostnameVerifier {
    fn verify_server_cert(
        &self,
        end_entity: &CertificateDer<'_>,
        intermediates: &[CertificateDer<'_>],
        _server_name: &ServerName<'_>,
        _ocsp_response: &[u8],
        now: UnixTime,
    ) -> Result<ServerCertVerified, TlsError> {
        let cert = ParsedCertificate::try_from(end_entity)?;

        rustls::client::verify_server_cert_signed_by_trust_anchor(
            &cert,
            &self.roots,
            intermediates,
            now,
            self.signature_algorithms.all,
        )?;
        Ok(ServerCertVerified::assertion())
    }

    fn verify_tls12_signature(
        &self,
        message: &[u8],
        cert: &CertificateDer<'_>,
        dss: &DigitallySignedStruct,
    ) -> Result<HandshakeSignatureValid, TlsError> {
        verify_tls12_signature(message, cert, dss, &self.signature_algorithms)
    }

    fn verify_tls13_signature(
        &self,
        message: &[u8],
        cert: &CertificateDer<'_>,
        dss: &DigitallySignedStruct,
    ) -> Result<HandshakeSignatureValid, TlsError> {
        verify_tls13_signature(message, cert, dss, &self.signature_algorithms)
    }

    fn supported_verify_schemes(&self) -> Vec<SignatureScheme> {
        self.signature_algorithms.supported_schemes()
    }
}

/// Configures the TLS configuration for an HTTP client.
///
/// This function handles the TLS configuration scenarios:
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
    builder = builder.tls_backend_rustls();

    // Handle insecure mode (skip all validation)
    if tls_config.insecure_skip_verify.unwrap_or(false) {
        builder = builder.tls_danger_accept_invalid_certs(true);
        return Ok(builder);
    }

    // Skip hostname verification with the preconfigured ClientConfig of rustls.
    // The standard hostname flag does not accept system roots, so the code builds
    // its own verifier from the root store of the system plus the custom CA.
    // The verifier checks the chain. It keeps SNI enabled. It skips only
    // the hostname check.
    if tls_config.insecure_skip_hostname_verify.unwrap_or(false) {
        let rustls_config = build_hostname_blind_config(tls_config)?;
        builder = builder.tls_backend_preconfigured(rustls_config);
        return Ok(builder);
    }

    // Add the custom CA certificates with the standard method of reqwest
    let ca_cert_data = load_ca_certificate_data(tls_config)?;
    if let Some(cert_bytes) = ca_cert_data {
        for cert in parse_ca_certificates(&cert_bytes)? {
            let cert = reqwest::Certificate::from_der(&cert)
                .map_err(|e| Error::Internal(format!("failed to parse CA certificate: {e}")))?;
            builder = builder.add_root_certificate(cert);
        }
    }

    // Handle client certificates for mutual TLS
    let client_cert_data = load_client_certificate_data(tls_config)?;
    if let Some((cert_bytes, key_bytes)) = client_cert_data {
        // Combine the client certificate with the key for the identity of reqwest
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

/// Builds the ClientConfig of rustls for skip of hostname verification.
/// It checks the chain against the root store of the system plus the custom CA.
/// If the TLS configuration contains a client certificate, it presents that certificate.
/// It keeps SNI enabled. The verifier skips only the hostname check.
fn build_hostname_blind_config(tls_config: &TlsConfig) -> Result<rustls::ClientConfig, Error> {
    let provider = rustls::crypto::CryptoProvider::get_default()
        .cloned()
        .unwrap_or_else(|| Arc::new(rustls::crypto::aws_lc_rs::default_provider()));
    let signature_algorithms = provider.signature_verification_algorithms;

    let mut roots = RootCertStore::empty();
    let native = rustls_native_certs::load_native_certs();
    if !native.errors.is_empty() {
        log::warn!(
            "errors loading native certs for hostname-skip TLS: {:?}",
            native.errors
        );
    }
    for cert in native.certs {
        roots
            .add(cert)
            .map_err(|e| Error::Internal(format!("failed to add native root certificate: {e}")))?;
    }

    if let Some(ca_bytes) = load_ca_certificate_data(tls_config)? {
        let custom = parse_ca_certificates(&ca_bytes)?;
        if custom.is_empty() {
            return Err(Error::Internal(
                "no valid certificates found in CA certificate data".into(),
            ));
        }
        for cert in custom {
            roots.add(cert).map_err(|e| {
                Error::Internal(format!("failed to add custom CA certificate: {e}"))
            })?;
        }
    }

    if roots.is_empty() {
        return Err(Error::Internal(
            "no root certificates available for hostname-skip TLS".into(),
        ));
    }

    let verifier = Arc::new(NoHostnameVerifier::new(roots, signature_algorithms));
    let config_builder = rustls::ClientConfig::builder_with_provider(provider)
        .with_protocol_versions(rustls::ALL_VERSIONS)
        .map_err(|_| Error::Internal("invalid TLS versions".into()))?
        .dangerous()
        .with_custom_certificate_verifier(verifier);

    let mut config = if let Some((cert_bytes, key_bytes)) =
        load_client_certificate_data(tls_config)?
    {
        let mut pem_data = Vec::new();
        pem_data.extend_from_slice(&cert_bytes);
        pem_data.extend_from_slice(&key_bytes);
        let (certs, key) = parse_client_identity(&pem_data)?;
        config_builder
            .with_client_auth_cert(certs, key)
            .map_err(|e| Error::Internal(format!("failed to configure client certificate: {e}")))?
    } else {
        config_builder.with_no_client_auth()
    };

    // Keep SNI enabled. The verifier skips only the hostname check.
    config.enable_sni = true;
    config.alpn_protocols = vec![b"h2".to_vec(), b"http/1.1".to_vec()];

    Ok(config)
}

/// Parses PEM CA bundle bytes into DER certificates.
fn parse_ca_certificates(pem_bytes: &[u8]) -> Result<Vec<CertificateDer<'static>>, Error> {
    CertificateDer::pem_slice_iter(pem_bytes)
        .collect::<Result<Vec<_>, _>>()
        .map_err(|_| Error::Internal("invalid CA certificate encoding".into()))
}

/// Parses combined client cert/key PEM bytes, mirroring
/// `reqwest::Identity::from_pem` (RSA, SEC1, PKCS#8 keys supported).
fn parse_client_identity(
    pem_bytes: &[u8],
) -> Result<(Vec<CertificateDer<'static>>, PrivateKeyDer<'static>), Error> {
    use rustls::pki_types::pem::{self, SectionKind};

    let mut cursor = Cursor::new(pem_bytes);
    let mut keys = Vec::new();
    let mut certs = Vec::new();

    while let Some((kind, data)) = pem::from_buf(&mut cursor)
        .map_err(|_| Error::Internal("invalid client identity PEM".into()))?
    {
        match kind {
            SectionKind::Certificate => certs.push(data.into()),
            SectionKind::PrivateKey => keys.push(PrivateKeyDer::Pkcs8(data.into())),
            SectionKind::RsaPrivateKey => keys.push(PrivateKeyDer::Pkcs1(data.into())),
            SectionKind::EcPrivateKey => keys.push(PrivateKeyDer::Sec1(data.into())),
            _ => {
                return Err(Error::Internal(
                    "unsupported section in client identity PEM".into(),
                ));
            }
        }
    }

    match (keys.pop(), certs.is_empty()) {
        (Some(key), false) => Ok((certs, key)),
        _ => Err(Error::Internal(
            "private key or certificate not found in client identity".into(),
        )),
    }
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

    #[test]
    fn test_hostname_skip_with_mtls_builds() {
        let cert_pem = include_str!("testdata/localhost.crt");
        let key_pem = include_str!("testdata/localhost.key");
        let tls_config = TlsConfig {
            insecure_skip_hostname_verify: Some(true),
            client_cert_data: Some(cert_pem.to_string()),
            client_key_data: Some(key_pem.to_string()),
            ..Default::default()
        };

        let builder = reqwest::Client::builder();
        let configured = configure_tls(builder, &tls_config).unwrap();
        assert!(configured.build().is_ok());
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
