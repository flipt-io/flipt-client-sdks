package evaluation

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	_ "embed"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

var (
	//go:embed ext/flipt_engine_wasm.wasm
	wasm []byte
)

const statusSuccess = "success"

// EvaluationClient wraps the functionality of evaluating Flipt feature flags.
type EvaluationClient struct {
	runtime wazero.Runtime
	mod     api.Module
	// guards access to etag and error state as they are shared between goroutines
	mu sync.RWMutex

	engine    uint32
	namespace string
	err       error

	url            string
	authentication any
	ref            string
	updateInterval time.Duration
	fetchMode      FetchMode
	errorStrategy  ErrorStrategy

	httpClient *http.Client
	etag       string

	wg   sync.WaitGroup
	done chan struct{}

	errChan      chan error
	snapshotChan chan snapshot

	// cached WASM functions
	allocFunc    api.Function
	deallocFunc  api.Function
	snapshotFunc api.Function
}

// ClientOption adds additional configuration for Client parameters
type ClientOption func(*EvaluationClient)

// WithNamespace allows for specifying which namespace the clients wants to make evaluations from.
func WithNamespace(namespace string) ClientOption {
	return func(c *EvaluationClient) {
		c.namespace = namespace
	}
}

// WithURL allows for configuring the URL of an upstream Flipt instance to fetch feature data.
func WithURL(url string) ClientOption {
	return func(c *EvaluationClient) {
		c.url = url
	}
}

// WithRef allows for specifying a reference to fetch feature data from.
func WithRef(ref string) ClientOption {
	return func(c *EvaluationClient) {
		c.ref = ref
	}
}

// WithUpdateInterval allows for specifying how often flag state data should be fetched from an upstream Flipt instance.
func WithUpdateInterval(updateInterval time.Duration) ClientOption {
	return func(c *EvaluationClient) {
		if updateInterval > 0 {
			c.updateInterval = updateInterval
		}
	}
}

// WithClientTokenAuthentication allows authenticating with Flipt using a static client token.
func WithClientTokenAuthentication(token string) ClientOption {
	return func(c *EvaluationClient) {
		c.authentication = clientTokenAuthentication{
			Token: token,
		}
	}
}

// WithJWTAuthentication allows authenticating with Flipt using a JSON Web Token.
func WithJWTAuthentication(token string) ClientOption {
	return func(c *EvaluationClient) {
		c.authentication = jwtAuthentication{
			Token: token,
		}
	}
}

// WithFetchMode allows for specifying the fetch mode for the Flipt client (e.g. polling, streaming).
// Note: Streaming is currently only supported when using the SDK with Flipt Cloud (https://flipt.io/cloud).
func WithFetchMode(fetchMode FetchMode) ClientOption {
	return func(c *EvaluationClient) {
		c.fetchMode = fetchMode
	}
}

// WithErrorStrategy allows for specifying the error strategy for the Flipt client when fetching flag state (e.g. fail, fallback).
func WithErrorStrategy(errorStrategy ErrorStrategy) ClientOption {
	return func(c *EvaluationClient) {
		c.errorStrategy = errorStrategy
	}
}

type fetchError error

