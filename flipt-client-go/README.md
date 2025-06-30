# Flipt Client Go

[![flipt-client-go](https://img.shields.io/github/v/tag/flipt-io/flipt-client-go?filter=v*&label=flipt-client-go)](https://github.com/flipt-io/flipt-client-go)
[![Go Reference](https://pkg.go.dev/badge/go.flipt.io/flipt-client.svg)](https://pkg.go.dev/go.flipt.io/flipt-client)

The `flipt-client-go` library contains the Go source code for the Flipt [client-side evaluation](https://www.flipt.io/docs/integration/client) client.

## Installation

```bash
go get go.flipt.io/flipt-client
```

## How Does It Work?

The `flipt-client-go` library is a wrapper around the [flipt-engine-wasm](https://github.com/flipt-io/flipt-client-sdks/tree/main/flipt-engine-wasm) library.

All evaluation happens within the SDK, using the shared library built from the [flipt-engine-wasm](https://github.com/flipt-io/flipt-client-sdks/tree/main/flipt-engine-wasm) library.

We use the [Wazero](https://github.com/tetratelabs/wazero) library to load the WASM module and call the functions that are exported from the module. Wazero implements a WebAssembly runtime for Go without the need for using CGO.

Because the evaluation happens within the SDK, the SDKs can be used in environments where the Flipt server is not available or reachable after the initial data is fetched.

## Data Fetching

Upon instantiation, the `flipt-client-go` library will fetch the flag state from the Flipt server and store it in memory. This means that the first time you use the SDK, it will make a request to the Flipt server.

### Polling (Default)

By default, the SDK will poll the Flipt server for new flag state at a regular interval. This interval can be configured using the `WithUpdateInterval` option when constructing a client. The default interval is 120 seconds.

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

- The package name is now `flipt` and the main client is `Client`.
- `WithEnvironment` is now a valid option for fetching flag state from Flipt v2.
- All client methods now use request structs (e.g., `EvaluateVariant(ctx, &flipt.EvaluationRequest{...})`).
- Configuration is now done via functional options (e.g., `WithNamespace`, `WithURL`, etc.).
- Error handling uses standardized error types and codes.

## Usage

```go
package main

import (
  "context"
  "fmt"
  "log"
  "time"

  flipt "go.flipt.io/flipt-client"
)

func main() {
  ctx := context.Background()
  client, err := flipt.NewClient(
    ctx,
    flipt.WithURL("http://localhost:8080"),
    flipt.WithRequestTimeout(30 * time.Second),
    flipt.WithUpdateInterval(2 * time.Minute),
    flipt.WithFetchMode(flipt.FetchModePolling),
    flipt.WithErrorStrategy(flipt.ErrorStrategyFail),
  )
  if err != nil {
    log.Fatal(err)
  }
  defer client.Close(ctx)

  variantResult, err := client.EvaluateVariant(ctx, &flipt.EvaluationRequest{
    FlagKey:  "flag1",
    EntityID: "someentity",
    Context:  map[string]string{"fizz": "buzz"},
  })
  if err != nil {
    log.Fatal(err)
  }
  fmt.Println(variantResult)
}
```

### Client Options

The `NewClient` constructor accepts a variadic number of `Option` functions that can be used to configure the client. The available options are:

- `WithEnvironment`: The environment (Flipt v2) to fetch flag state from. If not provided, the client will default to the `default` environment.
- `WithNamespace`: The namespace to fetch flag state from. If not provided, the client will default to the `default` namespace.
- `WithURL`: The URL of the upstream Flipt instance. If not provided, the client will default to `http://localhost:8080`.
- `WithRequestTimeout`: The timeout for total request time to the upstream Flipt instance. If not provided, the client will default to no timeout. Note: this only affects polling mode. Streaming mode will have no timeout set.
- `WithUpdateInterval`: The interval in which to fetch new flag state. If not provided, the client will default to 120 seconds.
- `With{Method}Authentication`: The authentication strategy to use when communicating with the upstream Flipt instance. If not provided, the client will default to no authentication. See the [Authentication](#authentication) section for more information.
- `WithReference`: The [reference](https://docs.flipt.io/guides/user/using-references) to use when fetching flag state. If not provided, reference will not be used.
- `WithFetchMode`: The fetch mode to use when fetching flag state. If not provided, the client will default to polling.
- `WithErrorStrategy`: The error strategy to use when fetching flag state. If not provided, the client will default to `Fail`. See the [Error Strategies](#error-strategies) section for more information.
- `WithTLSConfig`: The TLS configuration for connecting to servers with custom certificates. See [TLS Configuration](#tls-configuration).

### Authentication

The `Client` supports the following authentication strategies:

- No Authentication (default)
- [Client Token Authentication](https://docs.flipt.io/authentication/using-tokens)
- [JWT Authentication](https://docs.flipt.io/authentication/using-jwts)

### TLS Configuration

The `Client` supports configuring TLS settings for secure connections to Flipt servers using the standard library `tls.Config`. This provides maximum flexibility for:

- Connecting to Flipt servers with self-signed certificates
- Using custom Certificate Authorities (CAs)
- Implementing mutual TLS authentication
- Testing with insecure connections (development only)

#### Basic TLS with Custom CA Certificate

```go
package main

import (
  "context"
  "crypto/tls"
  "crypto/x509"
  "log"
  "os"

  flipt "go.flipt.io/flipt-client"
)

func main() {
  ctx := context.Background()

  // Load CA certificate
  caCert, err := os.ReadFile("/path/to/ca.pem")
  if err != nil {
    log.Fatal(err)
  }

  caCertPool := x509.NewCertPool()
  caCertPool.AppendCertsFromPEM(caCert)

  tlsConfig := &tls.Config{
    RootCAs: caCertPool,
  }

  client, err := flipt.NewClient(
    ctx,
    flipt.WithURL("https://flipt.example.com"),
    flipt.WithTLSConfig(tlsConfig),
    flipt.WithClientTokenAuthentication("your-token"),
  )
  if err != nil {
    log.Fatal(err)
  }
  defer client.Close(ctx)
}
```

#### Mutual TLS Authentication

```go
// Load client certificate and key
clientCert, err := tls.LoadX509KeyPair("/path/to/client.pem", "/path/to/client.key")
if err != nil {
  log.Fatal(err)
}

// Load CA certificate
caCert, err := os.ReadFile("/path/to/ca.pem")
if err != nil {
  log.Fatal(err)
}

caCertPool := x509.NewCertPool()
caCertPool.AppendCertsFromPEM(caCert)

tlsConfig := &tls.Config{
  Certificates: []tls.Certificate{clientCert},
  RootCAs:      caCertPool,
}

client, err := flipt.NewClient(
  ctx,
  flipt.WithURL("https://flipt.example.com"),
  flipt.WithTLSConfig(tlsConfig),
  flipt.WithClientTokenAuthentication("your-token"),
)
```

#### Development Mode (Insecure)

**⚠️ WARNING: Only use this in development environments!**

```go
// Skip certificate verification (NOT for production)
tlsConfig := &tls.Config{
  InsecureSkipVerify: true,
}

client, err := flipt.NewClient(
  ctx,
  flipt.WithURL("https://localhost:8443"),
  flipt.WithTLSConfig(tlsConfig),
  flipt.WithClientTokenAuthentication("your-token"),
)
```

#### Advanced TLS Configuration

Since the client accepts a standard `tls.Config`, you have access to all TLS configuration options:

```go
tlsConfig := &tls.Config{
  // Custom CA certificates
  RootCAs: caCertPool,
  
  // Client certificates for mutual TLS
  Certificates: []tls.Certificate{clientCert},
  
  // Minimum TLS version
  MinVersion: tls.VersionTLS12,
  
  // Custom cipher suites
  CipherSuites: []uint16{
    tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
    tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
  },
  
  // Server name for SNI
  ServerName: "flipt.example.com",
}
```

### Error Strategies

The `Client` supports the following error strategies:

- `ErrorStrategyFail`: The client will return an error for any method calls if the flag state cannot be fetched. This is the default behavior.
- `ErrorStrategyFallback`: The client will maintain the last known good state and use that state for evaluation in case of an error.

### `Err` Method

The `Err` method can be used to check the last error that occurred regardless of the error strategy. This allows you to handle errors in a more flexible way and decide what to do based on the error.

```go
err := client.Err()
if err != nil {
  log.Fatal(err)
}
```

In the case of non-fetch related errors, the client will stop polling or streaming for flag state changes and return the error for all subsequent method calls.

## Memory Management

The engine that is allocated on the WASM side to compute evaluations for flag state will not be properly deallocated unless you call the `Close()` method on a `Client` instance.

**Please be sure to do this to avoid leaking memory!**

```go
defer client.Close(context.Background())
```

## Benchmarking

The `flipt-client-go` library includes a benchmarking suite that can be used to test the performance of the SDK.

To run the benchmarks, use the following command:

```bash
go test -tags=benchmarks -benchmem -run='^$' -bench=. go.flipt.io/flipt-client
```

You should see output similar to the following:

```text
goos: darwin
goarch: arm64
pkg: go.flipt.io/flipt-client
cpu: Apple M1 Max
BenchmarkVariantEvaluation/Simple-10              160869              7426 ns/op           12900 B/op         32 allocs/op
BenchmarkVariantEvaluation/MediumContext-10       106569             11324 ns/op           13365 B/op         48 allocs/op
BenchmarkVariantEvaluation/LargeContext-10         18770             63237 ns/op           21763 B/op        228 allocs/op
BenchmarkBooleanEvaluation/Simple-10              181298              6624 ns/op           12788 B/op         30 allocs/op
BenchmarkBooleanEvaluation/MediumContext-10       114070             10429 ns/op           13253 B/op         46 allocs/op
BenchmarkBooleanEvaluation/LargeContext-10         19189             62621 ns/op           21650 B/op        226 allocs/op
BenchmarkBatchEvaluation/Simple-10                 74679             16137 ns/op           13181 B/op         46 allocs/op
BenchmarkBatchEvaluation/MediumBatch-10            15253             79676 ns/op           24290 B/op        301 allocs/op
BenchmarkBatchEvaluation/LargeBatch-10               405           2942603 ns/op          489492 B/op      10383 allocs/op
BenchmarkListFlags-10                             289599              4141 ns/op           12280 B/op         23 allocs/op
```

## Contributing

Contributions are welcome! Please feel free to open an issue or submit a Pull Request.

## License

This project is licensed under the [MIT License](LICENSE).
