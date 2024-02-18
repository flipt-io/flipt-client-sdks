# Flipt Client Java

[![Maven Central](https://img.shields.io/maven-central/v/io.flipt/flipt-client-java)](https://central.sonatype.com/artifact/io.flipt/flipt-client-java)

The `flipt-client-java` directory contains the Java source code for the Flipt [client-side evaluation](https://www.flipt.io/docs/integration/client) client.

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
        try {
            FliptEvaluationClient fliptClient = FliptEvaluationClient.builder()
                    .url("http://localhost:8080")
                    .authentication(new ClientTokenAuthentication("secret"))
                    .build();

            Map<String, String> context = new HashMap<>();
            context.put("fizz", "buzz");

            Result<VariantEvaluationResponse> result =
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
