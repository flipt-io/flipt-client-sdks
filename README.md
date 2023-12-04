# Client SDKs

![Status: Experimental](https://img.shields.io/badge/status-experimental-yellow)

This repository centralizes the client-side SDKs for [Flipt](https://github.com/flipt-io/flipt).

These client-side SDKs are responsible for evaluating context and returning the results of the evaluation. They enable developers to easily integrate Flipt into their applications without relying on server-side SDKs.

## Architecture

The client SDKs are designed to be embedded in end-user applications.

The evaluation logic is written in Rust and can be found in the `flipt-engine` directory. The language clients that are used in end-user applications wrap the engine can be found in the `flipt-client-{language}` directories.

See [ARCHITECTURE.md](./ARCHITECTURE.md).

## Language Support

We are constantly growing our list of clients. Currently, we support the following languages:

1. [Golang](./flipt-client-go)
1. [Python](./flipt-client-python)
1. [Ruby](./flipt-client-ruby)
1. [TypeScript](./flipt-client-node)
