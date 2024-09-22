import React from 'react';
import { render, act, renderHook, waitFor } from '@testing-library/react';
import { FliptProvider, useFliptClient, useFliptContext } from './index';
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
      renderedContext = useFliptContext();
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

describe('useFliptContext', () => {
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

    const { result } = renderHook(() => useFliptContext(), { wrapper });

    await waitFor(
      () => {
        expect(result.current.client).toEqual(mockClient);
      },
      { timeout: 1000 }
    );

    expect(result.current.isLoading).toBe(false);
    expect(result.current.error).toBe(null);

    // Call evaluateVariant on the client
    await result?.current?.client?.evaluateVariant('test-flag', 'user-1', {});

    // Assert that the mock client's evaluateVariant was called with the correct arguments
    expect(mockClient.evaluateVariant).toHaveBeenCalledWith(
      'test-flag',
      'user-1',
      {}
    );
  });
});

it('evaluates a boolean flag', async () => {
  const mockClient = {
    evaluateBoolean: jest.fn().mockReturnValue({ enabled: true })
  };

  (FliptEvaluationClient.init as jest.Mock).mockResolvedValue(mockClient);

  const wrapper = ({ children }: { children: React.ReactNode }) => (
    <FliptProvider namespace="test" options={{ url: 'http://localhost:8080' }}>
      {children}
    </FliptProvider>
  );

  const { result } = renderHook(() => useFliptContext(), { wrapper });

  await waitFor(
    () => {
      expect(result.current.client).toEqual(mockClient);
    },
    { timeout: 1000 }
  );

  expect(result.current.isLoading).toBe(false);
  expect(result.current.error).toBe(null);

  // Call evaluateBoolean on the client
  await result?.current?.client?.evaluateBoolean('test-flag', 'user-1', {});

  // Assert that the mock client's evaluateBoolean was called with the correct arguments
  expect(mockClient.evaluateBoolean).toHaveBeenCalledWith(
    'test-flag',
    'user-1',
    {}
  );
});

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
    <FliptProvider namespace="test" options={{ url: 'http://localhost:8080' }}>
      {children}
    </FliptProvider>
  );

  const { result } = renderHook(() => useFliptContext(), { wrapper });

  await waitFor(
    () => {
      expect(result.current.client).toEqual(mockClient);
    },
    { timeout: 1000 }
  );

  expect(result.current.isLoading).toBe(false);
  expect(result.current.error).toBe(null);

  // Call evaluateBatch on the client
  await result?.current?.client?.evaluateBatch([
    { context: {}, entityId: 'user-1', flagKey: 'flag1' },
    { context: {}, entityId: 'user-1', flagKey: 'flag2' }
  ]);

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

  (FliptEvaluationClient.init as jest.Mock).mockResolvedValue(mockClient);

  const wrapper = ({ children }: { children: React.ReactNode }) => (
    <FliptProvider namespace="test" options={{ url: 'http://localhost:8080' }}>
      {children}
    </FliptProvider>
  );

  const { result } = renderHook(() => useFliptContext(), { wrapper });

  await waitFor(
    () => {
      expect(result.current.client).toEqual(mockClient);
    },
    { timeout: 1000 }
  );

  expect(result.current.isLoading).toBe(false);
  expect(result.current.error).toBe(null);

  // Call listFlags on the client
  await result?.current?.client?.listFlags();

  // Assert that the mock client's listFlags was called
  expect(mockClient.listFlags).toHaveBeenCalled();
});
