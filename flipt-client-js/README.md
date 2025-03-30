# Flipt Client JS

[![flipt-client-js](https://img.shields.io/npm/v/@flipt-io/flipt-client-js?label=%40flipt-io%2Fflipt-client-js)](https://www.npmjs.com/package/@flipt-io/flipt-client-js)

The `flipt-client-js` library contains the JavaScript/TypeScript client for Flipt [client-side evaluation](https://www.flipt.io/docs/integration/client) in Web and Node.js environments. It supports:

- Node.js
- Browsers
- Edge Functions (Vercel Edge, Cloudflare Workers, etc.)
- Web Workers

## Installation

```bash
npm install @flipt-io/flipt-client-js
```

## Usage

```typescript
import { FliptClient } from '@flipt-io/flipt-client-js';

const client = await FliptClient.init({
  namespace: 'default',
  url: 'http://localhost:8080',
  authentication: {
    clientToken: 'secret'
  }
});

const variant = client.evaluateVariant({
  flagKey: 'flag1',
  entityId: 'someentity',
  context: {
    fizz: 'buzz'
  }
});

console.log(variant);
```

### Edge Function Usage (e.g., Vercel Edge)

```typescript
import { FliptClient } from '@flipt-io/flipt-client-js';

export const config = {
  runtime: 'edge'
};

export default async function middleware(req) {
  const client = await FliptClient.init({
    namespace: 'default',
    url: process.env.FLIPT_URL,
    authentication: {
      clientToken: process.env.FLIPT_AUTH_TOKEN
    }
  });

  const result = client.evaluateBoolean({
    flagKey: 'my-flag',
    entityId: 'user-123',
    context: {
      country: 'US'
    }
  });

  if (result.enabled) {
    return new Response('Feature enabled!');
  }

  return new Response('Feature disabled');
}
```

### Initialization Arguments

The `FliptClient` constructor accepts the following optional arguments:

- `options`: An instance of the `ClientOptions` type that supports several options for the client. The structure is:
  - `namespace`: The namespace to fetch flag state from. If not provided, the client will default to the `default` namespace.
  - `url`: The URL of the upstream Flipt instance. If not provided, the client will default to `http://localhost:8080`.
  - `updateInterval`: **Node.js Only** The interval (in seconds) in which to fetch new flag state. If not provided, the client will default to 120 seconds.
  - `authentication`: The authentication strategy to use when communicating with the upstream Flipt instance. If not provided, the client will default to no authentication. See the [Authentication](#authentication) section for more information.
  - `reference`: The [reference](https://docs.flipt.io/guides/user/using-references) to use when fetching flag state. If not provided, reference will not be used.
  - `fetcher`: An implementation of a [fetcher](https://github.com/flipt-io/flipt-client-sdks/blob/d0869dc9f6b3a65052712926b88fa0f9b18defaa/flipt-client-js/src/core/types.ts#L60-L62) interface to use when requesting flag state. If not provided, a default fetcher using the browser's [`fetch` API](https://developer.mozilla.org/en-US/docs/Web/API/Fetch_API) will be used.
  - `errorStrategy`: The error strategy to use when fetching flag state. If not provided, the client will default to fail. See the [Error Strategies](#error-strategies) section for more information.

### Authentication

The `FliptClient` supports the following authentication strategies:

- No Authentication (default)
- [Client Token Authentication](https://docs.flipt.io/authentication/using-tokens)
- [JWT Authentication](https://docs.flipt.io/authentication/using-jwts)

### Error Strategies

The client `errorStrategy` option supports the following error strategies:

- `fail`: The client will throw an error if the flag state cannot be fetched. This is the default behavior.
- `fallback`: The client will maintain the last known good state and use that state for evaluation in case of an error.

### Custom Fetcher

The `FliptClient` supports custom fetchers. This allows you to fetch flag state from a custom source or override HTTP headers.

The fetcher can be passed in as an argument to the `FliptClient` initializer function.

```typescript
const client = await FliptClient.init({
  namespace: 'default',
  url: 'http://localhost:8080',
  authentication: {
    clientToken
  },
  fetcher: customFetcher
});
```

The fetcher is a function that takes an optional [`IFetcherOpts`](https://github.com/flipt-io/flipt-client-sdks/blob/d0869dc9f6b3a65052712926b88fa0f9b18defaa/flipt-client-js/src/core/types.ts#L50-L55) argument and returns a `Promise` that resolves to a `Response` object.

## State Management

The `FliptClient` class pulls flag state from the Flipt instance at the `url` provided in the `options` object on instantiation.

To update the flag state, you can call the `refresh` method on the `FliptClient` class.

> [!NOTE]
> The `refresh` method returns a boolean indicating whether the flag state changed.

```typescript
// Refresh the flag state
let changed = await client.refresh();

if (changed) {
  // Do something
}
```

### Auto-Refresh (Node.js Only)

The `FliptClient` class supports auto-refreshing flag state.

To enable auto-refreshing, you can pass the `updateInterval` option to the `FliptClient` initializer function. Under the hood this uses a timer to fetch new flag state at a regular interval.

> [!NOTE]
> The `updateInterval` option is only supported in Node.js environments.

```typescript
const client = await FliptClient.init({
  namespace: 'default',
  url: 'http://localhost:8080',
  updateInterval: 30 // refresh every 30 seconds
});
```

Make sure to call the `close` method on the `FliptClient` class once you are done using it to stop the timer and clean up resources.

```typescript
client.close();
```

## ETag Support

The default fetcher uses [ETag HTTP headers](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/ETag) to reduce the number of requests to the Flipt instance. This is useful in scenarios where the flag state is not frequently updated and you want to reduce the load on the server.

To disable ETag support, you can implement a [custom fetcher](#custom-fetcher) that does not use ETags.

## Development

### WASM

This library uses a WebAssembly (WASM) layer to interact with the Flipt server. It is written in Rust and exposes a JavaScript API using the `wasm-bindgen` and `wasm-pack` tools. We wrap the built WASM layer in a JavaScript API to make it easier to use in a web environment.

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
