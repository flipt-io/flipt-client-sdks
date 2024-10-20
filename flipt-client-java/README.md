# Flipt Client Java

[![Maven Central](https://img.shields.io/maven-central/v/io.flipt/flipt-client-java?label=flipt-client-java)](https://central.sonatype.com/artifact/io.flipt/flipt-client-java)
[![Maven Central (Musl)](https://img.shields.io/maven-central/v/io.flipt/flipt-client-java-musl?label=flipt-client-java-musl)](https://central.sonatype.com/artifact/io.flipt/flipt-client-java-musl)

The `flipt-client-java` library contains the Java source code for the Flipt [client-side evaluation](https://www.flipt.io/docs/integration/client) client.

## Installation

### Gradle

Add the dependency in your `build.gradle`:

```groovy
dependencies {
    implementation 'io.flipt:flipt-client-java:0.x.x'
}
```

### Maven

Add the dependency in your `pom.xml`:

```xml
<dependency>
    <groupId>io.flipt</groupId>
    <artifactId>flipt-client-java</artifactId>
    <version>0.x.x</version>
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

### Streaming (Flipt Cloud Only)

[Flipt Cloud](https://flipt.io/cloud) users can use the `streaming` fetch method to stream flag state changes from the Flipt server to the SDK.

When in streaming mode, the SDK will connect to the Flipt server and open a persistent connection that will remain open until the client is closed. The SDK will then receive flag state changes in real-time.

## Supported Architectures

This SDK currently supports the following OSes/architectures:

- Linux x86_64
- Linux x86_64 (musl)
- Linux arm64
- Linux arm64 (musl)
- MacOS x86_64
- MacOS arm64
- Windows x86_64

### Glibc vs Musl

Most Linux distributions use [Glibc](https://en.wikipedia.org/wiki/Glibc), but some distributions like Alpine Linux use [Musl](https://en.wikipedia.org/wiki/Musl). If you are using Alpine Linux, you will need to install the `musl` version of the client.

## Usage

In your Java code you can import this client and use it as so:

```java
package org.example;

import java.util.HashMap;
import java.util.Map;
import io.flipt.client.FliptEvaluationClient;
import io.flipt.client.models.*;

public class Main {
    public static void main(String[] args) {
        FliptEvaluationClient fliptClient = null;
        try {
            fliptClient = FliptEvaluationClient.builder()
                    .url("http://localhost:8080")
                    .authentication(new ClientTokenAuthentication("secret"))
                    .build();

            Map<String, String> context = new HashMap<>();
            context.put("fizz", "buzz");

            VariantEvaluationResponse response =
                    fliptClient.evaluateVariant("flag1", "entity", context);
        } catch (Exception e) {
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

### Builder Methods

The `FliptEvaluationClient.builder()` method returns a `FliptEvaluationClient.Builder` object that allows you to configure the client with the following methods:

- `namespace`: The namespace to fetch flag state from. If not provided, the client will default to the `default` namespace.
- `url`: The URL of the upstream Flipt instance. If not provided, the client will default to `http://localhost:8080`.
- `updateInterval`: The interval (Duration) in which to fetch new flag state. If not provided, the client will default to 120 seconds.
- `authentication`: The authentication strategy to use when communicating with the upstream Flipt instance. If not provided, the client will default to no authentication. See the [Authentication](#authentication) section for more information.
- `reference`: The [reference](https://docs.flipt.io/guides/user/using-references) to use when fetching flag state. If not provided, reference will not be used.

### Authentication

The `FliptEvaluationClient` supports the following authentication strategies:

- No Authentication (default)
- [Client Token Authentication](https://docs.flipt.io/authentication/using-tokens)
- [JWT Authentication](https://docs.flipt.io/authentication/using-jwts)

## Contributing

Contributions are welcome! Please feel free to open an issue or submit a Pull Request.

## License

This project is licensed under the [MIT License](LICENSE).
