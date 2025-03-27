# Flipt Client SDKs

[![GitHub license](https://img.shields.io/github/license/flipt-io/flipt-client-sdks)](https://github.com/flipt-io/flipt-client-sdks/blob/main/LICENSE)

This repository centralizes the client-side SDKs for [Flipt](https://github.com/flipt-io/flipt).

These client-side SDKs are responsible for evaluating context and returning the results of the evaluation. They enable developers to easily integrate Flipt into their applications without relying on server-side evaluation.

Overall documentation for the client SDKs can be found on our [website](https://www.flipt.io/docs/integration/client).

Also, check out our introductory [blog post](https://www.flipt.io/blog/new-client-side-evaluation) on these client-side SDKs.

## Versions

There are two architectures for the client SDKs:

### FFI

The [Foreign Function Interface (FFI)](https://en.wikipedia.org/wiki/Foreign_function_interface) versions of the client SDKs are currently available.

#### Supported Architectures

The FFI-based SDKs are currently supported on the following OSes/architectures:

- Linux x86_64
- Linux arm64
- MacOS x86_64
- MacOS arm64
- Windows x86_64
- iOS (Swift)
- Android (Kotlin)

### WASM

The [WebAssembly (WASM)](https://webassembly.org/) versions of the client SDKs are OS and architecture agnostic.

## Architecture

The client SDKs are designed to be embedded in end-user applications.

The evaluation logic is written in Rust and can be found in the [flipt-evaluation](./flipt-evaluation/) directory.

The language clients used in end-user applications wrap the engines can be found in the `flipt-client-{language}` directories.

See [ARCHITECTURE.md](./ARCHITECTURE.md).

## Language Support

We are constantly growing our list of supported languages.

### Released

Currently, we support the following languages/platforms:

| Language/Platform                              | Version                                                                                                                                                                                         | Implementation |
| ---------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------- |
| [Android](./flipt-client-kotlin-android)       | [![flipt-client-android](https://img.shields.io/maven-central/v/io.flipt/flipt-client-android?label=flipt-client-android)](https://central.sonatype.com/artifact/io.flipt/flipt-client-android) | FFI            |
| [C#](./flipt-client-csharp)                    | [![flipt-client-c#](https://img.shields.io/nuget/v/flipt.client)](https://www.nuget.org/packages/Flipt.Client/)                                                                                 | FFI            |
| [Flutter/Dart](./flipt-client-dart)            | [![flipt-client-dart](https://img.shields.io/pub/v/flipt_client.svg)](https://pub.dev/packages/flipt_client)                                                                                    | FFI            |
| [Go](./flipt-client-go)                        | [![flipt-client-go](https://img.shields.io/github/v/tag/flipt-io/flipt-client-go?filter=v*&label=flipt-client-go)](https://github.com/flipt-io/flipt-client-go)                                 | WASM           |
| [Java](./flipt-client-java)                    | [![flipt-client-java](https://img.shields.io/maven-central/v/io.flipt/flipt-client-java?label=flipt-client-java)](https://central.sonatype.com/artifact/io.flipt/flipt-client-java)             | FFI            |
| [JavaScript (Browser)](./flipt-client-browser) | [![flipt-client-browser](https://img.shields.io/npm/v/@flipt-io/flipt-client-browser?label=%40flipt-io%2Fflipt-client-browser)](https://www.npmjs.com/package/@flipt-io/flipt-client-browser)   | WASM           |
| [NodeJS](./flipt-client-node)                  | [![flipt-client-node](https://img.shields.io/npm/v/@flipt-io/flipt-client?label=%40flipt-io%2Fflipt-client)](https://www.npmjs.com/package/@flipt-io/flipt-client)                              | WASM           |
| [Python](./flipt-client-python)                | [![flipt-client-python](https://img.shields.io/pypi/v/flipt-client.svg)](https://pypi.org/project/flipt-client)                                                                                 | FFI            |
| [React Web (Browser)](./flipt-client-react)    | [![flipt-client-react](https://img.shields.io/npm/v/@flipt-io/flipt-client-react?label=%40flipt-io%2Fflipt-client-react)](https://www.npmjs.com/package/@flipt-io/flipt-client-react)           | WASM           |
| [Ruby](./flipt-client-ruby)                    | [![flipt-client-ruby](https://badge.fury.io/rb/flipt_client.svg)](https://badge.fury.io/rb/flipt_client)                                                                                        | FFI            |
| [Swift](./flipt-client-swift)                  | [![flipt-client-swift](https://img.shields.io/github/v/tag/flipt-io/flipt-client-swift?filter=v*&label=flipt-client-swift)](https://github.com/flipt-io/flipt-client-swift)                     | FFI            |

Documentation for each client can be found in the README of that client's directory.

### Planned

Languages we are planning to support:

1. [Rust](https://github.com/flipt-io/flipt-client-sdks/issues/83)

### Help Wanted

Languages we would like to support but lack expertise in:

1. [React Native](https://github.com/flipt-io/flipt-client-sdks/issues/345)

Want to see a client in a language we don't support? [Open an issue](https://github.com/flipt-io/flipt-client-sdks/issues/new?assignees=&labels=new-language&projects=&template=new_language.yml) and let us know!

Alternatively, if you have experience in any of the above languages/platforms we welcome all contributions!! ❤️

## Installation

Please take a look at each client's README for installation and usage instructions.

## Use Cases

Why you may prefer to use a client-side SDK over our server-side SDKs:

1. You want extremely low-latency evaluation and high throughput.
1. You are ok with eventual consistency and can tolerate stale data for a short time.
1. You want to reduce the load in your network by not having each client make a request to the Flipt server for each evaluation.
1. You need evaluation to occur in-process for some reason (e.g. you are evaluating a flag in a web worker).

## Performance

We have done some simple benchmarking to test the performance of the client SDKs vs the server SDKs with Flipt running locally.

![Performance Benchmarks](.github/images/performance.png)

Here we performed 1000 evaluations of a flag using the client SDKs and the server SDKs. The client SDKs were able to perform the evaluations in a fraction of the time it took the server SDKs. This is because the client SDKs can perform the evaluations in memory without having to make a request to the Flipt server.

While the server SDKs performed evaluations in the range of 0-14ms, the client SDKs performed evaluations in the range of 0-0.1ms (100 microseconds).

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md).

### Help Wanted

We are not Rust experts and are constantly learning. If you see something that can be improved, especially in the [flipt-engine-ffi](./flipt-engine-ffi/) and [flipt-evaluation](./flipt-evaluation/) directories, please open an issue or a PR, we would love to learn from you. :heart:

## License

All code in this repository is licensed under the [MIT License](./LICENSE).

## Acknowledgements

- [Unleash/yggdrasil](https://github.com/Unleash/yggdrasil) - While we independently decided upon using Rust + FFI as the engine for the client SDKs, we were inspired by the design of the yggdrasil project from Unleash.
