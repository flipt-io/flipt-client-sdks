# Client SDKs

The intention of this repo is to centralize the core evaluation logic for Flipt's feature flags, and have thin multi-language wrappers around that logic.

The evaluation logic is written in Rust and can be found in the `flipt-engine` directory. The language clients that wrap the engine can be found in the `flipt-client-{language}` directories.

The Rust core is compiled down into a dynamically-linked library and the language clients are able to access that logic through [FFI (foreign function interface)](https://levelup.gitconnected.com/what-is-ffi-foreign-function-interface-an-intuitive-explanation-7327444e347a).

You can refer to the architecture diagram below:

<img src="./diagrams/architecture.png" alt="Client SDKs Architecture" width="500px" />

## Language Support

We are constantly growing our list of clients. Currently, we support the following languages:

1. [Golang](./flipt-client-go)
1. [Python](./flipt-client-python)
1. [Ruby](./flipt-client-ruby)
1. [TypeScript](./flipt-client-node)
