import {
  useContext,
  useSyncExternalStore,
  useCallback,
  createContext,
  useEffect,
  useRef
} from 'react';
import type {
  FliptEvaluationClient,
  ClientOptions
} from '@flipt-io/flipt-client-browser';

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

export const configureStore = (
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
      mounted = true;
      setupPolling();
    },
    detach: () => {
      mounted = false;
      clearInterval(intervalId);
      intervalId = undefined;
    }
  });

  const listeners = new Set<() => void>();
  const notify = () => {
    listeners.forEach((l) => l());
  };
  let intervalId: any;
  let mounted: boolean = false;

  const interval = options.updateInterval || 0;

  const setupPolling = () => {
    if (
      interval > 0 &&
      mounted &&
      storeRef.current.client !== null &&
      intervalId === undefined
    ) {
      intervalId = setInterval(() => {
        if (typeof window !== 'undefined' && navigator.onLine) {
          storeRef.current.client?.refresh().then((updated) => {
            if (updated) {
              notify();
            }
          });
        }
      }, interval);
    }
  };

  useEffect(() => {
    let isMounted = true;

    const initializeClient = async () => {
      try {
        const { FliptEvaluationClient } = await import('@flipt-io/flipt-client-browser');
        const client = await FliptEvaluationClient.init(namespace, options);
        
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

    initializeClient();

    return () => {
      isMounted = false;
    };
  }, [namespace, options]);

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
    client: FliptEvaluationClient | null,
    isLoading: boolean,
    error: Error | null
  ) => T
): T => {
  const store = useContext(FliptContext);
  if (store === null) {
    throw new Error('useFliptSelector must be used within a FliptProvider');
  }
  
  const selectorWrapper = useCallback(
    () => {
      return selector(store.client, store.isLoading, store.error);
    },
    [store, selector]
  );

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
        console.error(`Error evaluating variant flag ${flagKey}:`, e);
      }
    }
    return fallback;
  });

  return result;
};
