import { useState, useEffect } from 'react';
import {
  FliptEvaluationClient,
  ClientOptions
} from '@flipt-io/flipt-client-browser';

interface FliptClientHook {
  client: FliptEvaluationClient | null;
  isLoading: boolean;
  error: Error | null;
}

export const useFliptClient = (
  namespace: string,
  options: ClientOptions
): FliptClientHook => {
  const [state, setState] = useState<FliptClientHook>(() => ({
    client: null,
    isLoading: true,
    error: null
  }));

  useEffect(() => {
    let isMounted = true;

    const initClient = async () => {
      try {
        const fliptClient = await FliptEvaluationClient.init(
          namespace,
          options
        );
        if (isMounted) {
          setState({
            client: fliptClient,
            isLoading: false,
            error: null
          });
        }
      } catch (err) {
        if (isMounted) {
          setState({
            client: null,
            isLoading: false,
            error:
              err instanceof Error
                ? err
                : new Error('Failed to initialize Flipt client')
          });
        }
      }
    };

    initClient().catch((err) => {
      console.error(
        'Unexpected error during Flipt client initialization:',
        err
      );
    });

    return () => {
      isMounted = false;
    };
  }, [namespace, options.url]);

  return state;
};
