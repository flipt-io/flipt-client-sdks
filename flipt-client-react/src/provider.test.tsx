import React from 'react';
import { render, act, renderHook, waitFor } from '@testing-library/react';
import {
  FliptProvider,
  useFliptClient,
  useVariantFlag,
  useBooleanFlag,
  useBatchEvaluation,
  useFlags
} from './provider';
import { FliptEvaluationClient } from '@flipt-io/flipt-client-browser';

// Mock the FliptEvaluationClient
jest.mock('@flipt-io/flipt-client-browser', () => ({
  FliptEvaluationClient: {
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

    (FliptEvaluationClient.init as jest.Mock).mockResolvedValue(mockClient);

    let renderedContext: ReturnType<typeof useFliptClient> | undefined;

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
      renderedContext = useFliptClient();
      return null;
    }

    expect(FliptEvaluationClient.init).toHaveBeenCalledWith('test', {
      url: 'http://localhost:8080'
    });
    expect(renderedContext!.client).toBe(mockClient);
    expect(renderedContext!.isLoading).toBe(false);
    expect(renderedContext!.error).toBe(null);
  });
});

describe('useVariantFlag', () => {
  it('evaluates a variant flag', async () => {
    const mockClient = {
      evaluateVariant: jest.fn().mockReturnValue({ variant: 'test-variant' })
    };

    (FliptEvaluationClient.init as jest.Mock).mockResolvedValue(mockClient);

    const wrapper = ({ children }: { children: React.ReactNode }) => (
      <FliptProvider
        namespace="test"
        options={{ url: 'http://localhost:8080' }}
      >
        {children}
      </FliptProvider>
    );

    const { result } = renderHook(
      () => useVariantFlag('test-flag', 'user-1', {}),
      { wrapper }
    );

    await waitFor(
      () => {
        expect(result.current.value).toEqual({ variant: 'test-variant' });
      },
      { timeout: 1000 }
    );

    expect(result.current.isLoading).toBe(false);
    expect(result.current.error).toBe(null);
  });
});

describe('useBooleanFlag', () => {
  it('evaluates a boolean flag', async () => {
    const mockClient = {
      evaluateBoolean: jest.fn().mockReturnValue({ enabled: true })
    };

    (FliptEvaluationClient.init as jest.Mock).mockResolvedValue(mockClient);

    const wrapper = ({ children }: { children: React.ReactNode }) => (
      <FliptProvider
        namespace="test"
        options={{ url: 'http://localhost:8080' }}
      >
        {children}
      </FliptProvider>
    );

    const { result } = renderHook(
      () => useBooleanFlag('test-flag', 'user-1', {}),
      { wrapper }
    );

    await waitFor(
      () => {
        expect(result.current.value).toEqual({ enabled: true });
      },
      { timeout: 1000 }
    );

    expect(result.current.isLoading).toBe(false);
    expect(result.current.error).toBe(null);
  });
});

describe('useBatchEvaluation', () => {
  it('evaluates multiple flags in batch', async () => {
    const mockClient = {
      evaluateBatch: jest.fn().mockReturnValue({
        responses: [
          { flagKey: 'flag1', variant: 'variant1' },
          { flagKey: 'flag2', enabled: true }
        ]
      })
    };

    (FliptEvaluationClient.init as jest.Mock).mockResolvedValue(mockClient);

    const wrapper = ({ children }: { children: React.ReactNode }) => (
      <FliptProvider
        namespace="test"
        options={{ url: 'http://localhost:8080' }}
      >
        {children}
      </FliptProvider>
    );

    const { result } = renderHook(
      () =>
        useBatchEvaluation([
          { flagKey: 'flag1', entityId: 'user-1', context: {} },
          { flagKey: 'flag2', entityId: 'user-1', context: {} }
        ]),
      { wrapper }
    );

    await waitFor(
      () => {
        expect(result.current.results).toEqual({
          responses: [
            { flagKey: 'flag1', variant: 'variant1' },
            { flagKey: 'flag2', enabled: true }
          ]
        });
      },
      { timeout: 1000 }
    );

    expect(result.current.isLoading).toBe(false);
    expect(result.current.error).toBe(null);

    expect(mockClient.evaluateBatch).toHaveBeenCalledWith([
      { context: {}, entityId: 'user-1', flagKey: 'flag1' },
      { context: {}, entityId: 'user-1', flagKey: 'flag2' }
    ]);
  });
});

describe('useFlags', () => {
  it('fetches all flags', async () => {
    const mockClient = {
      listFlags: jest.fn().mockReturnValue([
        { key: 'flag1', type: 'VARIANT_FLAG' },
        { key: 'flag2', type: 'BOOLEAN_FLAG' }
      ])
    };

    (FliptEvaluationClient.init as jest.Mock).mockResolvedValue(mockClient);

    const wrapper = ({ children }: { children: React.ReactNode }) => (
      <FliptProvider
        namespace="test"
        options={{ url: 'http://localhost:8080' }}
      >
        {children}
      </FliptProvider>
    );

    const { result } = renderHook(() => useFlags(), { wrapper });

    await waitFor(
      () => {
        expect(result.current.flags).toEqual([
          { key: 'flag1', type: 'VARIANT_FLAG' },
          { key: 'flag2', type: 'BOOLEAN_FLAG' }
        ]);
      },
      { timeout: 1000 }
    );

    expect(result.current.isLoading).toBe(false);
    expect(result.current.error).toBe(null);
    expect(mockClient.listFlags).toHaveBeenCalled();
  });
});
