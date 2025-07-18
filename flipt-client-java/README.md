# Flipt Client Java

[![flipt-client-java](https://img.shields.io/maven-central/v/io.flipt/flipt-client-java?label=flipt-client-java)](https://central.sonatype.com/artifact/io.flipt/flipt-client-java)

The `flipt-client-java` library contains the Java source code for the Flipt [client-side evaluation](https://www.flipt.io/docs/integration/client) client.

## Installation

### Gradle

Add the dependency in your `build.gradle`:

```groovy
dependencies {
    implementation 'io.flipt:flipt-client-java:1.x.x'
}
```

### Maven

Add the dependency in your `pom.xml`:

```xml
<dependency>
    <groupId>io.flipt</groupId>
    <artifactId>flipt-client-java</artifactId>
    <version>1.x.x</version>
</dependency>
```

## How Does It Work?

The `flipt-client-java` library is a wrapper around the [flipt-engine-ffi](https://github.com/flipt-io/flipt-client-sdks/tree/main/flipt-engine-ffi) library.

All evaluation happens within the SDK, using the shared library built from the [flipt-engine-ffi](https://github.com/flipt-io/flipt-client-sdks/tree/main/flipt-engine-ffi) library.

Because the evaluation happens within the SDK, the SDKs can be used in environments where the Flipt server is not available or reachable after the initial data is fetched.

## Data Fetching

Upon instantiation, the `flipt-client-java` library will fetch the flag state from the Flipt server and store it in memory. This means that the first time you use the SDK, it will make a request to the Flipt server.

### Polling (Default)

By default, the SDK will poll the Flipt server for new flag state at a regular interval. This interval can be configured using the `fetchMode` option when constructing a client. The default interval is 120 seconds.

### Streaming (Flipt Cloud/Flipt v2)

[Flipt Cloud](https://flipt.io/cloud) and [Flipt v2](https://docs.flipt.io/v2) users can use the `streaming` fetch method to stream flag state changes from the Flipt server to the SDK.

When in streaming mode, the SDK will connect to the Flipt server and open a persistent connection that will remain open until the client is closed. The SDK will then receive flag state changes in real-time.

### Retries

The SDK will automatically retry fetching (or initiating streaming) flag state if the client is unable to reach the Flipt server temporarily.

The SDK will retry up to 3 times with an exponential backoff interval between retries. The base delay is 1 second and the maximum delay is 30 seconds.

Retriable errors include:

- `429 Too Many Requests`
- `502 Bad Gateway`
- `503 Service Unavailable`
- `504 Gateway Timeout`
- Other potential transient network or DNS errors

## Supported Architectures

This SDK currently supports the following OSes/architectures:

- Linux x86_64
- Linux arm64
- MacOS x86_64
- MacOS arm64
- Windows x86_64

## Migration Notes

### Pre-1.0.0 -> 1.0.0

This section is for users who are migrating from a previous (pre-1.0.0) version of the SDK.

- `FliptEvaluationClient` has been renamed to `FliptClient`. Update all usages and imports accordingly.
- The builder and all usages should be updated to the new class name and builder pattern.
- All response models now use `List` instead of arrays (e.g., `List<String>` instead of `String[]`). Update your code to use `List` and related collection APIs.
- Exceptions are now unchecked (runtime) and use `FliptException` and its subclasses (e.g., `FliptException.EvaluationException`).
- The builder uses Lombok and is more idiomatic, reducing boilerplate.

## Usage

In your Java code you can import this client and use it as so:

```java
package org.example;

import java.util.HashMap;
import java.util.Map;
import io.flipt.client.FliptClient;
import io.flipt.client.models.*;
import io.flipt.client.FliptException;

public class Main {
    public static void main(String[] args) {
        FliptClient fliptClient = null;
        try {
            fliptClient = FliptClient.builder()
                    .url("http://localhost:8080")
                    .authentication(new ClientTokenAuthentication("secret"))
                    .build();

            Map<String, String> context = new HashMap<>();
            context.put("fizz", "buzz");

            VariantEvaluationResponse response =
                    fliptClient.evaluateVariant("flag1", "entity", context);
        } catch (FliptException.EvaluationException e) {
            e.printStackTrace();
        } finally {
            // Important: always close the client to release resources
            if (fliptClient != null) {
                fliptClient.close();
            }
        }
    }
}
```

This client is thread-safe and can be reused across your application.

### Builder Methods

The `FliptClient.builder()` method returns a `FliptClient.Builder` object that allows you to configure the client with the following methods:

- `environment`: The environment (Flipt v2) to fetch flag state from. If not provided, the client will default to the `default` environment.
- `namespace`: The namespace to fetch flag state from. If not provided, the client will default to the `default` namespace.
- `url`: The URL of the upstream Flipt instance. If not provided, the client will default to `http://localhost:8080`.
- `requestTimeout`: The timeout (Duration) for requests to the upstream Flipt instance. If not provided, the client will default to no timeout. Note: this only affects polling mode. Streaming mode will have no timeout set.
- `updateInterval`: The interval (Duration) in which to fetch new flag state. If not provided, the client will default to 120 seconds.
- `authentication`: The authentication strategy to use when communicating with the upstream Flipt instance. If not provided, the client will default to no authentication. See the [Authentication](#authentication) section for more information.
- `reference`: The [reference](https://docs.flipt.io/guides/user/using-references) to use when fetching flag state. If not provided, reference will not be used.
- `fetchMode`: The fetch mode to use when fetching flag state. If not provided, the client will default to polling.
- `errorStrategy`: The error strategy to use when fetching flag state. If not provided, the client will default to fail. See the [Error Strategies](#error-strategies) section for more information.
- `snapshot`: The initial snapshot to use when instantiating the client. See the [Snapshotting](#snapshotting) section for more information.
- `tlsConfig`: The TLS configuration to use when connecting to the upstream Flipt instance. See the [TLS Configuration](#tls-configuration) section for more information.

### Authentication

The `FliptClient` supports the following authentication strategies:

- No Authentication (default)
- [Client Token Authentication](https://docs.flipt.io/authentication/using-tokens)
- [JWT Authentication](https://docs.flipt.io/authentication/using-jwts)

### TLS Configuration

The `FliptClient` supports configuring TLS settings for secure connections to Flipt servers. This is useful when:

- Connecting to Flipt servers with self-signed certificates
- Using custom Certificate Authorities (CAs)
- Implementing mutual TLS authentication
- Testing with insecure connections (development only)

#### Basic TLS with Custom CA Certificate

```java
// Using a CA certificate file
TlsConfig tlsConfig = TlsConfig.builder()
    .caCertFile("/path/to/ca.pem")
    .build();

FliptClient client = FliptClient.builder()
    .url("https://flipt.example.com")
    .tlsConfig(tlsConfig)
    .build();
```

```java
// Using CA certificate data directly
String caCertData = Files.readString(Paths.get("/path/to/ca.pem"));
TlsConfig tlsConfig = TlsConfig.builder()
    .caCertData(caCertData)
    .build();

FliptClient client = FliptClient.builder()
    .url("https://flipt.example.com")
    .tlsConfig(tlsConfig)
    .build();
```

#### Mutual TLS Authentication

```java
// Using certificate and key files
TlsConfig tlsConfig = TlsConfig.builder()
    .clientCertFile("/path/to/client.pem")
    .clientKeyFile("/path/to/client.key")
    .build();

FliptClient client = FliptClient.builder()
    .url("https://flipt.example.com")
    .tlsConfig(tlsConfig)
    .build();
```

```java
// Using certificate and key data directly
String clientCertData = Files.readString(Paths.get("/path/to/client.pem"));
String clientKeyData = Files.readString(Paths.get("/path/to/client.key"));

TlsConfig tlsConfig = TlsConfig.builder()
    .clientCertData(clientCertData)
    .clientKeyData(clientKeyData)
    .build();

FliptClient client = FliptClient.builder()
    .url("https://flipt.example.com")
    .tlsConfig(tlsConfig)
    .build();
```

#### Advanced TLS Configuration

```java
// Full TLS configuration with all options
TlsConfig tlsConfig = TlsConfig.builder()
    .caCertFile("/path/to/ca.pem")
    .clientCertFile("/path/to/client.pem")
    .clientKeyFile("/path/to/client.key")
    .build();

FliptClient client = FliptClient.builder()
    .url("https://flipt.example.com")
    .tlsConfig(tlsConfig)
    .build();
```

```java
// Configuration for self-signed certificates with hostname mismatch
TlsConfig tlsConfig = TlsConfig.builder()
    .caCertFile("/path/to/ca.pem")
    .insecureSkipHostnameVerify(true)  // Skip hostname verification only
    .build();

FliptClient client = FliptClient.builder()
    .url("https://localhost:8443")  // hostname doesn't match certificate
    .tlsConfig(tlsConfig)
    .build();
```

#### Development Mode (Insecure)

**⚠️ WARNING: Only use this in development environments!**

```java
// Skip certificate verification entirely (NOT for production)
TlsConfig tlsConfig = TlsConfig.builder()
    .insecureSkipVerify(true)
    .build();

FliptClient client = FliptClient.builder()
    .url("https://localhost:8443")
    .tlsConfig(tlsConfig)
    .build();
```

#### Self-Signed Certificates with Hostname Mismatch

**⚠️ WARNING: Only use this when you trust the server's certificate but have hostname mismatches!**

This is useful when connecting to `localhost` or internal hostnames that don't match the certificate's Common Name (CN) or Subject Alternative Names (SAN).

```java
// Skip hostname verification while still validating the certificate
TlsConfig tlsConfig = TlsConfig.builder()
    .insecureSkipHostnameVerify(true)
    .build();

FliptClient client = FliptClient.builder()
    .url("https://localhost:8443")  // hostname doesn't match certificate
    .tlsConfig(tlsConfig)
    .build();
```

```java
// Combine custom CA with hostname verification skip (recommended for self-signed certs)
TlsConfig tlsConfig = TlsConfig.builder()
    .caCertFile("/path/to/ca.pem")
    .insecureSkipHostnameVerify(true)
    .build();

FliptClient client = FliptClient.builder()
    .url("https://localhost:8443")
    .tlsConfig(tlsConfig)
    .build();
```

**Use Case**: You have a self-signed certificate for `example.com` but need to connect via `localhost:8443`. The certificate is valid and trusted (via custom CA), but the hostname doesn't match.

#### TLS Configuration Options

The `TlsConfig` class supports the following options:

- `caCertFile`: Path to custom CA certificate file (PEM format)
- `caCertData`: Raw CA certificate content (PEM format) - takes precedence over `caCertFile`
- `insecureSkipVerify`: Skip certificate verification entirely (development only)
- `insecureSkipHostnameVerify`: Skip hostname verification while maintaining certificate validation (development only)
- `clientCertFile`: Client certificate file for mutual TLS (PEM format)
- `clientKeyFile`: Client private key file for mutual TLS (PEM format)
- `clientCertData`: Raw client certificate content (PEM format) - takes precedence over `clientCertFile`
- `clientKeyData`: Raw client private key content (PEM format) - takes precedence over `clientKeyFile`

#### Security Considerations

- **`insecureSkipVerify`**: Disables all certificate validation. This makes connections vulnerable to man-in-the-middle attacks. Only use in development.
- **`insecureSkipHostnameVerify`**: Validates the certificate chain but skips hostname verification. Use this when connecting to servers with valid certificates but hostname mismatches (e.g., `localhost` vs certificate's CN/SAN).

> **Note**: When both file paths and data are provided, the data fields take precedence. For example, if both `caCertFile` and `caCertData` are set, `caCertData` will be used.

### Error Strategies

The client `errorStrategy` method supports the following error strategies:

- `fail`: The client will throw an error if the flag state cannot be fetched. This is the default behavior.
- `fallback`: The client will maintain the last known good state and use that state for evaluation in case of an error.

### Snapshotting

The client supports snapshotting of flag state as well as seeding the client with a snapshot for evaluation. This is helpful if you want to use the client in an environment where the Flipt server is not guaranteed to be available or reachable on startup.

To get the snapshot for the client, you can use the `getSnapshot` method. This returns a base64 encoded JSON string that represents the flag state for the client.

You can set the snapshot for the client using the `snapshot` builder method when constructing a client.

**Note:** You most likely will want to also set the `errorStrategy` to `fallback` when using snapshots. This will ensure that you wont get an error if the Flipt server is not available or reachable even on the initial fetch.

You also may want to store the snapshot in a local file so that you can use it to seed the client on startup.

> [!IMPORTANT]
> If the Flipt server becomes reachable after the setting the snapshot, the client will replace the snapshot with the new flag state from the Flipt server.

## Contributing

Contributions are welcome! Please feel free to open an issue or submit a Pull Request.

## License

This project is licensed under the [MIT License](LICENSE).
