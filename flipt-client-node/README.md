<!-- Last published: Wed Dec 27 22:18:09 UTC 2023 -->
# Flipt Client Node

[![npm](https://img.shields.io/npm/v/@flipt-io/flipt-client?label=%40flipt-io%2Fflipt-client)](https://www.npmjs.com/package/@flipt-io/flipt-client)

The `flipt-client-node` directory contains the TypeScript source code for the Flipt [client-side evaluation](https://www.flipt.io/docs/integration/client) client.

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
//  "auth_token": "secret"
// }
//
// You can replace the url with where your upstream Flipt instance points to, the update interval for how long you are willing
// to wait for updated flag state, and the auth token if your Flipt instance requires it.
const fliptEvaluationClient = new FliptEvaluationClient();

const variant = fliptEvaluationClient.evaluateVariant("flag1", "someentity", {"fizz": "buzz"});

console.log(variant);
```

## Memory Management

Since TypeScript/JavaScript is a garbage collected language there is no concept of "freeing" memory. We have to allocate memory for the engine through the `initialize_engine` FFI call.

Make sure to call the `close` method on the `FliptEvaluationClient` class once you are done using it.

```typescript
fliptEvaluationClient.close();
```
