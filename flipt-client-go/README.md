<!-- Last published: Wed Dec 27 22:18:09 UTC 2023 -->
# Flipt Client Go

[![Client tag](https://img.shields.io/github/v/tag/flipt-io/flipt-client-go?label=latest)](https://github.com/flipt-io/flipt-client-go)

The `flipt-client-go` directory contains the Go source code for the Flipt [client-side evaluation](https://www.flipt.io/docs/integration/client) client.

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
  // The NewClient() accepts options which are the following:
  // flipt.WithNamespace(string): configures which namespace you will be making evaluations on
  // flipt.WithURL(string): configures which upstream Flipt data should be fetched from
  // flipt.WithUpdateInterval(int): configures how often data should be fetched from the upstream
  // flipt.WithAuthToken(string): configures an auth token if your upstream Flipt instance requires it
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

## Memory Management

The engine that is allocated on the Rust side to compute evaluations for flag state will not be properly deallocated unless you call the `Close()` method on a `Client` instance.

**Please be sure to do this to avoid leaking memory!**

```go
defer evaluationClient.Close()
```
