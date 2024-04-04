# Flipt Client Node

[![npm](https://img.shields.io/npm/v/@flipt-io/flipt-client?label=%40flipt-io%2Fflipt-client)](https://www.npmjs.com/package/@flipt-io/flipt-client)

The `flipt-client-node` library contains the JavaScript/TypeScript source code for the Flipt [client-side evaluation](https://www.flipt.io/docs/integration/client) client.

## Installation

```bash
npm install @flipt-io/flipt-client
```

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

## Memory Management

Since TypeScript/JavaScript is a garbage collected language there is no concept of "freeing" memory. We have to allocate memory for the engine through the `initialize_engine` FFI call.

Make sure to call the `close` method on the `FliptEvaluationClient` class once you are done using it.

```typescript
fliptEvaluationClient.close();
```

## Potential Issues

### Node v21+

If you are using Node v21 or higher, you'll likely run into issues with the `ffi-napi` library. This is due to the fact that `ffi-napi` does not seem to be compatible with Node v21+.

See: [node-ffi-napi #267](https://github.com/node-ffi-napi/node-ffi-napi/issues/267)

To work around this issue you'll need to downgrade to Node v20 or lower. :(

We plan to look into alternative ffi libraries for node in the future. See [#200](https://github.com/flipt-io/flipt-client-sdks/issues/200)

### Vitest

If you are using this library in combination with [vitest](https://github.com/vitest-dev/vitest), you may run into segmentation fault issues. This is due to the fact that `vitest` does not work nicely with `ffi-napi` by default.

See the following issue for more information: [vitest #2091: vitest will crash when using ffi-napi](https://github.com/vitest-dev/vitest/issues/2091).

To work around this issue, it is recommended to run your tests with the `--pool=forks` flag.

See [#169](https://github.com/flipt-io/flipt-client-sdks/issues/169) for more information.
