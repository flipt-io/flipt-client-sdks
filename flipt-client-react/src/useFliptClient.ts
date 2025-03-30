import {
  useContext,
  useSyncExternalStore,
  useCallback,
  createContext,
  useEffect,
  useRef,
  useMemo
} from 'react';
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

export const useStore = (
  namespace: string,
  options: ClientOptions
): FliptStore => {
  const storeRef = useRef<FliptStore>({
    client: null,
    isLoading: true,
    error: null,
    subscribe: (listener: () => void) => {
      listeners.add(listener);
      return () => listeners.delete(listener);
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
  });

  const listeners = useMemo(() => new Set<() => void>(), []);

  const notify = useCallback(() => {
    listeners.forEach((l) => l());
  }, [listeners]);

  const intervalIdRef = useRef<ReturnType<typeof setInterval> | undefined>(
    undefined
  );
  const mountedRef = useRef<boolean>(false);

  const setupPolling = useCallback(() => {
    // Default to 120 seconds if updateInterval is not set
    const interval =
      (options.updateInterval !== undefined ? options.updateInterval : 120) *
      1000;
    if (
      interval > 0 &&
      mountedRef.current &&
      storeRef.current.client !== null &&
      intervalIdRef.current === undefined
    ) {
      intervalIdRef.current = setInterval(() => {
        if (
          typeof window !== 'undefined' &&
          navigator.onLine &&
          storeRef.current.client
        ) {
          storeRef.current.client
            .refresh()
            .then((updated) => {
              if (updated) {
                notify();
              }
            })
            .catch((error) => {
              console.error('Error refreshing client:', error);
              storeRef.current.error = error as Error;
              notify();
            });
        }
      }, interval);
    }
  }, [options, notify]);

  useEffect(() => {
    mountedRef.current = true;

    return () => {
      mountedRef.current = false;
      if (intervalIdRef.current) {
        clearInterval(intervalIdRef.current);
        intervalIdRef.current = undefined;
      }
    };
  }, []);

  useEffect(() => {
    let isMounted = true;

    const initializeClient = async () => {
      try {
        const { FliptClient } = await import('@flipt-io/flipt-client-js');
        const client = await FliptClient.init({
          namespace,
          ...options
        });

        if (isMounted) {
          storeRef.current.client = client;
          storeRef.current.isLoading = false;
          setupPolling();
          notify();
        }
      } catch (err) {
        if (isMounted) {
          console.error('Error initializing client:', err);
          storeRef.current.error = err as Error;
          storeRef.current.isLoading = false;
          notify();
        }
      }
    };

    initializeClient().catch((err) => {
      console.error('Unhandled error in initializeClient:', err);
      if (isMounted) {
        storeRef.current.error = err as Error;
        storeRef.current.isLoading = false;
        notify();
      }
    });

    return () => {
      isMounted = false;
    };
  }, [namespace, options, notify, setupPolling]);

  return storeRef.current;
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