// NewEvaluationClient constructs a Client and performs an initial fetch of flag state.
func NewEvaluationClient(ctx context.Context, opts ...ClientOption) (*EvaluationClient, error) {
	runtime := wazero.NewRuntime(ctx)
	wasi_snapshot_preview1.MustInstantiate(ctx, runtime)

	compiled, err := runtime.CompileModule(ctx, wasm)
	if err != nil {
		runtime.Close(ctx)
		return nil, fmt.Errorf("failed to compile and load evaluation engine: %w", err)
	}

	cfg := wazero.NewModuleConfig().WithName("flipt_engine")
	mod, err := runtime.InstantiateModule(ctx, compiled, cfg)
	if err != nil {
		runtime.Close(ctx)
		return nil, fmt.Errorf("can't instantiate Wasm module: %w", err)
	}

	var (
		// cache common WASM functions
		allocFunc    = mod.ExportedFunction("allocate")
		deallocFunc  = mod.ExportedFunction("deallocate")
		snapshotFunc = mod.ExportedFunction("snapshot")
	)

	client := &EvaluationClient{
		runtime: runtime,
		mod:     mod,

		// default values
		namespace:      "default",
		url:            "http://localhost:8080",
		updateInterval: 2 * time.Minute, // default 120 seconds
		errorStrategy:  ErrorStrategyFail,
		fetchMode:      FetchModePolling,

		httpClient: &http.Client{},

		done:         make(chan struct{}),
		errChan:      make(chan error, 1),
		snapshotChan: make(chan snapshot, 1),

		// cache WASM functions
		allocFunc:    allocFunc,
		deallocFunc:  deallocFunc,
		snapshotFunc: snapshotFunc,
	}

	for _, opt := range opts {
		opt(client)
	}

	if client.namespace == "" {
		runtime.Close(ctx)
		return nil, fmt.Errorf("namespace cannot be empty")
	}

	var initializeEngine = mod.ExportedFunction("initialize_engine")

	type allocation struct {
		ptr  uint64
		size uint64
	}

	// track allocations that need to be freed in case of error
	var allocations []allocation

	// cleanup function that only runs if we return an error
	var cleanup bool
	defer func() {
		if cleanup {
			for _, a := range allocations {
				deallocFunc.Call(ctx, a.ptr, a.size)
			}
			client.Close(ctx)
		}
	}()

	// allocate namespace
	nsPtr, err := allocFunc.Call(ctx, uint64(len(client.namespace)))
	if err != nil {
		cleanup = true
		return nil, fmt.Errorf("failed to allocate memory for namespace: %w", err)
	}
	allocations = append(allocations, allocation{nsPtr[0], uint64(len(client.namespace))})

	if !mod.Memory().Write(uint32(nsPtr[0]), []byte(client.namespace)) {
		cleanup = true
		return nil, fmt.Errorf("failed to write namespace to memory")
	}

	// fetch initial state
	snapshot, err := client.fetch(ctx, "")
	if err != nil {
		cleanup = true
		return nil, fmt.Errorf("failed to fetch initial state: %w", err)
	}

	client.etag = snapshot.etag

	// allocate payload
	pmPtr, err := allocFunc.Call(ctx, uint64(len(snapshot.payload)))
	if err != nil {
		cleanup = true
		return nil, fmt.Errorf("failed to allocate memory for payload: %w", err)
	}
	allocations = append(allocations, allocation{pmPtr[0], uint64(len(snapshot.payload))})

	if !mod.Memory().Write(uint32(pmPtr[0]), []byte(snapshot.payload)) {
		cleanup = true
		return nil, fmt.Errorf("failed to write payload to memory")
	}

	// initialize engine
	res, err := initializeEngine.Call(ctx, nsPtr[0], uint64(len(client.namespace)), pmPtr[0], uint64(len(snapshot.payload)))
	if err != nil {
		cleanup = true
		return nil, fmt.Errorf("failed to initialize engine: %w", err)
	}

	enginePtr := res[0]
	client.engine = uint32(enginePtr)

	client.wg.Add(1)
	go func() {
		defer client.wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case <-client.done:
				return
			case err := <-client.errChan:
				if err == nil {
					continue
				}

				client.mu.Lock()
				client.err = err
				client.mu.Unlock()

				var fetchErr fetchError
				// if we got a fetch error and the error strategy is to fail, we can continue
				// as the fetch error will be returned from the evaluation methods
				if errors.As(err, &fetchErr) && client.errorStrategy == ErrorStrategyFail {
					continue
				}

				// if we got a sentinel error we need to signal the background goroutines to stop
				close(client.done)
				return
			}
		}
	}()

	client.wg.Add(1)
	go func() {
		defer client.wg.Done()
		if err := client.handleUpdates(ctx); err != nil {
			client.errChan <- err
		}
	}()

	client.wg.Add(1)
	go func() {
		defer client.wg.Done()
		switch client.fetchMode {
		case FetchModeStreaming:
			if err := client.startStreaming(ctx); err != nil {
				client.errChan <- err
			}
		case FetchModePolling:
			if err := client.startPolling(ctx); err != nil {
				client.errChan <- err
			}
		default:
			client.errChan <- fmt.Errorf("invalid fetch mode: %s", client.fetchMode)
			return
		}
	}()

	return client, nil
}

