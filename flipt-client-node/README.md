# Flipt Client Node

[![npm](https://img.shields.io/npm/v/@flipt-io/flipt-client?label=%40flipt-io%2Fflipt-client)](https://www.npmjs.com/package/@flipt-io/flipt-client)

The `flipt-client-node` library contains the JavaScript/TypeScript source code for the Flipt [client-side evaluation](https://www.flipt.io/docs/integration/client) client.

## Installation

```bash
npm install @flipt-io/flipt-client
```

## Supported Architectures

This SDK currently supports the following OSes/architectures:

- Linux x86_64
- Linux arm64
- MacOS x86_64
- MacOS arm64

## Usage

In your Node code you can import this client and use it as so:

```typescript
import { FliptEvaluationClient } from '@flipt-io/flipt-client';

// namespace is the first positional argument and is optional here and will have a value of "default" if not specified.
// engine_opts is the second positional argument and is also optional, the structure is:
// {
//  "url": "http://localhost:8080",
//  "update_interval": 120,
//  "authentication": {
//    "client_token": "secret"
//  }
// }
//
// You can replace the url with where your upstream Flipt instance points to, the update interval for how long you are willing
// to wait for updated flag state, and the auth token if your Flipt instance requires it.
const fliptEvaluationClient = new FliptEvaluationClient();

const variant = fliptEvaluationClient.evaluateVariant('flag1', 'someentity', {
  fizz: 'buzz'
});

console.log(variant);
```

### Constructor Arguments

The `FliptEvaluationClient` constructor accepts two optional arguments:

- `namespace`: The namespace to fetch flag state from. If not provided, the client will default to the `default` namespace.
- `engine_opts`: An instance of the `EngineOpts` type that supports several options for the client. The structure is:
  - `url`: The URL of the upstream Flipt instance. If not provided, the client will default to `http://localhost:8080`.
  - `update_interval`: The interval (in seconds) in which to fetch new flag state. If not provided, the client will default to 120 seconds.
  - `authentication`: The authentication strategy to use when communicating with the upstream Flipt instance. If not provided, the client will default to no authentication. See the [Authentication](#authentication) section for more information.
  - `reference`: The [reference](https://docs.flipt.io/guides/user/using-references) to use when fetching flag state. If not provided, reference will not be used.

### Authentication

The `FliptEvaluationClient` supports the following authentication strategies:

- No Authentication (default)
- [Client Token Authentication](https://docs.flipt.io/authentication/using-tokens)
- [JWT Authentication](https://docs.flipt.io/authentication/using-jwts)

## Memory Management

Since TypeScript/JavaScript is a garbage collected language there is no concept of "freeing" memory. We have to allocate memory for the engine through the `initialize_engine` FFI call.

Make sure to call the `close` method on the `FliptEvaluationClient` class once you are done using it.

```typescript
fliptEvaluationClient.close();
```

## Potential Issues

### Vitest

If you are using this library in combination with [vitest](https://github.com/vitest-dev/vitest), you may run into segmentation fault issues. This is due to the fact that `vitest` does not work nicely with `ffi-napi` by default.

See the following issue for more information: [vitest #2091: vitest will crash when using ffi-napi](https://github.com/vitest-dev/vitest/issues/2091).

To work around this issue, it is recommended to run your tests with the `--pool=forks` flag.

See [#169](https://github.com/flipt-io/flipt-client-sdks/issues/169) for more information.
