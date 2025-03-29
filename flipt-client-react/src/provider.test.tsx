import React from 'react';
import { render, act, renderHook, waitFor } from '@testing-library/react';
import {
  FliptProvider,
  FliptClientHook,
  useFliptVariant,
  useFliptBoolean,
  useFliptSelector
} from './index';
import {
  BatchEvaluationResponse,
  BooleanEvaluationResponse,
  FliptClient,
  VariantEvaluationResponse
} from '@flipt-io/flipt-client-js';

// Mock the FliptEvaluationClient
jest.mock('@flipt-io/flipt-client-js', () => ({
  FliptClient: {
    init: jest.fn()
  }
}));

describe('FliptProvider', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('initializes the client and provides context', async () => {
    const mockClient = {
      evaluateVariant: jest.fn(),
      evaluateBoolean: jest.fn()
    };

    (FliptClient.init as jest.Mock).mockResolvedValue(mockClient);

    let renderedContext: string | undefined;

    let gotClient: FliptClient | null = null;
    let gotIsLoading: boolean = true;
    let gotError: Error | null | undefined = undefined;

    await act(async () => {
      render(
        <FliptProvider
          namespace="test"
          options={{ url: 'http://localhost:8080' }}
        >
          <TestComponent />
        </FliptProvider>
      );
    });

    function TestComponent() {
      renderedContext = useFliptSelector((client, isLoading, error) => {
        gotClient = client;
        gotIsLoading = isLoading;
        gotError = error;
        return 'loaded';
      });
      return null;
    }

    expect(FliptClient.init).toHaveBeenCalledWith({
      namespace: 'test',
      url: 'http://localhost:8080'
    });
    expect(gotClient).toBe(mockClient);
    expect(gotIsLoading).toBe(false);
    expect(gotError).toBe(null);
  });
});

describe('useFliptContext', () => {
  it('evaluates a variant flag', async () => {
    const mockClient = {
      evaluateVariant: jest.fn().mockReturnValue({
        variantKey: 'test-variant'
      } as VariantEvaluationResponse)
    };

    (FliptClient.init as jest.Mock).mockResolvedValue(mockClient);

    const wrapper = ({ children }: { children: React.ReactNode }) => (
      <FliptProvider
        namespace="test"
        options={{ url: 'http://localhost:8080' }}
      >
        {children}
      </FliptProvider>
    );

    const { result } = renderHook(
      () => useFliptVariant('test-flag', 'fallback', 'user-1'),
      { wrapper }
    );

    await waitFor(
      () => {
        expect(result.current).toEqual('test-variant');
      },
      { timeout: 1000 }
    );

    // Assert that the mock client's evaluateVariant was called with the correct arguments
    expect(mockClient.evaluateVariant).toHaveBeenCalledWith({
      flagKey: 'test-flag',
      entityId: 'user-1',
      context: {}
    });
  });
});

it('evaluates a boolean flag', async () => {
  const mockClient = {
    evaluateBoolean: jest
      .fn()
      .mockReturnValue({ enabled: true } as BooleanEvaluationResponse)
  };

  (FliptClient.init as jest.Mock).mockResolvedValue(mockClient);

  const wrapper = ({ children }: { children: React.ReactNode }) => (
    <FliptProvider namespace="test" options={{ url: 'http://localhost:8080' }}>
      {children}
    </FliptProvider>
  );

  const { result } = renderHook(
    () => useFliptBoolean('test-flag', false, 'user-1'),
    { wrapper }
  );

  await waitFor(
    () => {
      expect(result.current).toEqual(true);
    },
    { timeout: 1000 }
  );

  // Assert that the mock client's evaluateBoolean was called with the correct arguments
  expect(mockClient.evaluateBoolean).toHaveBeenCalledWith({
    flagKey: 'test-flag',
    entityId: 'user-1',
    context: {}
  });
});

it('evaluates multiple flags in batch', async () => {
  const mockClient = {
    evaluateBatch: jest.fn().mockReturnValue({
      responses: [
        {
          type: 'VARIANT_EVALUATION_RESPONSE_TYPE',
          variantEvaluationResponse: {
            flagKey: 'flag1',
            variantKey: 'variant1'
          }
        },
        {
          type: 'BOOLEAN_EVALUATION_RESPONSE_TYPE',
          booleanEvaluationResponse: { flagKey: 'flag2', enabled: true }
        }
      ]
    } as BatchEvaluationResponse)
  };

  (FliptClient.init as jest.Mock).mockResolvedValue(mockClient);

  const wrapper = ({ children }: { children: React.ReactNode }) => (
    <FliptProvider namespace="test" options={{ url: 'http://localhost:8080' }}>
      {children}
    </FliptProvider>
  );

  const { result } = renderHook(
    () =>
      useFliptSelector((client) => {
        // Call evaluateBatch on the client
        const values = client?.evaluateBatch([
          { context: {}, entityId: 'user-1', flagKey: 'flag1' },
          { context: {}, entityId: 'user-1', flagKey: 'flag2' }
        ]);
        return (
          values?.responses[0].variantEvaluationResponse?.variantKey +
          '-' +
          values?.responses[1].booleanEvaluationResponse?.enabled
        );
      }),
    { wrapper }
  );

  await waitFor(
    () => {
      expect(result.current).toEqual('variant1-true');
    },
    { timeout: 1000 }
  );

  // Assert that the mock client's evaluateBatch was called with the correct arguments
  expect(mockClient.evaluateBatch).toHaveBeenCalledWith([
    { context: {}, entityId: 'user-1', flagKey: 'flag1' },
    { context: {}, entityId: 'user-1', flagKey: 'flag2' }
  ]);
});

it('fetches all flags', async () => {
  const mockClient = {
    listFlags: jest.fn().mockReturnValue([
      { key: 'flag1', type: 'VARIANT_FLAG' },
      { key: 'flag2', type: 'BOOLEAN_FLAG' }
    ])
  };

  (FliptClient.init as jest.Mock).mockResolvedValue(mockClient);

  const wrapper = ({ children }: { children: React.ReactNode }) => (
    <FliptProvider namespace="test" options={{ url: 'http://localhost:8080' }}>
      {children}
    </FliptProvider>
  );

  const { result } = renderHook(
    () =>
      useFliptSelector((client) => {
        return client
          ?.listFlags()
          .map((f) => f.key)
          .join('-');
      }),
    {
      wrapper
    }
  );

  await waitFor(
    () => {
      expect(result.current).toEqual('flag1-flag2');
    },
    { timeout: 1000 }
  );

  // Assert that the mock client's listFlags was called
  expect(mockClient.listFlags).toHaveBeenCalled();
});