// Err returns the last error that occurred.
func (e *EvaluationClient) Err() error {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.err
}

// EvaluateVariant performs evaluation for a variant flag.
func (e *EvaluationClient) EvaluateVariant(ctx context.Context, flagKey, entityID string, evalContext map[string]string) (*VariantEvaluationResponse, error) {
	e.mu.RLock()
	if e.err != nil && e.errorStrategy == ErrorStrategyFail {
		e.mu.RUnlock()
		return nil, e.err
	}
	e.mu.RUnlock()

	request := EvaluationRequest{
		FlagKey:  flagKey,
		EntityId: entityID,
		Context:  evalContext,
	}

	result, err := e.evaluateWASM(ctx, "evaluate_variant", request)
	if err != nil {
		return nil, err
	}

	var variantResult *VariantResult
	if err := json.Unmarshal(result, &variantResult); err != nil {
		return nil, fmt.Errorf("failed to unmarshal variant result: %w", err)
	}

	if variantResult == nil {
		return nil, errors.New("failed to unmarshal variant result: nil")
	}

	if variantResult.Status != statusSuccess {
		return nil, errors.New(variantResult.ErrorMessage)
	}

	return variantResult.Result, nil
}

// EvaluateBoolean performs evaluation for a boolean flag.
func (e *EvaluationClient) EvaluateBoolean(ctx context.Context, flagKey, entityID string, evalContext map[string]string) (*BooleanEvaluationResponse, error) {
	e.mu.RLock()
	if e.err != nil && e.errorStrategy == ErrorStrategyFail {
		e.mu.RUnlock()
		return nil, e.err
	}
	e.mu.RUnlock()

	request := EvaluationRequest{
		FlagKey:  flagKey,
		EntityId: entityID,
		Context:  evalContext,
	}

	result, err := e.evaluateWASM(ctx, "evaluate_boolean", request)
	if err != nil {
		return nil, err
	}

	var booleanResult *BooleanResult
	if err := json.Unmarshal(result, &booleanResult); err != nil {
		return nil, fmt.Errorf("failed to unmarshal boolean result: %w", err)
	}

	if booleanResult == nil {
		return nil, errors.New("failed to unmarshal boolean result: nil")
	}

	if booleanResult.Status != statusSuccess {
		return nil, errors.New(booleanResult.ErrorMessage)
	}

	return booleanResult.Result, nil
}

// EvaluateBatch performs evaluation for a batch of flags.
func (e *EvaluationClient) EvaluateBatch(ctx context.Context, requests []*EvaluationRequest) (*BatchEvaluationResponse, error) {
	e.mu.RLock()
	if e.err != nil && e.errorStrategy == ErrorStrategyFail {
		e.mu.RUnlock()
		return nil, e.err
	}
	e.mu.RUnlock()

	result, err := e.evaluateWASM(ctx, "evaluate_batch", requests)
	if err != nil {
		return nil, err
	}

	var batchResult *BatchResult
	if err := json.Unmarshal(result, &batchResult); err != nil {
		return nil, fmt.Errorf("failed to unmarshal batch result: %w", err)
	}

	if batchResult == nil {
		return nil, errors.New("failed to unmarshal batch result: nil")
	}

	if batchResult.Status != statusSuccess {
		return nil, errors.New(batchResult.ErrorMessage)
	}

	return batchResult.Result, nil
}

