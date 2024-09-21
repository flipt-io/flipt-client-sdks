import React, { createContext, useContext, useEffect, useState } from 'react';
import {
  FliptEvaluationClient,
  EvaluationRequest,
  BooleanEvaluationResponse,
  BatchEvaluationResponse,
  Flag,
  VariantEvaluationResponse
} from '@flipt-io/flipt-client-browser';
import { FliptProviderProps } from './types';

interface FliptContextType {
  client: FliptEvaluationClient | null;
  isLoading: boolean;
  error: Error | null;
}

const FliptContext = createContext<FliptContextType>({
  client: null,
  isLoading: true,
  error: null
});

export const FliptProvider: React.FC<FliptProviderProps> = ({
  namespace,
  options,
  children
}) => {
  const [client, setClient] = useState<FliptEvaluationClient | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    const initClient = async () => {
      try {
        const fliptClient = await FliptEvaluationClient.init(
          namespace,
          options
        );
        setClient(fliptClient);
        setIsLoading(false);
      } catch (err) {
        setError(
          err instanceof Error
            ? err
            : new Error('Failed to initialize Flipt client')
        );
        setIsLoading(false);
      }
    };

    initClient();
  }, [namespace, options]);

  return (
    <FliptContext.Provider value={{ client, isLoading, error }}>
      {children}
    </FliptContext.Provider>
  );
};

export const useFliptClient = () => useContext(FliptContext);

export const useRefresh = () => {
  const { client, error } = useFliptClient();

  return React.useCallback(() => {
    if (error) {
      console.error('Flipt client error:', error);
      return;
    }
    client?.refresh();
  }, [client, error]);
};

export const useVariantFlag = (
  flagKey: string,
  entityId: string,
  context: {}
) => {
  const { client, isLoading, error } = useFliptClient();
  const [flagValue, setFlagValue] = useState<VariantEvaluationResponse | null>(
    null
  );
  const [flagError, setFlagError] = useState<Error | null>(null);

  useEffect(() => {
    if (client && !isLoading && !error) {
      try {
        const result = client.evaluateVariant(flagKey, entityId, context);
        setFlagValue(result);
      } catch (err) {
        setFlagError(
          err instanceof Error
            ? err
            : new Error('Failed to evaluate variant flag')
        );
      }
    }
  }, [client, isLoading, error, flagKey, entityId, context]);

  return { value: flagValue, isLoading, error: error || flagError };
};

export const useBooleanFlag = (
  flagKey: string,
  entityId: string,
  context: {}
) => {
  const { client, isLoading, error } = useFliptClient();
  const [flagValue, setFlagValue] = useState<BooleanEvaluationResponse | null>(
    null
  );
  const [flagError, setFlagError] = useState<Error | null>(null);

  useEffect(() => {
    if (client && !isLoading && !error) {
      try {
        const result = client.evaluateBoolean(flagKey, entityId, context);
        setFlagValue(result);
      } catch (err) {
        setFlagError(
          err instanceof Error
            ? err
            : new Error('Failed to evaluate boolean flag')
        );
      }
    }
  }, [client, isLoading, error, flagKey, entityId, context]);

  return { value: flagValue, isLoading, error: error || flagError };
};

export const useBatchEvaluation = (requests: EvaluationRequest[]) => {
  const { client, isLoading, error } = useFliptClient();
  const [results, setResults] = useState<BatchEvaluationResponse | null>(null);
  const [batchError, setBatchError] = useState<Error | null>(null);

  useEffect(() => {
    if (client && !isLoading && !error) {
      try {
        const batchResult = client.evaluateBatch(requests);
        setResults(batchResult);
      } catch (err) {
        setBatchError(
          err instanceof Error ? err : new Error('Failed to evaluate batch')
        );
      }
    }
  }, [client, isLoading, error, requests]);

  return { results, isLoading, error: error || batchError };
};

export const useFlags = () => {
  const { client, isLoading, error } = useFliptClient();
  const [flags, setFlags] = useState<Flag[] | null>(null);
  const [flagsError, setFlagsError] = useState<Error | null>(null);

  useEffect(() => {
    if (client && !isLoading && !error) {
      try {
        const flagList = client.listFlags();
        setFlags(flagList);
      } catch (err) {
        setFlagsError(
          err instanceof Error ? err : new Error('Failed to list flags')
        );
      }
    }
  }, [client, isLoading, error]);

  return { flags, isLoading, error: error || flagsError };
};

export * from './types';
