# Flipt React SDK

[![npm](https://img.shields.io/npm/v/@flipt-io/flipt-client-react?label=%40flipt-io%2Fflipt-client-react)](https://www.npmjs.com/package/@flipt-io/flipt-client-react)

The Flipt React SDK provides a convenient way to integrate [Flipt](https://flipt.io) feature flags into your React applications using a custom hook and a context provider. It is built on top of the Flipt [client-side evaluation](https://www.flipt.io/docs/integration/client) client for the browser.

## Installation

```bash
npm install @flipt-io/flipt-client-react
```

> [!IMPORTANT]
> The Flipt React SDK does not currently work with Next.js App Router because Next.js App Router does not support WASM dependencies. See [this issue](https://github.com/vercel/next.js/issues/55537) for more information.

## Usage

There are two ways to use the Flipt React SDK.

1. Use the `useFliptBoolean` hook for boolean evaluation in a functional component that is wrapped in a `FliptProvider`.
2. Use the `useFliptVariant` hook for variant evaluation in a functional component that is wrapped in a `FliptProvider`.
3. Use the `useFliptSelector` hook for custom evaluation in a functional component that is wrapped in a `FliptProvider`.

### FliptProvider

First, wrap your application or a part of it with the `FliptProvider`:

```tsx
import { FliptProvider } from '@flipt-io/flipt-client-react';

function App() {
  return (
    <FliptProvider
      namespace="default"
      options={{
        url: 'https://your-flipt-instance.com'
        // Add other configuration options as needed
      }}
    >
      {/* Your app components */}
    </FliptProvider>
  );
}
```

#### useFliptBoolean Hook

The `useFliptBoolean` hook is a convenience hook that returns the boolean evaluation that it can be used in a functional component that is wrapped in a `FliptProvider`. `useFliptBoolean` requires the fallback value if it's impossible to evaluate the flag at the moment.

```tsx
import { useFliptBoolean } from '@flipt-io/flipt-client-react';

function MyComponent() {
  const result = useFliptBoolean('my-flag', false, 'user-123', {
    /*additional context*/
  });
  // Use the client to evaluate flags
  const handleCheckFlag = async () => {
    console.log('Flag evaluation result:', result);
  };

  return (
    <div>
      <button onClick={handleCheckFlag}>Check Flag</button>
    </div>
  );
}
```

#### useFliptVariant Hook

The `useFliptVariant` hook is a convenience hook that returns the variant evaluation that it can be used in a functional component that is wrapped in a `FliptProvider`. `useFliptVariant` requires the fallback value if it's impossible to evaluate the flag at the moment.

```tsx
import { useFliptVariant } from '@flipt-io/flipt-client-react';

function MyComponent() {
  const result = useFliptVariant('my-flag', 'fallback', 'user-123', {
    /*additional context*/
  });
  // Use the client to evaluate flags
  const handleCheckFlag = async () => {
    console.log('Flag evaluation result:', result);
  };

  return (
    <div>
      <button onClick={handleCheckFlag}>Check Flag</button>
    </div>
  );
}
```

### useFliptSelector Hook

The `useFliptSelector` hook allows the direct access to the Flipt client so that it can be used in a functional component that is wrapped in a `FliptProvider`.

This is useful for more complex evaluations or in cases where the Flipt client other methods should be called.

> [!CAUTION]
> `flipt-client-react` heavily depends on the `useSyncExternalStore` and it has the same [caveats](https://react.dev/reference/react/useSyncExternalStore#caveats)

```tsx
import { useFliptSelector } from '@flipt-io/flipt-client-react';

function MyComponent() {
  const result = useFliptSelector((client, isLoading, error) => {
    const result = client?.evaluateBoolean('my-flag', 'user-123', {
      /* additional context */
    });
    console.log('Flag evaluation internals:', result, isLoading, error);
    return result?.enabled;
  });

  // Use the client to evaluate flags
  const handleCheckFlag = async () => {
    console.log('Flag evaluation result:', result);
  };

  return (
    <div>
      <button onClick={handleCheckFlag}>Check Flag</button>
    </div>
  );
}
```

### Initialization Arguments

The `FliptProvider` component accepts two optional arguments:

- `namespace`: The namespace to fetch flag state from. If not provided, the client will default to the `default` namespace.
- `options`: An instance of the `ClientOptions` type that supports several options for the client. The structure is:
  - `url`: The URL of the upstream Flipt instance. If not provided, the client will default to `http://localhost:8080`.
  - `authentication`: The authentication strategy to use when communicating with the upstream Flipt instance. If not provided, the client will default to no authentication. See the [Authentication](#authentication) section for more information.
  - `updateInterval`: The polling interval for fetching new state from Flipt. Default set `120` seconds. To disable the polling please use `0`.
  - `reference`: The [reference](https://docs.flipt.io/guides/user/using-references) to use when fetching flag state. If not provided, reference will not be used.
  - `fetcher`: An implementation of a [fetcher](https://github.com/flipt-io/flipt-client-sdks/blob/4821cb227c6c8b10419b96674d44ad1d6668a647/flipt-client-browser/src/models.ts#L5) interface to use when requesting flag state. If not provided, a default fetcher using the browser's [`fetch` API](https://developer.mozilla.org/en-US/docs/Web/API/Fetch_API) will be used.

### Authentication

The `FliptProvider` component supports the following authentication strategies:

- No Authentication (default)
- [Client Token Authentication](https://docs.flipt.io/authentication/using-tokens)
- [JWT Authentication](https://docs.flipt.io/authentication/using-jwts)

## Examples

Here's a more complete example of how to use the Flipt React SDK in your application:

```tsx
import React from 'react';
import { FliptProvider, useFliptBoolean } from '@flipt-io/flipt-client-react';

function FeatureFlag({ flagKey, entityId, children }) {
  const isEnabled = useFliptBoolean('flagKey', false, entityId, {
    /*additional context*/
  });
  return isEnabled ? children : null;
}

function App() {
  return (
    <FliptProvider options={{ url: 'https://your-flipt-instance.com' }}>
      <h1>My App</h1>
      <FeatureFlag flagKey="new-feature" entityId="user-123">
        <div>This is a new feature!</div>
      </FeatureFlag>
    </FliptProvider>
  );
}

export default App;
```

## Contributing

Contributions are welcome! Please feel free to open an issue or submit a Pull Request.

## License

This project is licensed under the [MIT License](LICENSE).