// ListFlags lists all flags.
func (e *EvaluationClient) ListFlags(ctx context.Context) ([]Flag, error) {
	e.mu.RLock()
	if e.err != nil && e.errorStrategy == ErrorStrategyFail {
		e.mu.RUnlock()
		return nil, e.err
	}
	e.mu.RUnlock()

	if e.engine == 0 {
		return nil, errors.New("engine not initialized")
	}

	evalFunc := e.mod.ExportedFunction("list_flags")
	res, err := evalFunc.Call(ctx, uint64(e.engine))
	if err != nil {
		return nil, fmt.Errorf("failed to call list_flags: %w", err)
	}

	if len(res) < 1 {
		return nil, fmt.Errorf("failed to call list_flags: no result returned")
	}

	ptr, len := decodePtr(res[0])
	defer e.deallocFunc.Call(ctx, uint64(ptr), uint64(len))

	b, ok := e.mod.Memory().Read(ptr, len)
	if !ok {
		return nil, fmt.Errorf("failed to read result from memory")
	}

	var result *ListFlagsResult
	if err := json.Unmarshal(b, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal flags: %w", err)
	}

	if result == nil {
		return nil, errors.New("failed to unmarshal flags: nil")
	}

	if result.Status != statusSuccess {
		return nil, errors.New(result.ErrorMessage)
	}

	return *result.Result, nil
}

// Close cleans up the allocated resources.
func (e *EvaluationClient) Close(ctx context.Context) (err error) {
	// signal background goroutines to stop
	close(e.done)

	// wait for all background goroutines to finish
	e.wg.Wait()

	// close channels
	close(e.errChan)
	close(e.snapshotChan)

	if e.engine != 0 {
		// destroy engine
		dealloc := e.mod.ExportedFunction("destroy_engine")
		dealloc.Call(ctx, uint64(e.engine))
	}

	// closing the runtime will close the module too
	return e.runtime.Close(ctx)
}

// unexported

func decodePtr(ptr uint64) (uint32, uint32) {
	return uint32(ptr >> 32), uint32(ptr)
}

type snapshot struct {
	payload json.RawMessage
	etag    string
}

func (e *EvaluationClient) handleUpdates(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			close(e.snapshotChan)
			return nil
		case <-e.done:
			close(e.snapshotChan)
			return nil
		case s, ok := <-e.snapshotChan:
			if !ok {
				// we are likely shutting down
				return nil
			}

			e.mu.Lock()
			e.etag = s.etag
			e.mu.Unlock()

			// skip update if no changes (304 response) or error
			if len(s.payload) == 0 {
				continue
			}

			pmPtr, err := e.allocFunc.Call(ctx, uint64(len(s.payload)))
			if err != nil {
				return fmt.Errorf("failed to allocate memory for payload: %w", err)
			}

			if !e.mod.Memory().Write(uint32(pmPtr[0]), []byte(s.payload)) {
				e.deallocFunc.Call(ctx, uint64(pmPtr[0]), uint64(len(s.payload)))
				return fmt.Errorf("failed to write payload to memory")
			}

			_, err = e.snapshotFunc.Call(ctx, uint64(e.engine), pmPtr[0], uint64(len(s.payload)))
			if err != nil {
				e.deallocFunc.Call(ctx, uint64(pmPtr[0]), uint64(len(s.payload)))
				return fmt.Errorf("failed to update engine: %w", err)
			}

			// dont defer in loop to avoid stack overflow
			e.deallocFunc.Call(ctx, uint64(pmPtr[0]), uint64(len(s.payload)))
		}
	}
}

// startPolling starts the background polling for the given update interval.
func (e *EvaluationClient) startPolling(ctx context.Context) error {
	ticker := time.NewTicker(e.updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-e.done:
			return nil
		case <-ticker.C:
			e.mu.RLock()
			etag := e.etag
			e.mu.RUnlock()

			snapshot, err := e.fetch(ctx, etag)
			if err != nil {
				e.errChan <- err
				continue
			}

			e.snapshotChan <- snapshot
		}
	}
}

