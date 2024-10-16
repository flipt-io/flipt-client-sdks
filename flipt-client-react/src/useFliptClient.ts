import { useContext, useSyncExternalStore, useCallback, createContext } from 'react';
import { FliptEvaluationClient, ClientOptions } from '@flipt-io/flipt-client-browser';

export interface FliptClientHook {
  client: FliptEvaluationClient | null;
  isLoading: boolean;
  error: Error | null;
}

export interface FliptStore extends FliptClientHook {
  subscribe: (onStoreChange: () => void) => () => void;
  attach: () => void;
  detach: () => void;
}

export const configureStore = (namespace: string, options: ClientOptions): FliptStore => {
  const listeners = new Set<() => void>();
  const subscribe = (listener: () => void): (() => void) => {
    listeners.add(listener);
    return () => listeners.delete(listener);
  };
  const notify = () => {
    listeners.forEach((l) => l());
  };
  let intervalId: any;
  let mounted: boolean = false;

  const store: FliptStore = {
    client: null,
    isLoading: true,
    error: null,
    subscribe,
    attach: () => {
      mounted = true;
      setupPolling();
    },
    detach: () => {
      mounted = false;
      clearInterval(intervalId);
      intervalId = undefined;
    },
  };
  const interval = options.updateInterval || 0;

  const setupPolling = () => {
    if (interval > 0 && mounted && store.client !== null && intervalId === undefined) {
      intervalId = setInterval(() => {
        if (typeof window !== 'undefined' && navigator.onLine) {
          store.client?.refresh().then((updated) => {
            if (updated) {
              notify();
            }
          });
        }
      }, interval);
    }
  };

  FliptEvaluationClient.init(namespace, options)
    .then((client) => {
      store.client = client;
      setupPolling();
    })
    .catch((err: Error) => {
      store.error = err;
    })
    .finally(() => {
      store.isLoading = false;
      notify();
    });

  return store;
};

export const FliptContext = createContext<FliptStore | null>(null);

export const useFliptContext = (): FliptClientHook => {
  const context = useContext(FliptContext);
  if (context === null) {
    throw new Error('useFliptContext must be used within a FliptProvider');
  }
  return context;
};

export const useFliptSelector = <T>(
  selector: (client: FliptEvaluationClient | null, isLoading: boolean, error: Error | null) => T
): T => {
  const store = useContext(FliptContext);
  if (store === null) {
    throw new Error('useFliptSelector must be used within a FliptProvider');
  }
  return useSyncExternalStore(
    store.subscribe,
    useCallback(() => selector(store.client, store.isLoading, store.error), [store, selector])
  );
};

export const useFliptBoolean = (
  flagKey: string,
  fallback: boolean,
  entityId: string,
  context: Record<string, string> = {}
): boolean => {
  const result = useFliptSelector((client, isLoading, error) => {
    if (client && !isLoading && !error) {
      try {
        return client.evaluateBoolean(flagKey, entityId, context).enabled;
      } catch (e) {
        console.error(`Error evaluating boolean flag ${flagKey}:`, e);
      }
    }
    return fallback;
  });

  return result;
};

export const useFliptVariant = (
  flagKey: string,
  fallback: string,
  entityId: string,
  context: Record<string, string> = {}
): string => {
  const result = useFliptSelector((client, isLoading, error) => {
    if (client && !isLoading && !error) {
      try {
        return client.evaluateVariant(flagKey, entityId, context).variantKey;
      } catch (e) {
        console.error(`Error evaluating boolean flag ${flagKey}:`, e);
      }
    }
    return fallback;
  });

  return result;
};
