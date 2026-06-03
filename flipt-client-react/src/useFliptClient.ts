import {
  useContext,
  useCallback,
  createContext,
  useEffect,
  useRef,
  useState
} from 'react';
import { useSyncExternalStore } from 'use-sync-external-store/shim';
import type { FliptClient, ClientOptions } from '@flipt-io/flipt-client-js';

export interface FliptClientHook {
  client: FliptClient | null;
  isLoading: boolean;
  error: Error | null;
}

export interface FliptStore extends FliptClientHook {
  subscribe: (onStoreChange: () => void) => () => void;
  attach: () => void;
  detach: () => void;
}

export type FliptProviderOptions = ClientOptions & {
  maxRetryAttempts?: number;
};

export const useStore = (options: FliptProviderOptions): FliptStore => {
  const maxRetryAttempts =
    options.maxRetryAttempts !== undefined
      ? Math.max(0, options.maxRetryAttempts)
      : Infinity;
  const listenersRef = useRef(new Set<() => void>());
  const intervalIdRef = useRef<ReturnType<typeof setInterval> | undefined>(
    undefined
  );
  const mountedRef = useRef<boolean>(false);
  const dataRef = useRef<{
    client: FliptClient | null;
    isLoading: boolean;
    error: Error | null;
  }>({
    client: null,
    isLoading: true,
    error: null
  });

  const notify = useCallback(() => {
    listenersRef.current.forEach((l) => l());
  }, []);

  const setupPolling = useCallback(() => {
    const interval =
      (options.updateInterval !== undefined ? options.updateInterval : 120) *
      1000;
    if (
      interval > 0 &&
      mountedRef.current &&
      dataRef.current.client !== null &&
      intervalIdRef.current === undefined
    ) {
      intervalIdRef.current = setInterval(() => {
        if (
          typeof window !== 'undefined' &&
          navigator.onLine &&
          dataRef.current.client
        ) {
          dataRef.current.client
            .refresh()
            .then((updated) => {
              if (updated) {
                notify();
              }
            })
            .catch((error) => {
              console.error('Error refreshing client:', error);
              dataRef.current.error = error as Error;
              notify();
            });
        }
      }, interval);
    }
  }, [options, notify]);

  const [store] = useState<FliptStore>(() => ({
    get client() {
      return dataRef.current.client;
    },
    get isLoading() {
      return dataRef.current.isLoading;
    },
    get error() {
      return dataRef.current.error;
    },
    subscribe: (listener: () => void) => {
      listenersRef.current.add(listener);
      return () => listenersRef.current.delete(listener);
    },
    attach: () => {
      mountedRef.current = true;
      setupPolling();
    },
    detach: () => {
      mountedRef.current = false;
      clearInterval(intervalIdRef.current);
      intervalIdRef.current = undefined;
    }
  }));

  useEffect(() => {
    let isMounted = true;

    const initializeClient = async () => {
      let attempt = 0;

      while (isMounted && (maxRetryAttempts === Infinity || attempt < maxRetryAttempts)) {
        try {
          const { FliptClient } = await import('@flipt-io/flipt-client-js');
          const client = await FliptClient.init({
            ...options
          });

          if (isMounted) {
            dataRef.current.client = client;
            dataRef.current.isLoading = false;
            dataRef.current.error = null;
            setupPolling();
            notify();
          }
          return;
        } catch (err) {
          if (!isMounted) return;

          attempt++;

          if (maxRetryAttempts !== Infinity && attempt >= maxRetryAttempts) {
            console.error(
              `Flipt client initialization failed after ${attempt} attempts, giving up.`,
              err
            );
            dataRef.current.error = err as Error;
            dataRef.current.isLoading = false;
            notify();
            return;
          }

          const delay = Math.min(1000 * Math.pow(2, attempt - 1), 30000);
          console.warn(
            `Flipt client initialization failed (attempt ${attempt}), retrying in ${delay / 1000}s...`,
            err
          );
          dataRef.current.error = err as Error;
          notify();

          await new Promise((resolve) => setTimeout(resolve, delay));
        }
      }
    };

    void initializeClient();

    return () => {
      isMounted = false;
    };
  }, [options, notify, setupPolling, maxRetryAttempts]);

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
  selector: (
    client: FliptClient | null,
    isLoading: boolean,
    error: Error | null
  ) => T
): T => {
  const store = useContext(FliptContext);
  if (store === null) {
    throw new Error('useFliptSelector must be used within a FliptProvider');
  }

  const selectorWrapper = useCallback(() => {
    return selector(store.client, store.isLoading, store.error);
  }, [store, selector]);

  return useSyncExternalStore(
    store.subscribe,
    selectorWrapper,
    selectorWrapper
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
        return client.evaluateBoolean({
          flagKey,
          entityId,
          context
        }).enabled;
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
        return client.evaluateVariant({
          flagKey,
          entityId,
          context
        }).variantKey;
      } catch (e) {
        console.error(`Error evaluating variant flag ${flagKey}:`, e);
      }
    }
    return fallback;
  });

  return result;
};
