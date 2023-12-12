# Flipt Client Go

The `flipt-client-go` directory contains the Go source code for the Flipt client-side evaluation client.

## Installation

Currently, to use this client, you'll need to build the dynamic library and the gem locally and install it. This is a temporary solution until we can figure out a better way to package and distribute the libraries.

The dynamic library will contain the functionality necessary for the client to make calls to the Flipt engine via FFI. See [flipt-engine](../flipt-engine) for more information on the Flipt engine and FFI.

### Prerequisites

- [Rust](https://www.rust-lang.org/tools/install)
- [Go](https://golang.org/doc/install)
- [Make](https://www.gnu.org/software/make/)

### Automated Build

1. Build and copy the dynamic library to the `flipt-client-go/ext` directory.

    ```bash
    make go
    ```

### Manual Build

1. Build the Rust dynamic library

    ```bash
    cargo build --release
    ```

This should generate a `target/` directory in the root of this repository, which contains the dynamically linked library built for your platform.

2. You'll need to copy the dynamic library to the `flipt-client-go/ext` directory. This is a temporary solution until we can figure out a better way to package the libraries with the module.

The `path/to/lib` will be the path to the dynamic library which will have the following paths depending on your platform.

- **Linux**: `{REPO_ROOT}/target/release/libfliptengine.so`
- **Windows**: `{REPO_ROOT}/target/release/libfliptengine.dll`
- **MacOS**: `{REPO_ROOT}/target/release/libfliptengine.dylib`

3. You can then install the module locally. You can do this by running the following command from the `flipt-client-go` directory:

    ```bash
    go install .
    ```

## Usage

In your Go code you can import this client and use it as so:

```go
package main

import (
 "context"
 "fmt"
 "log"

 evaluation "go.flipt.io/flipt/flipt-client"
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
