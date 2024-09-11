# Flipt Client Go

[![Client tag](https://img.shields.io/github/v/tag/flipt-io/flipt-client-go?filter=v*&label=flipt-client-go)](https://github.com/flipt-io/flipt-client-go)
[![Client tag (musl)](https://img.shields.io/github/v/tag/flipt-io/flipt-client-go?filter=musl-v*&label=flipt-client-go-musl)](https://github.com/flipt-io/flipt-client-go)
[![Go Reference](https://pkg.go.dev/badge/go.flipt.io/flipt-client.svg)](https://pkg.go.dev/go.flipt.io/flipt-client)

The `flipt-client-go` library contains the Go source code for the Flipt [client-side evaluation](https://www.flipt.io/docs/integration/client) client.

## Installation

```bash
go get go.flipt.io/flipt-client
```

## Supported Architectures

This SDK currently supports the following OSes/architectures:

- Linux x86_64
- Linux x86_64 (musl)
- Linux arm64
- Linux arm64 (musl)
- MacOS x86_64
- MacOS arm64

### Glibc vs Musl

Most Linux distributions use [Glibc](https://en.wikipedia.org/wiki/Glibc), but some distributions like Alpine Linux use [Musl](https://en.wikipedia.org/wiki/Musl). If you are using Alpine Linux, you will need to install the `musl` tagged version of the client.

Example:

```bash
go install go.flipt.io/flipt-client@musl-v0.0.1
```

See [flipt-client-sdks #141](https://github.com/flipt-io/flipt-client-sdks/issues/141) for more information.

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
  evaluationClient, err := flipt.NewClient()
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

### Authentication

The `Client` supports the following authentication strategies:

- No Authentication (default)
- [Client Token Authentication](https://docs.flipt.io/authentication/using-tokens)
- [JWT Authentication](https://docs.flipt.io/authentication/using-jwts)

## Memory Management

The engine that is allocated on the Rust side to compute evaluations for flag state will not be properly deallocated unless you call the `Close()` method on a `Client` instance.

**Please be sure to do this to avoid leaking memory!**

```go
defer evaluationClient.Close()
```

## Contributing

Contributions are welcome! Please feel free to open an issue or submit a Pull Request.

## License

This project is licensed under the [MIT License](LICENSE).
