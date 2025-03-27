# Flipt React SDK

[![flipt-client-react](https://img.shields.io/npm/v/@flipt-io/flipt-client-react?label=%40flipt-io%2Fflipt-client-react)](https://www.npmjs.com/package/@flipt-io/flipt-client-react)

The Flipt React SDK provides a convenient way to integrate [Flipt](https://flipt.io) feature flags into your React applications using a custom hook and a context provider. It is built on top of the Flipt [client-side evaluation](https://www.flipt.io/docs/integration/client) client for the browser.

## Installation

```bash
npm install @flipt-io/flipt-client-react
```

## Usage

There are three ways to use the Flipt React SDK.

1. Use the `useFliptBoolean` hook for boolean evaluation in a functional component that is wrapped in a `FliptProvider`.
2. Use the `useFliptVariant` hook for variant evaluation in a functional component that is wrapped in a `FliptProvider`.
3. Use the `useFliptSelector` hook for custom evaluation in a functional component that is wrapped in a `FliptProvider`.

<!-- prettier-ignore-start -->

> [!TIP]
> Looking for NextJS support? Check out our [NextJS examples](https://github.com/flipt-io/flipt/tree/main/examples/nextjs) in the main Flipt repo.

<!-- prettier-ignore-end -->

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

The `useFliptBoolean` hook simplifies the process of evaluating a boolean feature flag. This hook must be used within a functional component that is wrapped by the `FliptProvider` context.
A default value is returned if the feature flag cannot be evaluated at the current time (e.g., due to network issues or missing data).

```tsx
import { useFliptBoolean } from '@flipt-io/flipt-client-react';

function MyComponent() {
  const result = useFliptBoolean('my-flag', false, 'user-123', {
    // additional context
  });
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

The `useFliptVariant` hook simplifies the process of evaluating a variant feature flag. This hook must be used within a functional component that is wrapped by the `FliptProvider` context.
A default value is returned if the feature flag cannot be evaluated at the current time (e.g., due to network issues or missing data).

```tsx
import { useFliptVariant } from '@flipt-io/flipt-client-react';

function MyComponent() {
  const result = useFliptVariant('my-flag', 'fallback', 'user-123', {
    // additional context
  });
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

The `useFliptSelector` hook allows direct access to the Flipt client so that it can be used in a functional component that is wrapped in a `FliptProvider`.

This is useful for more complex evaluations or in cases where you wish to call other methods on the Flipt client.

<!-- prettier-ignore-start -->

> [!WARNING]
> `flipt-client-react` heavily depends on the `useSyncExternalStore` hook which has some [caveats](https://react.dev/reference/react/useSyncExternalStore#caveats)

<!-- prettier-ignore-end -->

```tsx
import { useFliptSelector } from '@flipt-io/flipt-client-react';

function MyComponent() {
  const result = useFliptSelector((client, isLoading, error) => {
    const result = client?.evaluateBoolean('my-flag', 'user-123', {
      // additional context
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
  - `updateInterval`: The polling interval for fetching new state from Flipt. Set to `120` seconds by default. A `0` value disables polling completely after the initial fetch.
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

// A custom component that renders its children if the feature flag is enabled
function FeatureFlag({ flagKey, entityId, children }) {
  const isEnabled = useFliptBoolean('flagKey', false, entityId, {
    // additional context
  });
  return isEnabled ? children : null;
}

function App() {
  return (
    <FliptProvider options={{ url: 'https://your-flipt-instance.com' }}>
      <h1>My App</h1>
      {/* Will render the children if the feature flag evaluation results in true */}
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
