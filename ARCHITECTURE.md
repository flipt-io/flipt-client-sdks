# Architecture

## FFI

[`flipt-engine-ffi`](./flipt-engine-ffi/) is a Rust library that polls for evaluation state from the Flipt server and uses this state to determine the evaluation results. The engine is designed to be embedded in the native language client SDKs. The native language client SDKs send context to the client engine via [FFI](https://en.wikipedia.org/wiki/Foreign_function_interface) and receive the results of the evaluation from the engine.

This design allows for the client evaluation logic to be written once in a memory-safe language (Rust) and embedded in the native language client SDKs.

You can refer to the architecture diagram below:

<p align="center">
    <img src=".github/images/architecture-ffi.png" alt="Client SDKs Architecture (FFI)" width="500px" />
</p>

### Evaluation Library

[`flipt-engine-ffi`](./flipt-engine-ffi) uses the [`flipt-evaluation`](./flipt-evaluation) library to actually evaluate context against the evaluation state. The [`flipt-evaluation`](../flipt-evaluation) library is a Rust library responsible for the following:

- Polling for evaluation state from the Flipt server.
- Unmarshalling the evaluation state from JSON to memory.
- Storing the evaluation state in memory per namespace.
- Evaluating context against the evaluation state and returning the evaluation results.

The evaluation logic is extracted into a separate library to allow for the evaluation logic to be reused by both the FFI engine and a potential future WASM engine.

### Client SDKs

The client SDKs are responsible for the following:

- Marshalling context to JSON.
- Sending context to the engine via FFI.
- Unmarshalling the results of the evaluation from JSON to memory and returning the results to the caller.
- Providing a high-level API for the caller to interact with the client SDK.
