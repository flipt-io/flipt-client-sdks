package evaluation

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	_ "embed"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

//go:embed ext/flipt_engine_wasm.wasm
var wasm []byte

const (
	statusSuccess = "success"

	fInitializeEngine = "initialize_engine"
	fAllocate         = "allocate"
	fDeallocate       = "deallocate"
	fSnapshot         = "snapshot"
	fEvaluateVariant  = "evaluate_variant"
	fEvaluateBoolean  = "evaluate_boolean"
	fEvaluateBatch    = "evaluate_batch"
	fListFlags        = "list_flags"
	fDestroyEngine    = "destroy_engine"
)

// EvaluationClient wraps the functionality of evaluating Flipt feature flags.
type EvaluationClient struct {
	runtime wazero.Runtime
	mod     api.Module
	mu      sync.RWMutex

	engine    uint32
	namespace string
	err       error

	url            string
	authentication any
	ref            string
	updateInterval time.Duration
	fetchMode      FetchMode
	errorStrategy  ErrorStrategy

	httpClient     *http.Client
	requestTimeout time.Duration
	etag           string

	wg     sync.WaitGroup
	cancel context.CancelFunc

	errChan      chan error
	snapshotChan chan snapshot

	closeOnce sync.Once
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
		c.updateInterval = updateInterval
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

// WithRequestTimeout allows for specifying the request timeout for the Flipt client.
// Note: this only affects polling mode. Streaming mode will have no timeout set.
func WithRequestTimeout(timeout time.Duration) ClientOption {
	return func(c *EvaluationClient) {
		c.requestTimeout = timeout
	}
}

type fetchError error

// NewEvaluationClient constructs a Client and performs an initial fetch of flag state.
func NewEvaluationClient(ctx context.Context, opts ...ClientOption) (_ *EvaluationClient, cerr error) {
	runtime := wazero.NewRuntime(ctx)

	// cleanup function that only runs if we return an error
	defer func() {
		if cerr != nil {
			// closing the runtime will release any allocated memory
			runtime.Close(ctx)
		}
	}()

	_, err := wasi_snapshot_preview1.Instantiate(ctx, runtime)
	if err != nil {
		cerr = fmt.Errorf("failed to instantiate WASI: %w", err)
		return
	}

	compiled, err := runtime.CompileModule(ctx, wasm)
	if err != nil {
		cerr = fmt.Errorf("failed to compile and load evaluation engine: %w", err)
		return
	}

	cfg := wazero.NewModuleConfig().WithName("flipt_engine")
	mod, err := runtime.InstantiateModule(ctx, compiled, cfg)
	if err != nil {
		cerr = fmt.Errorf("can't instantiate Wasm module: %w", err)
		return
	}

	ctx, cancel := context.WithCancel(ctx)

	client := &EvaluationClient{
		runtime: runtime,
		mod:     mod,

		// default values
		namespace:      "default",
		url:            "http://localhost:8080",
		updateInterval: 2 * time.Minute, // default 120 seconds
		errorStrategy:  ErrorStrategyFail,
		fetchMode:      FetchModePolling,
		requestTimeout: 30 * time.Second, // default timeout

		cancel:       cancel,
		errChan:      make(chan error, 1),
		snapshotChan: make(chan snapshot, 1),
	}

	for _, opt := range opts {
		opt(client)
	}

	if client.namespace == "" {
		cerr = fmt.Errorf("namespace cannot be empty")
		return
	}

	if client.url == "" {
		cerr = fmt.Errorf("url cannot be empty")
		return
	}

	if client.updateInterval <= 0 && client.fetchMode == FetchModePolling {
		cerr = fmt.Errorf("update interval must be greater than 0")
		return
	}

	if client.requestTimeout <= 0 && client.fetchMode == FetchModePolling {
		cerr = fmt.Errorf("request timeout must be greater than 0")
		return
	}

	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second, // connection timeout
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}

	client.httpClient = &http.Client{
		Transport: transport,
	}

	// only set timeout for polling mode
	// streaming mode will have no timeout set
	if client.fetchMode == FetchModePolling {
		client.httpClient.Timeout = client.requestTimeout
	}

	var (
		initializeEngine = mod.ExportedFunction(fInitializeEngine)
		allocFunc        = mod.ExportedFunction(fAllocate)
	)

	// allocate namespace
	nsPtr, err := allocFunc.Call(ctx, uint64(len(client.namespace)))
	if err != nil {
		cerr = fmt.Errorf("failed to allocate memory for namespace: %w", err)
		return
	}

	if !mod.Memory().Write(uint32(nsPtr[0]), []byte(client.namespace)) {
		cerr = fmt.Errorf("failed to write namespace to memory")
		return
	}

	// fetch initial state
	snapshot, err := client.fetch(ctx, "")
	if err != nil {
		cerr = fmt.Errorf("failed to fetch initial state: %w", err)
		return
	}

	// set initial etag
	client.etag = snapshot.etag

	// allocate payload
	pmPtr, err := allocFunc.Call(ctx, uint64(len(snapshot.payload)))
	if err != nil {
		cerr = fmt.Errorf("failed to allocate memory for payload: %w", err)
		return
	}

	if !mod.Memory().Write(uint32(pmPtr[0]), []byte(snapshot.payload)) {
		cerr = fmt.Errorf("failed to write payload to memory")
		return
	}

	// initialize engine
	res, err := initializeEngine.Call(ctx, nsPtr[0], uint64(len(client.namespace)), pmPtr[0], uint64(len(snapshot.payload)))
	if err != nil {
		cerr = fmt.Errorf("failed to initialize engine: %w", err)
		return
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
				client.cancel()
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
			client.startStreaming(ctx)
		case FetchModePolling:
			client.startPolling(ctx)
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

	result, err := e.evaluateWASM(ctx, fEvaluateVariant, request)
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

	result, err := e.evaluateWASM(ctx, fEvaluateBoolean, request)
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

	result, err := e.evaluateWASM(ctx, fEvaluateBatch, requests)
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
	defer e.mu.RUnlock()

	if e.err != nil && e.errorStrategy == ErrorStrategyFail {
		return nil, e.err
	}

	if e.engine == 0 {
		return nil, errors.New("engine not initialized")
	}

	evalFunc := e.mod.ExportedFunction(fListFlags)
	res, err := evalFunc.Call(ctx, uint64(e.engine))
	if err != nil {
		return nil, fmt.Errorf("failed to call list_flags: %w", err)
	}

	if len(res) < 1 {
		return nil, fmt.Errorf("failed to call list_flags: no result returned")
	}

	ptr, length := decodePtr(res[0])
	deallocFunc := e.mod.ExportedFunction(fDeallocate)

	b, ok := e.mod.Memory().Read(ptr, length)
	if !ok {
		deallocFunc.Call(ctx, uint64(ptr), uint64(length))
		return nil, fmt.Errorf("failed to read result from memory")
	}

	// make a copy of the result before deallocating
	result := make([]byte, len(b))
	copy(result, b)

	// clean up memory
	deallocFunc.Call(ctx, uint64(ptr), uint64(length))

	var listResult *ListFlagsResult
	if err := json.Unmarshal(result, &listResult); err != nil {
		return nil, fmt.Errorf("failed to unmarshal flags: %w", err)
	}

	if listResult == nil {
		return nil, errors.New("failed to unmarshal flags: nil")
	}

	if listResult.Status != statusSuccess {
		return nil, errors.New(listResult.ErrorMessage)
	}

	return *listResult.Result, nil
}

// Close cleans up the allocated resources.
func (e *EvaluationClient) Close(ctx context.Context) (err error) {
	e.closeOnce.Do(func() {
		// signal background goroutines to stop
		e.cancel()

		// wait for all background goroutines to finish
		e.wg.Wait()

		// close channels
		close(e.errChan)

		if e.engine != 0 {
			// destroy engine
			dealloc := e.mod.ExportedFunction(fDestroyEngine)
			dealloc.Call(ctx, uint64(e.engine))
		}

		// closing the runtime will close the module too and deallocate all memory
		err = e.runtime.Close(ctx)
	})

	return
}

// unexported

func decodePtr(ptr uint64) (uint32, uint32) {
	return uint32(ptr >> 32), uint32(ptr)
}

type snapshot struct {
	payload []byte
	etag    string
}

func (e *EvaluationClient) handleUpdates(ctx context.Context) error {
	var (
		allocFunc    = e.mod.ExportedFunction(fAllocate)
		deallocFunc  = e.mod.ExportedFunction(fDeallocate)
		snapshotFunc = e.mod.ExportedFunction(fSnapshot)
	)

	for {
		select {
		case <-ctx.Done():
			close(e.snapshotChan)
			return nil
		case s, ok := <-e.snapshotChan:
			if !ok {
				return nil
			}

			e.mu.Lock()
			e.etag = s.etag
			e.mu.Unlock()

			if len(s.payload) == 0 {
				continue
			}

			e.mu.Lock()
			// allocate memory for the new payload
			pmPtr, err := allocFunc.Call(ctx, uint64(len(s.payload)))
			if err != nil {
				e.mu.Unlock()
				return fmt.Errorf("failed to allocate memory for payload: %w", err)
			}

			// write the new payload to memory
			if !e.mod.Memory().Write(uint32(pmPtr[0]), s.payload) {
				e.mu.Unlock()
				deallocFunc.Call(ctx, uint64(pmPtr[0]), uint64(len(s.payload)))
				return fmt.Errorf("failed to write payload to memory")
			}

			// update the engine with the new snapshot while holding the lock
			res, err := snapshotFunc.Call(ctx, uint64(e.engine), pmPtr[0], uint64(len(s.payload)))

			ptr, length := decodePtr(res[0])
			// always deallocate the memory after we're done with it
			deallocFunc.Call(ctx, uint64(pmPtr[0]), uint64(len(s.payload)))
			deallocFunc.Call(ctx, uint64(ptr), uint64(length))

			e.mu.Unlock()
			if err != nil {
				return fmt.Errorf("failed to update engine: %w", err)
			}
		}
	}
}

// startPolling starts the background polling for the given update interval.
func (e *EvaluationClient) startPolling(ctx context.Context) {
	ticker := time.NewTicker(e.updateInterval)
	defer ticker.Stop()

	// create a new context for the polling so any timeouts
	// are independent of the client context
	fctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.mu.RLock()
			etag := e.etag
			e.mu.RUnlock()

			snapshot, err := e.fetch(fctx, etag)
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
	if resp != nil {
		defer func() {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}()
	}
	if err != nil {
		return snapshot{}, fetchError(fmt.Errorf("failed to fetch snapshot: %w", err))
	}

	// always read the entire body so the connection can be reused
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return snapshot{}, fetchError(fmt.Errorf("failed to read response body: %w", err))
	}

	if resp.StatusCode == http.StatusNotModified {
		return snapshot{}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return snapshot{}, fetchError(fmt.Errorf("unexpected status code: %d", resp.StatusCode))
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
// Note: currently any errors cause this method to exit. We still need to implement retries for trying to reconnect.
func (e *EvaluationClient) startStreaming(ctx context.Context) {
	url := fmt.Sprintf("%s/internal/v1/evaluation/snapshots?[]namespaces=%s", e.url, e.namespace)

	// create a new context for the streaming so any timeouts
	// are independent of the client context
	fctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, err := http.NewRequestWithContext(fctx, "GET", url, nil)
	if err != nil {
		e.errChan <- fmt.Errorf("failed to create request: %w", err)
		return
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
	if resp != nil {
		defer func() {
			resp.Body.Close()
		}()
	}
	if err != nil {
		e.errChan <- fetchError(fmt.Errorf("failed to fetch snapshot: %w", err))
		return
	}

	if resp.StatusCode != http.StatusOK {
		e.errChan <- fetchError(fmt.Errorf("unexpected status code: %d", resp.StatusCode))
		return
	}

	reader := bufio.NewReader(resp.Body)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// create a channel to receive the read result
			readChan := make(chan struct {
				line []byte
				err  error
			})

			// start a goroutine to perform the blocking read
			go func() {
				line, err := reader.ReadBytes('\n')
				readChan <- struct {
					line []byte
					err  error
				}{line, err}
			}()

			// wait for either the read to complete or context cancellation
			select {
			case <-ctx.Done():
				return
			case result := <-readChan:
				if result.err != nil {
					if errors.Is(result.err, io.EOF) {
						return
					}
					e.errChan <- fetchError(fmt.Errorf("failed to read stream chunk: %w", result.err))
					return
				}

				var chunk streamChunk
				if err := json.Unmarshal(result.line, &chunk); err != nil {
					e.errChan <- fmt.Errorf("failed to unmarshal stream chunk: %w", err)
					return
				}

				for ns, payload := range chunk.Result.Namespaces {
					if ns == e.namespace {
						e.snapshotChan <- snapshot{payload: payload}
					}
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

	var (
		allocFunc   = e.mod.ExportedFunction(fAllocate)
		deallocFunc = e.mod.ExportedFunction(fDeallocate)
	)

	reqBytes, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	e.mu.Lock()
	reqPtr, err := allocFunc.Call(ctx, uint64(len(reqBytes)))
	if err != nil {
		e.mu.Unlock()
		return nil, fmt.Errorf("failed to allocate memory for request: %w", err)
	}

	if !e.mod.Memory().Write(uint32(reqPtr[0]), reqBytes) {
		deallocFunc.Call(ctx, reqPtr[0], uint64(len(reqBytes)))
		e.mu.Unlock()
		return nil, fmt.Errorf("failed to write request to memory")
	}

	evalFunc := e.mod.ExportedFunction(funcName)
	res, err := evalFunc.Call(ctx, uint64(e.engine), reqPtr[0], uint64(len(reqBytes)))
	if err != nil {
		deallocFunc.Call(ctx, reqPtr[0], uint64(len(reqBytes)))
		e.mu.Unlock()
		return nil, fmt.Errorf("failed to call %s: %w", funcName, err)
	}

	// clean up request memory
	deallocFunc.Call(ctx, reqPtr[0], uint64(len(reqBytes)))

	if len(res) < 1 {
		e.mu.Unlock()
		return nil, fmt.Errorf("failed to call %s: no result returned", funcName)
	}

	ptr, length := decodePtr(res[0])
	b, ok := e.mod.Memory().Read(ptr, length)
	if !ok {
		deallocFunc.Call(ctx, uint64(ptr), uint64(length))
		e.mu.Unlock()
		return nil, fmt.Errorf("failed to read result from memory")
	}

	// make a copy of the result before deallocating
	result := make([]byte, len(b))
	copy(result, b)

	// clean up result memory
	deallocFunc.Call(ctx, uint64(ptr), uint64(length))
	e.mu.Unlock()

	return result, nil
}
