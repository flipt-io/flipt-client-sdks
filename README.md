# Flipt Client SDKs

![Status: Experimental](https://img.shields.io/badge/status-experimental-yellow)

This repository centralizes the client-side SDKs for [Flipt](https://github.com/flipt-io/flipt).

These client-side SDKs are responsible for evaluating context and returning the results of the evaluation. They enable developers to easily integrate Flipt into their applications without relying on server-side evaluation.

> [!WARNING]
> These SDKs are currently experimental. We are looking for feedback on the design and implementation. Please open an issue if you have any feedback or questions.

## Architecture

The client SDKs are designed to be embedded in end-user applications.

The evaluation logic is written in Rust and can be found in the [flipt-engine](./flipt-engine/) directory. The language clients that are used in end-user applications wrap the engine can be found in the `flipt-client-{language}` directories.

See [ARCHITECTURE.md](./ARCHITECTURE.md).

## Language Support

We are constantly growing our list of clients. Currently, we support the following languages:

1. [Go](./flipt-client-go)
1. [Python](./flipt-client-python)
1. [Ruby](./flipt-client-ruby)
1. [TypeScript](./flipt-client-node)

Languages we are planning to support:

1. Java
1. Rust
1. C#
1. PHP

## Installation

See each client's README for installation instructions.

Currently, you'll need to build the dynamic library and the library locally and install it for your architecture. This is a temporary solution until we package and distribute the libraries using their respective package managers.

## Use Cases

Why you may prefer to use a client-side SDK over our server-side SDKs:

1. You want extreme low-latency evaluation and high throughput.
1. You are ok with eventual consistency and can tolerate stale data for a short period of time.
1. You want to reduce the load in your network by not having each client make a request to the Flipt server for each evaluation.
1. You need evaluation to take place in process for some reason (e.g. you are evaluating a flag in a web worker).

## Performance

We have done some simple benchmarking to test the performance of the client SDKs vs the server SDKs with Flipt running locally.

![Performance Benchmarks](.github/images/performance.png)

Here we performed 1000 evaluations of a flag using the client SDKs and the server SDKs. The client SDKs were able to perform the evaluations in a fraction of the time it took the server SDKs. This is because the client SDKs are able to perform the evaluations in-memory without having to make a request to the Flipt server.

While the server SDKs performed evaluations in the range of 0-14ms, the client SDKs performed evaluations in the range of 0-0.1ms (100 microseconds).

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md).

### Help Wanted

We are not Rust experts, and are constantly learning. If you see something that can be improved especially in the [flipt-engine](./flipt-engine/), please open an issue or a PR, we would love to learn from you. :heart:

## License

All code in this repository is licensed under the MIT License. See [LICENSE](./LICENSE).

## Acknowledgements

- [Unleash/yggdrasil](https://github.com/Unleash/yggdrasil) - While we independently decided upon using Rust + FFI as the engine for the client SDKs, we were inspired by the design of the yggdrasil project from Unleash.
