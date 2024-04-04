# Flipt Client Go

[![Client tag](https://img.shields.io/github/v/tag/flipt-io/flipt-client-go?label=latest)](https://github.com/flipt-io/flipt-client-go)
[![Go Reference](https://pkg.go.dev/badge/go.flipt.io/flipt-client.svg)](https://pkg.go.dev/go.flipt.io/flipt-client)

The `flipt-client-go` library contains the Go source code for the Flipt [client-side evaluation](https://www.flipt.io/docs/integration/client) client.

## Installation

```bash
go get go.flipt.io/flipt-client
```

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

  fmt.Println(*variantResult.Result)
}
```

## ⚠️ Linking

We package the Flipt evaluation engine as a dynamic library with the Go module. Unfortunately, we have not found a way yet to instruct Go to link the dynamic library at runtime. As a result, you will need to set the `LD_LIBRARY_PATH` environment variable to the path of the dynamic library before running your Go application.

```bash
export LD_LIBRARY_PATH=$GOPATH/pkg/mod/go.flipt.io/flipt-client@v0.x.x/ext/{OS}_{ARCH}:$LD_LIBRARY_PATH
```

Replace `v0.x.x` with the version of the client you are using.

Replace `{OS}` and `{ARCH}` with the appropriate values for your system. For example, on a Linux system with an AMD64 architecture, you would set the `LD_LIBRARY_PATH` as follows:

```bash
export LD_LIBRARY_PATH=$GOPATH/pkg/mod/go.flipt.io/flipt-client@v0.4.7/ext/linux_x86_64:$LD_LIBRARY_PATH
```

We are actively working on a solution to this problem and will update this README when we have a better solution.

See: [#205](https://github.com/flipt-io/flipt-client-sdks/issues/205) for more information.

## Memory Management

The engine that is allocated on the Rust side to compute evaluations for flag state will not be properly deallocated unless you call the `Close()` method on a `Client` instance.

**Please be sure to do this to avoid leaking memory!**

```go
defer evaluationClient.Close()
```