// fetch fetches the snapshot for the given namespace.
func (e *EvaluationClient) fetch(ctx context.Context, etag string) (snapshot, error) {
	url := fmt.Sprintf("%s/internal/v1/evaluation/snapshot/namespace/%s", e.url, e.namespace)
	if e.ref != "" {
		url = fmt.Sprintf("%s?reference=%s", url, e.ref)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return snapshot{}, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-flipt-accept-server-version", "1.47.0")

	if e.authentication != nil {
		switch auth := e.authentication.(type) {
		case clientTokenAuthentication:
			req.Header.Set("Authorization", "Bearer "+auth.Token)
		case jwtAuthentication:
			req.Header.Set("Authorization", "JWT "+auth.Token)
		}
	}

	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return snapshot{}, fetchError(fmt.Errorf("failed to fetch snapshot: %w", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		return snapshot{}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return snapshot{}, fetchError(fmt.Errorf("unexpected status code: %d", resp.StatusCode))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return snapshot{}, fetchError(fmt.Errorf("failed to read response body: %w", err))
	}

	etag = resp.Header.Get("ETag")
	return snapshot{payload: body, etag: etag}, nil
}

type streamChunk struct {
	Result streamResult
}

type streamResult struct {
	Namespaces map[string]json.RawMessage
}

// startStreaming starts the background streaming.
func (e *EvaluationClient) startStreaming(ctx context.Context) error {
	url := fmt.Sprintf("%s/internal/v1/evaluation/snapshots?[]namespaces=%s", e.url, e.namespace)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-flipt-accept-server-version", "1.47.0")

	if e.authentication != nil {
		switch auth := e.authentication.(type) {
		case clientTokenAuthentication:
			req.Header.Set("Authorization", "Bearer "+auth.Token)
		case jwtAuthentication:
			req.Header.Set("Authorization", "JWT "+auth.Token)
		}
	}

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return fetchError(fmt.Errorf("failed to fetch snapshot: %w", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fetchError(fmt.Errorf("unexpected status code: %d", resp.StatusCode))
	}

	reader := bufio.NewReader(resp.Body)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-e.done:
			return nil
		default:
			line, err := reader.ReadBytes('\n')
			if err != nil {
				if errors.Is(err, io.EOF) {
					return nil
				}
				return fetchError(fmt.Errorf("failed to read stream chunk: %w", err))
			}

			var chunk streamChunk
			if err := json.Unmarshal(line, &chunk); err != nil {
				return fetchError(fmt.Errorf("failed to unmarshal stream chunk: %w", err))
			}

			for ns, payload := range chunk.Result.Namespaces {
				if ns == e.namespace {
					e.snapshotChan <- snapshot{payload: payload}
				}
			}
		}
	}
}

// evaluateWASM evaluates the given function name with the given request.
func (e *EvaluationClient) evaluateWASM(ctx context.Context, funcName string, request any) ([]byte, error) {
	if e.engine == 0 {
		return nil, errors.New("engine not initialized")
	}

	reqBytes, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	reqPtr, err := e.allocFunc.Call(ctx, uint64(len(reqBytes)))
	if err != nil {
		return nil, fmt.Errorf("failed to allocate memory for request: %w", err)
	}
	defer e.deallocFunc.Call(ctx, reqPtr[0], uint64(len(reqBytes)))

	if !e.mod.Memory().Write(uint32(reqPtr[0]), reqBytes) {
		return nil, fmt.Errorf("failed to write request to memory")
	}

	evalFunc := e.mod.ExportedFunction(funcName)
	res, err := evalFunc.Call(ctx, uint64(e.engine), reqPtr[0], uint64(len(reqBytes)))
	if err != nil {
		return nil, fmt.Errorf("failed to call %s: %w", funcName, err)
	}

	if len(res) < 1 {
		return nil, fmt.Errorf("failed to call %s: no result returned", funcName)
	}

	ptr, len := decodePtr(res[0])
	defer e.deallocFunc.Call(ctx, uint64(ptr), uint64(len))

	b, ok := e.mod.Memory().Read(ptr, len)
	if !ok {
		return nil, fmt.Errorf("failed to read result from memory")
	}

	return b, nil
}
