# Flipt Client Go

[![Client tag](https://img.shields.io/github/v/tag/flipt-io/flipt-client-go?filter=v*&label=flipt-client-go)](https://github.com/flipt-io/flipt-client-go)
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

### Streaming (Flipt Cloud Only)

[Flipt Cloud](https://flipt.io/cloud) users can use the `streaming` fetch method to stream flag state changes from the Flipt server to the SDK.

When in streaming mode, the SDK will connect to the Flipt server and open a persistent connection that will remain open until the client is closed. The SDK will then receive flag state changes in real-time.

## Supported Architectures

This SDK currently supports the following OSes/architectures:

- Linux x86_64
- Linux arm64
- MacOS x86_64
- MacOS arm64
- Windows x86_64

## Usage

In your Go code you can import this client and use it as so:

```go
package main

import (
  "context"
  "fmt"
  "log"

  flipt "go.flipt.io/flipt-client"
)

func main() {
  evaluationClient, err := flipt.NewEvaluationClient(context.Background())
  if err != nil {
    log.Fatal(err)
  }

  defer evaluationClient.Close()

  variantResult, err := evaluationClient.EvaluateVariant(context.Background(), "flag1", "someentity", map[string]string{
    "fizz": "buzz",
  })

  if err != nil {
    log.Fatal(err)
  }

  fmt.Println(*variantResult)
}
```

### Client Options

The `NewClient` constructor accepts a variadic number of `ClientOption` functions that can be used to configure the client. The available options are:

- `WithNamespace`: The namespace to fetch flag state from. If not provided, the client will default to the `default` namespace.
- `WithURL`: The URL of the upstream Flipt instance. If not provided, the client will default to `http://localhost:8080`.
- `WithUpdateInterval`: The interval (in seconds) in which to fetch new flag state. If not provided, the client will default to 120 seconds.
- `With{Method}Authentication`: The authentication strategy to use when communicating with the upstream Flipt instance. If not provided, the client will default to no authentication. See the [Authentication](#authentication) section for more information.
- `WithReference`: The [reference](https://docs.flipt.io/guides/user/using-references) to use when fetching flag state. If not provided, reference will not be used.
- `WithFetchMode`: The fetch mode to use when fetching flag state. If not provided, the client will default to polling.
- `WithErrorStrategy`: The error strategy to use when fetching flag state. If not provided, the client will default to `Fail`. See the [Error Strategies](#error-strategies) section for more information.

### Authentication

The `Client` supports the following authentication strategies:

- No Authentication (default)
- [Client Token Authentication](https://docs.flipt.io/authentication/using-tokens)
- [JWT Authentication](https://docs.flipt.io/authentication/using-jwts)

### Error Strategies

The `Client` supports the following error strategies:

- `Fail`: The client will throw an error if the flag state cannot be fetched. This is the default behavior.
- `Fallback`: The client will maintain the last known good state and use that state for evaluation in case of an error.

## Memory Management

The engine that is allocated on the Rust side to compute evaluations for flag state will not be properly deallocated unless you call the `Close()` method on a `Client` instance.

**Please be sure to do this to avoid leaking memory!**

```go
defer evaluationClient.Close(context.Background())
```

## Benchmarking

The `flipt-client-go` library includes a benchmarking suite that can be used to test the performance of the SDK.

To run the benchmarks, use the following command:

```bash
go test -benchmem -run='^$' -bench=. go.flipt.io/flipt-client
```

You should see output similar to the following:

```
goos: darwin
goarch: arm64
pkg: go.flipt.io/flipt-client
cpu: Apple M1 Max
BenchmarkVariantEvaluation-10    	  122054	      9892 ns/op	   36207 B/op	      36 allocs/op
BenchmarkBooleanEvaluation-10    	  124194	      9141 ns/op	   36095 B/op	      34 allocs/op
BenchmarkBatchEvaluation-10      	   62458	     19393 ns/op	   36490 B/op	      50 allocs/op
BenchmarkListFlags-10            	  224454	      5464 ns/op	   23928 B/op	      25 allocs/op
PASS
ok  	go.flipt.io/flipt-client	5.746s
```

## Contributing

Contributions are welcome! Please feel free to open an issue or submit a Pull Request.

## License

This project is licensed under the [MIT License](LICENSE).
