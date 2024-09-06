# Flipt Client Browser

![Status: Hardening](https://img.shields.io/badge/status-hardening-orange)
[![npm](https://img.shields.io/npm/v/@flipt-io/flipt-client-browser?label=%40flipt-io%2Fflipt-client-browser)](https://www.npmjs.com/package/@flipt-io/flipt-client-browser)

The `flipt-client-browser` library contains the JavaScript/TypeScript source code for the Flipt [client-side evaluation](https://www.flipt.io/docs/integration/client) client for the browser.

## Installation

```bash
npm install @flipt-io/flipt-client-browser
```

## Usage

In your JavaScript/Typescript code you can import this client and use it as so:

```typescript
import { FliptEvaluationClient } from '@flipt-io/flipt-client-browser';

// namespace is the first positional argument and is optional here and will have a value of "default" if not specified.
// engine_opts is the second positional argument and is also optional, the structure is:
// {
//  "url": "http://localhost:8080",
//  "authentication": {
//    "client_token": "secret"
//  }
// }
//
const fliptEvaluationClient = await FliptEvaluationClient.init('default', {
  url: 'http://localhost:8080',
  authentication: {
    client_token
  }
});

const variant = fliptEvaluationClient.evaluateVariant('flag1', 'someentity', {
  fizz: 'buzz'
});

console.log(variant);
```

### Initialization Arguments

The `FliptEvaluationClient` constructor accepts two optional arguments:

- `namespace`: The namespace to fetch flag state from. If not provided, the client will default to the `default` namespace.
- `engine_opts`: An instance of the `EngineOpts` type that supports several options for the client. The structure is:
  - `url`: The URL of the upstream Flipt instance. If not provided, the client will default to `http://localhost:8080`.
  - `authentication`: The authentication strategy to use when communicating with the upstream Flipt instance. If not provided, the client will default to no authentication. See the [Authentication](#authentication) section for more information.
  - `reference`: The [reference](https://docs.flipt.io/guides/user/using-references) to use when fetching flag state. If not provided, reference will not be used.
  - `fetcher`: An implementation of a [fetcher](https://github.com/flipt-io/flipt-client-sdks/blob/4821cb227c6c8b10419b96674d44ad1d6668a647/flipt-client-browser/src/models.ts#L5) interface to use when requesting flag state. If not provided, a default fetcher using the browser's [`fetch` API](https://developer.mozilla.org/en-US/docs/Web/API/Fetch_API) will be used.

### Authentication

The `FliptEvaluationClient` supports the following authentication strategies:

- No Authentication (default)
- [Client Token Authentication](https://docs.flipt.io/authentication/using-tokens)
- [JWT Authentication](https://docs.flipt.io/authentication/using-jwts)

### Custom Fetcher

The `FliptEvaluationClient` supports custom fetchers. This allows you to fetch flag state from a custom source or override HTTP headers.

The fetcher can be passed in as an argument to the `FliptEvaluationClient` initializer function.

```typescript
const fliptEvaluationClient = await FliptEvaluationClient.init('default', {
  url: 'http://localhost:8080',
  authentication: {
    client_token
  },
  fetcher: customFetcher
});
```

The fetcher is a function that takes an optional [`IFetcherOpts`](https://github.com/flipt-io/flipt-client-sdks/blob/4821cb227c6c8b10419b96674d44ad1d6668a647/flipt-client-browser/src/models.ts#L1) argument and returns a `Promise` that resolves to a `Response` object.

## State Management

The `FliptEvaluationClient` class pulls flag state from the Flipt instance at the `url` provided in the `engine_opts` object on instantiation.

To update the flag state, you can call the `refresh` method on the `FliptEvaluationClient` class.

```typescript
// Refresh the flag state
fliptEvaluationClient.refresh();
```

## ETag Support

The default fetcher uses [ETag HTTP headers](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/ETag) to reduce the number of requests to the Flipt instance. This is useful in scenarios where the flag state is not frequently updated and you want to reduce the load on the server.

To disable ETag support, you can implement a [custom fetcher](#custom-fetcher) that does not use ETags.

## Development

### WASM

This library uses a WebAssembly (WASM) layer to interact with the Flipt server. It is written in Rust and exposes a JavaScript API using the `wasm-bindgen` and `wasm-pack` tools. We wrap the built WASM layer in a JavaScript API to make it easier to use in a browser environment.

### Prerequisites

- [flipt-engine-wasm](../flipt-engine-wasm)

### Build

```bash
npm run build
```

### Test

```bash
npm install
npm test
```

## Contributing

Contributions are welcome! Please feel free to open an issue or submit a Pull Request.

## License

This project is licensed under the [MIT License](LICENSE).
