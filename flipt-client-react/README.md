# Flipt React SDK

[![npm](https://img.shields.io/npm/v/@flipt-io/flipt-client-react?label=%40flipt-io%2Fflipt-client-react)](https://www.npmjs.com/package/@flipt-io/flipt-client-react)

The Flipt React SDK provides a convenient way to integrate [Flipt](https://flipt.io) feature flags into your React applications using a custom hook and a context provider. It is built on top of the Flipt [client-side evaluation](https://www.flipt.io/docs/integration/client) client for the browser.

## Installation

```bash
npm install @flipt-io/flipt-client-react
```

## Usage

There are two ways to use the Flipt React SDK.

1. Use the `useFliptContext` hook in a functional component that is wrapped in a `FliptProvider`.
2. Use the `useFliptClient` hook in a functional component that is **not** wrapped in a `FliptProvider`.

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

#### useFliptContext Hook

The `useFliptContext` hook is a convenience hook that returns the Flipt client so that it can be used in a functional component that is wrapped in a `FliptProvider`.

```tsx
import { useFliptContext } from '@flipt-io/flipt-client-react';

function MyComponent() {
  const { client, loading, error } = useFliptContext();

  if (loading) {
    return <div>Loading Flipt client...</div>;
  }

  if (error) {
    return <div>Error initializing Flipt client: {error.message}</div>;
  }

  // Use the client to evaluate flags
  const handleCheckFlag = async () => {
    const result = await client.evaluateBoolean({
      flagKey: 'my-flag',
      entityId: 'user-123',
      context: {
        /* additional context */
      }
    });
    console.log('Flag evaluation result:', result);
  };

  return (
    <div>
      <button onClick={handleCheckFlag}>Check Flag</button>
    </div>
  );
}
```

This is useful for evalating flags throughtout your application without having to initialize the Flipt client in each component.

### useFliptClient Hook

The `useFliptClient` hook is similar to the `useFliptContext` hook but it returns the Flipt client so that it can be used in a functional component that is **not** wrapped in a `FliptProvider`.

This is useful for one-off evaluations or in cases where the Flipt client is not needed in the component tree.

```tsx
import { useFliptClient } from '@flipt-io/flipt-client-react';

function MyComponent() {
  const { client, loading, error } = useFliptClient(
    namespace: 'default',
    options: {
      url: 'https://your-flipt-instance.com',
      // Add other configuration options as needed
    }
  );

  if (loading) {
    return <div>Loading Flipt client...</div>;
  }

  if (error) {
    return <div>Error initializing Flipt client: {error.message}</div>;
  }

  // Use the client to evaluate flags
  const handleCheckFlag = async () => {
    const result = await client.evaluateBoolean({
      flagKey: 'my-flag',
      entityId: 'user-123',
      context: { /* additional context */ },
    });
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
  - `reference`: The [reference](https://docs.flipt.io/guides/user/using-references) to use when fetching flag state. If not provided, reference will not be used.
  - `fetcher`: An implementation of a [fetcher](https://github.com/flipt-io/flipt-client-sdks/blob/4821cb227c6c8b10419b96674d44ad1d6668a647/flipt-client-browser/src/models.ts#L5) interface to use when requesting flag state. If not provided, a default fetcher using the browser's [`fetch` API](https://developer.mozilla.org/en-US/docs/Web/API/Fetch_API) will be used.

### Authentication

The `FliptProvider` component supports the following authentication strategies:

- No Authentication (default)
- [Client Token Authentication](https://docs.flipt.io/authentication/using-tokens)
- [JWT Authentication](https://docs.flipt.io/authentication/using-jwts)

## State Management

The `FliptProvider` component pulls flag state from the Flipt instance at the `url` provided in the `options` object on instantiation.

To update the flag state, you can call the `refresh` method on the `FliptEvaluationClient` class.

```typescript
// Refresh the flag state
client.refresh();
```

## Examples

Here's a more complete example of how to use the Flipt React SDK in your application:

```tsx
import React from 'react';
import { FliptProvider, useFliptContext } from '@flipt-io/flipt-client-react';

function FeatureFlag({ flagKey, entityId, children }) {
  const { client, loading, error } = useFliptContext();
  const [isEnabled, setIsEnabled] = React.useState(false);

  React.useEffect(() => {
    if (client) {
      client
        .evaluateBoolean({
          flagKey,
          entityId,
          context: {
            /* additional context */
          }
        })
        .then((result) => {
          setIsEnabled(result.enabled);
        });
    }
  }, [client, flagKey, entityId]);

  if (loading) return <div>Loading...</div>;
  if (error) return <div>Error: {error.message}</div>;

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
