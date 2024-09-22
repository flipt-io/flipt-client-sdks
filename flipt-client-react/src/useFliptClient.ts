import { useState, useEffect } from 'react';
import { FliptEvaluationClient } from '@flipt-io/flipt-client-browser';

interface FliptClientHook {
  client: FliptEvaluationClient | null;
  isLoading: boolean;
  error: Error | null;
}

export const useFliptClient = (
  namespace: string,
  options: { url: string }
): FliptClientHook => {
  console.log('useFliptClient function body entered');

  console.log('About to call useState1');
  const [state, setState] = useState<FliptClientHook>(() => ({
    client: null,
    isLoading: true,
    error: null
  }));
  console.log('useState called');

  console.log('About to call useEffect');
  useEffect(() => {
    console.log('useEffect in useFliptClient triggered');
    console.log('Dependency values:', { namespace, url: options.url });
    let isMounted = true;

    const initClient = async () => {
      console.log('initClient started');
      try {
        const fliptClient = await FliptEvaluationClient.init(
          namespace,
          options
        );
        if (isMounted) {
          console.log('Flipt client initialized:', fliptClient);
          setState({
            client: fliptClient,
            isLoading: false,
            error: null
          });
        }
      } catch (err) {
        if (isMounted) {
          console.error('Failed to initialize Flipt client:', err);
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
      console.log('Cleanup function called');
      isMounted = false;
    };
  }, [namespace, options.url]);
  console.log('useEffect hook declared');

  console.log('useFliptClient returning state:', state);
  return state;
};
