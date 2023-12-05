# Flipt Client Go

The `flipt-client-go` directory contains the Go source code for a Flipt evaluation client.

## Instructions

To use this client, you can run the following command from the root of the repository:

```bash
cargo build --release
```

This should generate a `target/` directory in the root of this repository, which contains the dynamic linking library built for your platform. This dynamic library will contain the functionality necessary for the Go client to make FFI calls.

You can import the module that contains the evaluation client: `go.flipt.io/flipt/flipt-client-go` and build your Go project with the `CGO_LDFLAGS` environment variable set:

```bash
CGO_LDFLAGS="-L/path/to/lib -lfliptengine"
```

The `path/to/lib` will be the path to the dynamic library which will have the following paths depending on your platform.

- **Linux**: `{REPO_ROOT}/target/release/libfliptengine.so`
- **Windows**: `{REPO_ROOT}/target/release/libfliptengine.dll`
- **MacOS**: `{REPO_ROOT}/target/release/libfliptengine.dylib`

You can then use the client like so:

```go
package main

import (
 "context"
 "fmt"
 "log"

 evaluation "go.flipt.io/flipt/flipt-client-go"
)

func main() {
 // The NewClient() accepts options which are the following:
 // evaluation.WithNamespace(string): configures which namespace you will be making evaluations on
 // evaluation.WithURL(string): configures which upstream Flipt data should be fetched from
 // evaluation.WithUpdateInterval(int): configures how often data should be fetched from the upstream
 // evaluation.WithAuthToken(string): configures an auth token if your upstream Flipt instance requires it
 evaluationClient, err := evaluation.NewClient()
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

The engine that is allocated on the Rust side to compute evaluations for flag state will not be properly deallocated unless you call the `Close()` method on a `Client` instance. Please be sure to do this to avoid leaking memory.

```go
defer evaluationClient.Close()
```
