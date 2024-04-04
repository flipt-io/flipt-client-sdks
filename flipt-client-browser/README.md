# Flipt Client Browser

![Status: Beta](https://img.shields.io/badge/status-beta-yellow)
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

## State Management

The `FliptEvaluationClient` class pulls flag state from the Flipt instance at the `url` provided in the `engine_opts` object on instantiation.

To update the flag state, you can call the `refresh` method on the `FliptEvaluationClient` class.

```typescript
// Refresh the flag state
fliptEvaluationClient.refresh();
```

This allows you to update the flag state in a controlled manner, such as in a polling loop or when a user interacts with your application.

## Development

### WASM

This library uses a WebAssembly (WASM) layer to interact with the Flipt server. It is written in Rust and exposes a JavaScript API using the `wasm-bindgen` and `wasm-pack` tools. We wrap the built WASM layer in a JavaScript API to make it easier to use in a browser environment.

## Architecture

See [ARCHITECTURE.md](./ARCHITECTURE.md).

### Prerequisites

- [Rust](https://www.rust-lang.org/tools/install)
- [wasm-pack](https://rustwasm.github.io/wasm-pack/installer/)
- [wasm-bindgen](https://rustwasm.github.io/wasm-bindgen/)

### Build

```bash
npm run build
```

### Test

```bash
npm install
npm test
```
