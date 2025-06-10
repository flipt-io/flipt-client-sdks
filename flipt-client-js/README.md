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

### Initialization Arguments

The `FliptClient` constructor accepts the following optional arguments:

- `options`: An instance of the `ClientOptions` type that supports several options for the client. The structure is:
  - `environment`: The environment to use when evaluating flags (Flipt v2). If not provided, the client will default to the `default` environment.
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

## Slim Module

The `slim` module is designed for advanced or non-standard environments where you need full control over how the WASM binary is loaded. Unlike the default client, which tries to load the WASM file automatically, the slim client requires you to explicitly provide the WASM file (as a URL, buffer, etc.).

**Why use the slim module?**

- **Edge/Serverless/Custom Environments:** Some platforms (like Cloudflare Workers, Vercel Edge, or certain Node.js setups) restrict how files are loaded or bundled. The slim client lets you load the WASM file in a way that works for your environment.
- **Full Control:** You decide how and when the WASM file is loaded—fetch from a CDN, load from disk, or use a custom loader.
- **Compatibility:** The slim client removes all assumptions about WASM loading, making it compatible with any environment where you can provide the WASM binary.
- **Bundle Size:** The slim build may exclude some code related to WASM loading, making it lighter and more tree-shakable for certain use cases.

If you are using a standard browser app with a bundler, the default client is usually easier. For edge, serverless, or custom setups, the slim client is the right tool.

### Using the `@flipt-io/flipt-client-js/slim` Module

The `slim` build of the FliptClient is designed for environments where you want to provide your own WASM binary, such as modern browsers (with a bundler) or Node.js.

#### Browser Usage (with a Bundler like Vite, Webpack, etc.)

1. **Import and initialize the client:**

   ```ts
   import { FliptClient } from '@flipt-io/flipt-client-js/slim';
   import wasmUrl from '@flipt-io/flipt-client-js/engine.wasm';

   async function main() {
     const client = await FliptClient.init(
       {
         namespace: 'default',
         url: 'http://localhost:8080'
       },
       { wasm: wasmUrl }
     );

     // Use the client as normal
     const result = client.evaluateBoolean({
       flagKey: 'my-flag',
       entityId: 'user-123',
       context: { country: 'US' }
     });

     console.log(result);
   }

   main();
   ```

2. **Ensure your bundler is configured to handle `.wasm` files.**
   - For Vite, this works out of the box.
   - For Webpack, you may need to add a rule for `.wasm` files.

---

#### Node.js Usage

1. **Import and initialize the client:**

   ```ts
   import { FliptClient } from '@flipt-io/flipt-client-js/slim';
   import { readFileSync } from 'fs';
   import path from 'path';
   import { fileURLToPath } from 'url';

   // Resolve the path to the WASM file
   const __filename = fileURLToPath(import.meta.url);
   const __dirname = path.dirname(__filename);
   const wasmPath = path.join(
     __dirname,
     '../node_modules/@flipt-io/flipt-client-js/engine.wasm'
   );
   const wasmBuffer = readFileSync(wasmPath);

   async function main() {
     const client = await FliptClient.init(
       {
         namespace: 'default',
         url: 'http://localhost:8080'
       },
       { wasm: wasmBuffer }
     );

     // Use the client as normal
     const result = client.evaluateBoolean({
       flagKey: 'my-flag',
       entityId: 'user-123',
       context: { country: 'US' }
     });

     console.log(result);
   }

   main();
   ```

   - **Note:** In Node.js, you must load the WASM file as a buffer using `fs.readFileSync`.

2. **Run your script with a loader that supports ESM and top-level await, such as `ts-node` or `tsx`:**

   ```sh
   npx tsx src/index.ts
   ```

---

### Additional Notes

- The `slim` build requires you to provide the WASM binary via the `wasm` option.
- For browser usage, pass the URL to the WASM file (using your bundler's asset handling).
- For Node.js, pass the WASM file as a buffer.
- Make sure your environment supports top-level `await` or wrap your code in an async function.

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
