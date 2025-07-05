import io.flipt.client.models.TlsConfig;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;

public class TlsConfigTest {

  @Test
  void testInsecureSkipVerifyBuilder() {
    // Test the new convenience method (no arguments)
    TlsConfig config = TlsConfig.builder().insecureSkipVerify(true).build();

    Assertions.assertTrue(config.getInsecureSkipVerify().isPresent());
    Assertions.assertTrue(config.getInsecureSkipVerify().get());
  }

  @Test
  void testInsecureSkipHostnameVerifyBuilder() {
    // Test the new convenience method (no arguments)
    TlsConfig config = TlsConfig.builder().insecureSkipHostnameVerify(true).build();

    Assertions.assertTrue(config.getInsecureSkipHostnameVerify().isPresent());
    Assertions.assertTrue(config.getInsecureSkipHostnameVerify().get());
  }

  @Test
  void testCombinedTlsConfig() {
    // Test combining CA cert with hostname verification skip
    TlsConfig config =
        TlsConfig.builder().caCertFile("/path/to/ca.pem").insecureSkipHostnameVerify(true).build();

    Assertions.assertTrue(config.getCaCertFile().isPresent());
    Assertions.assertEquals("/path/to/ca.pem", config.getCaCertFile().get());
    Assertions.assertTrue(config.getInsecureSkipHostnameVerify().isPresent());
    Assertions.assertTrue(config.getInsecureSkipHostnameVerify().get());
  }

  @Test
  void testFullTlsConfig() {
    // Test all options together
    TlsConfig config =
        TlsConfig.builder()
            .caCertFile("/path/to/ca.pem")
            .caCertData("-----BEGIN CERTIFICATE-----\n...")
            .clientCertFile("/path/to/client.pem")
            .clientKeyFile("/path/to/client.key")
            .clientCertData("-----BEGIN CERTIFICATE-----\n...")
            .clientKeyData("-----BEGIN PRIVATE KEY-----\n...")
            .insecureSkipVerify(false)
            .insecureSkipHostnameVerify(true)
            .build();

    Assertions.assertTrue(config.getCaCertFile().isPresent());
    Assertions.assertTrue(config.getCaCertData().isPresent());
    Assertions.assertTrue(config.getClientCertFile().isPresent());
    Assertions.assertTrue(config.getClientKeyFile().isPresent());
    Assertions.assertTrue(config.getClientCertData().isPresent());
    Assertions.assertTrue(config.getClientKeyData().isPresent());
    Assertions.assertTrue(config.getInsecureSkipVerify().isPresent());
    Assertions.assertFalse(config.getInsecureSkipVerify().get());
    Assertions.assertTrue(config.getInsecureSkipHostnameVerify().isPresent());
    Assertions.assertTrue(config.getInsecureSkipHostnameVerify().get());
  }
}
