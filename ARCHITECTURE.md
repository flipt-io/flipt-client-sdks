# Architecture

The [flipt-engine](./flipt-engine/) is a Rust library responsible for evaluating context and returning the results of the evaluation.

The client engine polls for evaluation state from the Flipt server and uses this state to determine the results of the evaluation. The client engine is designed to be embedded in the native language client SDKs. The native language client SDKs will send context to the client engine via [FFI](https://en.wikipedia.org/wiki/Foreign_function_interface) and receive the results of the evaluation from engine.

This design allows for the client evaluation logic to be written once in a memory safe language (Rust) and embedded in the native language client SDKs.

You can refer to the architecture diagram below:

<p align="center">
    <img src="./diagrams/architecture.png" alt="Client SDKs Architecture" width="500px" />
</p>
