# Flipt Client Node

> [!WARNING]
> This SDK is deprecated and will not be maintained going forward. Please use the new [Flipt Client JS SDK](https://github.com/flipt-io/flipt-client-sdks/tree/main/flipt-client-js) instead.

[![flipt-client-node](https://img.shields.io/npm/v/@flipt-io/flipt-client?label=%40flipt-io%2Fflipt-client)](https://www.npmjs.com/package/@flipt-io/flipt-client)
![deprecated](https://img.shields.io/badge/status-deprecated-red)

The `flipt-client-node` library contains the JavaScript/TypeScript source code for the Flipt [client-side evaluation](https://www.flipt.io/docs/integration/client) client.

## Installation

```bash
npm install @flipt-io/flipt-client
```

## Usage

In your Node.js code you can import this client and use it as so:

```typescript
import { FliptEvaluationClient } from '@flipt-io/flipt-client';

// namespace is the first positional argument and is optional here and will have a value of "default" if not specified.
// options is the second positional argument and is also optional, the structure is:
// {
//  "url": "http://localhost:8080",
//  "updateInterval": 120,
//  "authentication": {
//    "clientToken": "secret"
//  }
// }
//
// You can replace the url with where your upstream Flipt instance points to, the updateInterval for how long you are willing
// to wait for updated flag state, and the auth token if your Flipt instance requires it.
const fliptEvaluationClient = await FliptEvaluationClient.init('default', {
  url: 'http://localhost:8080',
  authentication: {
    clientToken: 'secret'
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
- `options`: An instance of the `ClientOptions` type that supports several options for the client. The structure is:
  - `url`: The URL of the upstream Flipt instance. If not provided, the client will default to `http://localhost:8080`.
  - `updateInterval`: The interval (in seconds) in which to fetch new flag state. If not provided, the client will default to 120 seconds.
  - `authentication`: The authentication strategy to use when communicating with the upstream Flipt instance. If not provided, the client will default to no authentication. See the [Authentication](#authentication) section for more information.
  - `reference`: The [reference](https://docs.flipt.io/guides/user/using-references) to use when fetching flag state. If not provided, reference will not be used.
  - `fetcher`: An implementation of a [fetcher](https://github.com/flipt-io/flipt-client-sdks/blob/bc1c19b553bf265f7f49ed4901d59f4202aff135/flipt-client-node/src/models.ts#L1-L16) interface to use when requesting flag state. If not provided, a default fetcher using Node.js's [`fetch` API](https://nodejs.org/en/learn/getting-started/fetch) will be used.
  - `errorStrategy`: The error strategy to use when fetching flag state. If not provided, the client will default to fail. See the [Error Strategies](#error- strategies) section for more information.

### Authentication

The `FliptEvaluationClient` supports the following authentication strategies:

- No Authentication (default)
- [Client Token Authentication](https://docs.flipt.io/authentication/using-tokens)
- [JWT Authentication](https://docs.flipt.io/authentication/using-jwts)

### Error Strategies

The client `errorStrategy` option supports the following error strategies:

- `fail`: The client will throw an error if the flag state cannot be fetched. This is the default behavior.
- `fallback`: The client will maintain the last known good state and use that state for evaluation in case of an error.

### Custom Fetcher

The `FliptEvaluationClient` supports custom fetchers. This allows you to fetch flag state from a custom source or override HTTP headers.

The fetcher can be passed in as an argument to the `FliptEvaluationClient` initializer function.

```typescript
const fliptEvaluationClient = await FliptEvaluationClient.init('default', {
  url: 'http://localhost:8080',
  authentication: {
    clientToken
  },
  fetcher: customFetcher
});
```

The fetcher is a function that takes an optional [`IFetcherOpts`](https://github.com/flipt-io/flipt-client-sdks/blob/bc1c19b553bf265f7f49ed4901d59f4202aff135/flipt-client-node/src/models.ts#L1-L16) argument and returns a `Promise` that resolves to a `Response` object.

## Timer Management

The `FliptEvaluationClient` class uses a timer to fetch new flag state at a regular interval.

Make sure to call the `close` method on the `FliptEvaluationClient` class once you are done using it to stop the timer and clean up resources.

```typescript
fliptEvaluationClient.close();
```

## State Management

The `FliptEvaluationClient` class pulls flag state from the Flipt instance at the `url` provided in the `options` object on instantiation.

This state is pulled from the Flipt instance on instantiation and every `update_interval` seconds thereafter.

To update the flag state manually, you can call the `refresh` method on the `FliptEvaluationClient` class.

> [!NOTE]
> The `refresh` method returns a boolean indicating whether the flag state changed.

```typescript
// Refresh the flag state
let changed = await fliptEvaluationClient.refresh();

if (changed) {
  // Do something
}
```

## ETag Support

The default fetcher uses [ETag HTTP headers](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/ETag) to reduce overhead building and sending previously observed snapshots. This is useful in scenarios where the flag state is not frequently updated and you want to reduce the load on the server.

To disable ETag support, you can implement a [custom fetcher](#custom-fetcher) that does not use ETags.

## Development

### WASM

This library uses a WebAssembly (WASM) layer to interact with the Flipt server. It is written in Rust and exposes a JavaScript API using the `wasm-bindgen` and `wasm-pack` tools. We wrap the built WASM layer in a JavaScript API to make it easier to use in a Node.js environment.

### Prerequisites

- [flipt-engine-wasm-js](../flipt-engine-wasm-js)

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
