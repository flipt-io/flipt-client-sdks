package evaluation

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	_ "embed"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"golang.org/x/net/http2"
)

//go:embed ext/flipt_engine_wasm.wasm
var wasm []byte

const (
	defaultNamespace = "default"
	statusSuccess    = "success"

	fInitializeEngine = "initialize_engine"
	fAllocate         = "allocate"
	fDeallocate       = "deallocate"
	fSnapshot         = "snapshot"
	fEvaluateVariant  = "evaluate_variant"
	fEvaluateBoolean  = "evaluate_boolean"
	fEvaluateBatch    = "evaluate_batch"
	fListFlags        = "list_flags"
	fDestroyEngine    = "destroy_engine"

	// API paths
	snapshotPath  = "/internal/v1/evaluation/snapshot/namespace/%s"
	streamingPath = "/internal/v1/evaluation/snapshots?[]namespaces=%s"
)

// EvaluationClient wraps the functionality of evaluating Flipt feature flags.
type EvaluationClient struct {
	runtime wazero.Runtime
	mod     api.Module
	mu      sync.RWMutex

	engine    uint32
	namespace string
	err       error

	baseURL        string
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

	exportedFuncs map[string]api.Function
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
		c.baseURL = url
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

type fetchError struct {
	code int
	err  error
}

func (e fetchError) Error() string {
	if e.err == nil {
		return fmt.Sprintf("fetch error: %d", e.code)
	}
	return e.err.Error()
}

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

	c := &EvaluationClient{
		runtime: runtime,
		mod:     mod,

		// default values
		namespace:      defaultNamespace,
		baseURL:        "http://localhost:8080",
		updateInterval: 2 * time.Minute, // default 120 seconds
		errorStrategy:  ErrorStrategyFail,
		fetchMode:      FetchModePolling,
		requestTimeout: 30 * time.Second, // default timeout

		cancel:       cancel,
		errChan:      make(chan error, 1),
		snapshotChan: make(chan snapshot, 1),
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.namespace == "" {
		cerr = fmt.Errorf("namespace cannot be empty")
		return
	}

	if c.baseURL == "" {
		cerr = fmt.Errorf("baseURL cannot be empty")
		return
	}

	if c.updateInterval <= 0 && c.fetchMode == FetchModePolling {
		cerr = fmt.Errorf("update interval must be greater than 0")
		return
	}

	if c.requestTimeout <= 0 && c.fetchMode == FetchModePolling {
		cerr = fmt.Errorf("request timeout must be greater than 0")
		return
	}

	c.httpClient = &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 5 * time.Second, // More aggressive TCP keepalive
			}).DialContext,
			IdleConnTimeout:       60 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   10,
			// Enable HTTP/2 ping frames and other settings
			Proxy:           http.ProxyFromEnvironment,
			ReadBufferSize:  32 * 1024,
			WriteBufferSize: 32 * 1024,
			// HTTP/2 specific settings
			MaxResponseHeaderBytes: 4 * 1024, // 4KB
			DisableCompression:     false,    // Enable compression
		},
	}

	// Store the base URL without trailing slash
	c.baseURL = strings.TrimRight(c.baseURL, "/")

	// Set timeout only for polling mode
	if c.fetchMode == FetchModePolling {
		c.httpClient.Timeout = c.requestTimeout
	}

	// Validate fetch mode
	if c.fetchMode != FetchModePolling && c.fetchMode != FetchModeStreaming {
		return nil, fmt.Errorf("invalid fetch mode: %s", c.fetchMode)
	}

	// Get initial snapshot URL (always uses polling endpoint)
	initialURL := fmt.Sprintf(c.baseURL+snapshotPath, c.namespace)
	if c.ref != "" {
		initialURL += "?reference=" + c.ref
	}

	var (
		initializeEngine = mod.ExportedFunction(fInitializeEngine)
		allocFunc        = mod.ExportedFunction(fAllocate)
		deallocFunc      = mod.ExportedFunction(fDeallocate)
	)

	c.exportedFuncs = map[string]api.Function{
		fAllocate:        allocFunc,
		fDeallocate:      deallocFunc,
		fEvaluateVariant: mod.ExportedFunction(fEvaluateVariant),
		fEvaluateBoolean: mod.ExportedFunction(fEvaluateBoolean),
		fEvaluateBatch:   mod.ExportedFunction(fEvaluateBatch),
		fListFlags:       mod.ExportedFunction(fListFlags),
	}

	// allocate namespace
	nsPtr, err := allocFunc.Call(ctx, uint64(len(c.namespace)))
	if err != nil {
		cerr = fmt.Errorf("failed to allocate memory for namespace: %w", err)
		return
	}

	if !mod.Memory().Write(uint32(nsPtr[0]), []byte(c.namespace)) {
		cerr = fmt.Errorf("failed to write namespace to memory")
		return
	}

	// fetch initial state
	snapshot, err := c.fetch(ctx, initialURL, "")
	if err != nil {
		cerr = fmt.Errorf("failed to fetch initial state: %w", err)
		return
	}

	// set initial etag
	c.etag = snapshot.etag

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
	res, err := initializeEngine.Call(ctx, nsPtr[0], uint64(len(c.namespace)), pmPtr[0], uint64(len(snapshot.payload)))
	if err != nil {
		cerr = fmt.Errorf("failed to initialize engine: %w", err)
		return
	}

	if len(res) < 1 || res[0] == 0 {
		cerr = fmt.Errorf("failed to initialize engine: invalid pointer returned")
		return
	}

	c.engine = uint32(res[0])

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case err := <-c.errChan:
				if err != nil && c.errorStrategy == ErrorStrategyFail {
					c.mu.Lock()
					c.err = err
					c.mu.Unlock()
				}
			}
		}
	}()

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		if err := c.handleUpdates(ctx); err != nil {
			c.errChan <- err
		}
	}()

	// create a new context for the http operations so any timeouts
	// are independent of the client context
	fctx, cancel := context.WithCancel(context.Background())

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		<-ctx.Done()
		// cancel the http context so any pending requests are cancelled if we're closing
		cancel()
	}()

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		switch c.fetchMode {
		case FetchModeStreaming:
			c.startStreaming(fctx)
		case FetchModePolling:
			c.startPolling(fctx)
		}
	}()

	return c, nil
}

// Err returns the last error that occurred.
func (c *EvaluationClient) Err() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.err
}

// EvaluateVariant performs evaluation for a variant flag.
func (c *EvaluationClient) EvaluateVariant(ctx context.Context, flagKey, entityID string, evalContext map[string]string) (*VariantEvaluationResponse, error) {
	request := EvaluationRequest{
		FlagKey:  flagKey,
		EntityId: entityID,
		Context:  evalContext,
	}

	result, err := c.evaluateWASM(ctx, fEvaluateVariant, request)
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
func (c *EvaluationClient) EvaluateBoolean(ctx context.Context, flagKey, entityID string, evalContext map[string]string) (*BooleanEvaluationResponse, error) {
	request := EvaluationRequest{
		FlagKey:  flagKey,
		EntityId: entityID,
		Context:  evalContext,
	}

	result, err := c.evaluateWASM(ctx, fEvaluateBoolean, request)
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
func (c *EvaluationClient) EvaluateBatch(ctx context.Context, requests []*EvaluationRequest) (*BatchEvaluationResponse, error) {
	result, err := c.evaluateWASM(ctx, fEvaluateBatch, requests)
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
func (c *EvaluationClient) ListFlags(ctx context.Context) ([]Flag, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.err != nil && c.errorStrategy == ErrorStrategyFail {
		return nil, c.err
	}

	listFlagsFunc, ok := c.exportedFuncs[fListFlags]
	if !ok {
		return nil, errors.New("failed to find list_flags function")
	}

	res, err := listFlagsFunc.Call(ctx, uint64(c.engine))
	if err != nil {
		return nil, fmt.Errorf("failed to call list_flags: %w", err)
	}

	if len(res) < 1 {
		return nil, fmt.Errorf("failed to call list_flags: no result returned")
	}

	var (
		ptr, length = decodePtr(res[0])
		deallocFunc = c.exportedFuncs[fDeallocate]
	)

	b, ok := c.mod.Memory().Read(ptr, length)
	if !ok {
		_, _ = deallocFunc.Call(ctx, uint64(ptr), uint64(length))
		return nil, fmt.Errorf("failed to read result from memory")
	}

	// make a copy of the result before deallocating
	result := make([]byte, len(b))
	copy(result, b)

	// clean up memory
	if _, err := deallocFunc.Call(ctx, uint64(ptr), uint64(length)); err != nil {
		return nil, fmt.Errorf("failed to deallocate memory: %w", err)
	}

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
func (c *EvaluationClient) Close(ctx context.Context) (err error) {
	c.closeOnce.Do(func() {
		// signal background goroutines to stop
		c.cancel()

		// wait for all background goroutines to finish
		c.wg.Wait()

		// close channels
		close(c.errChan)

		if c.engine != 0 {
			// destroy engine
			dealloc := c.mod.ExportedFunction(fDestroyEngine)
			_, _ = dealloc.Call(ctx, uint64(c.engine))
		}

		// closing the runtime will close the module too and deallocate all memory
		err = c.runtime.Close(ctx)
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

func (c *EvaluationClient) handleUpdates(ctx context.Context) error {
	var (
		allocFunc    = c.exportedFuncs[fAllocate]
		deallocFunc  = c.exportedFuncs[fDeallocate]
		snapshotFunc = c.mod.ExportedFunction(fSnapshot)
	)

	for {
		select {
		case <-ctx.Done():
			close(c.snapshotChan)
			return ctx.Err()
		case s, ok := <-c.snapshotChan:
			if !ok {
				return nil
			}

			c.mu.Lock()
			c.etag = s.etag
			c.mu.Unlock()

			if len(s.payload) == 0 {
				c.mu.Lock()
				c.err = nil
				c.mu.Unlock()
				continue
			}

			c.mu.Lock()
			// allocate memory for the new payload
			pmPtr, err := allocFunc.Call(ctx, uint64(len(s.payload)))
			if err != nil {
				c.mu.Unlock()
				return fmt.Errorf("failed to allocate memory for payload: %w", err)
			}

			// write the new payload to memory
			if !c.mod.Memory().Write(uint32(pmPtr[0]), s.payload) {
				c.mu.Unlock()
				_, _ = deallocFunc.Call(ctx, uint64(pmPtr[0]), uint64(len(s.payload)))
				return fmt.Errorf("failed to write payload to memory")
			}

			// update the engine with the new snapshot while holding the lock
			res, err := snapshotFunc.Call(ctx, uint64(c.engine), pmPtr[0], uint64(len(s.payload)))

			ptr, length := decodePtr(res[0])
			// always deallocate the memory after we're done with it
			if _, err := deallocFunc.Call(ctx, uint64(pmPtr[0]), uint64(len(s.payload))); err != nil {
				c.mu.Unlock()
				return fmt.Errorf("failed to deallocate memory: %w", err)
			}
			if _, err := deallocFunc.Call(ctx, uint64(ptr), uint64(length)); err != nil {
				c.mu.Unlock()
				return fmt.Errorf("failed to deallocate memory: %w", err)
			}
			c.err = nil
			c.mu.Unlock()
			if err != nil {
				return fmt.Errorf("failed to update engine: %w", err)
			}
		}
	}
}

// startPolling starts the background polling for the given update interval.
func (c *EvaluationClient) startPolling(ctx context.Context) {
	ticker := time.NewTicker(c.updateInterval)
	defer ticker.Stop()

	url := c.getSnapshotURL()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.mu.RLock()
			etag := c.etag
			c.mu.RUnlock()

			snapshot, err := c.fetch(ctx, url, etag)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				c.errChan <- err
				continue
			}

			c.snapshotChan <- snapshot
		}
	}
}

const (
	maxRetries = 3
	baseDelay  = 1 * time.Second
	maxDelay   = 30 * time.Second
)

// doWithRetry executes the given function with exponential backoff and jitter.
// It will retry up to maxRetries times if the function returns a transient error.
// The function can modify the attempt counter through the pointer, useful for long-lived
// connections that want to reset the counter on success.
func doWithRetry[T any](ctx context.Context, fn func(attempt *int) (T, error)) (T, error) {
	var (
		lastErr error
		zero    T
		attempt int
	)

	for {
		if attempt > 0 {
			// calculate exponential backoff with jitter
			delay := min(baseDelay*time.Duration(1<<uint(attempt)), maxDelay)
			// add jitter: ±10% of the delay
			jitter := time.Duration(rand.Float64()*float64(delay)*0.2 - float64(delay)*0.1)
			delay += jitter

			select {
			case <-ctx.Done():
				return zero, ctx.Err()
			case <-time.After(delay):
			}
		}

		result, err := fn(&attempt)
		if err == nil {
			return result, nil
		}

		lastErr = err
		if !isTransientError(err) {
			return zero, err
		}

		attempt++
		if attempt >= maxRetries {
			return zero, fmt.Errorf("failed after %d retries, last error: %w", maxRetries, lastErr)
		}
	}
}

func (c *EvaluationClient) newRequest(ctx context.Context, url string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "flipt-client-go/"+Version)
	req.Header.Set("x-flipt-accept-server-version", "1.47.0")

	if c.authentication != nil {
		switch auth := c.authentication.(type) {
		case clientTokenAuthentication:
			req.Header.Set("Authorization", "Bearer "+auth.Token)
		case jwtAuthentication:
			req.Header.Set("Authorization", "JWT "+auth.Token)
		}
	}
	return req, nil
}

// fetch fetches the snapshot for the given namespace.
func (c *EvaluationClient) fetch(ctx context.Context, url, etag string) (snapshot, error) {
	return doWithRetry(ctx, func(_ *int) (snapshot, error) {
		req, err := c.newRequest(ctx, url)
		if err != nil {
			return snapshot{}, err
		}
		if etag != "" {
			req.Header.Set("If-None-Match", etag)
		}

		resp, err := c.httpClient.Do(req)
		if resp != nil {
			defer func() {
				_, _ = io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}()
		}

		if err != nil {
			return snapshot{}, err
		}

		// always read the entire body so the connection can be reused
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return snapshot{}, err
		}

		if resp.StatusCode == http.StatusNotModified {
			return snapshot{etag: etag}, nil
		}

		if resp.StatusCode != http.StatusOK {
			return snapshot{}, fetchError{resp.StatusCode, fmt.Errorf("unexpected status code: %d", resp.StatusCode)}
		}

		etag = resp.Header.Get("ETag")
		return snapshot{payload: body, etag: etag}, nil
	})
}

type streamChunk struct {
	Result streamResult
}

type streamResult struct {
	Namespaces map[string]json.RawMessage
}

// startStreaming starts the background streaming.
func (c *EvaluationClient) startStreaming(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if err := c.initiateStream(ctx); err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				c.errChan <- err
			}
		}
	}
}

func (c *EvaluationClient) initiateStream(ctx context.Context) error {
	url := c.getSnapshotURL()
	_, err := doWithRetry(ctx, func(attempt *int) (struct{}, error) {
		req, err := c.newRequest(ctx, url)
		if err != nil {
			return struct{}{}, err
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return struct{}{}, err
		}

		// Ensure response body is closed when we're done
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return struct{}{}, fetchError{resp.StatusCode, fmt.Errorf("unexpected status code: %d", resp.StatusCode)}
		}

		// Reset attempt counter on successful connection
		*attempt = 0

		if err := c.handleStream(ctx, resp.Body); err != nil && ctx.Err() == nil {
			return struct{}{}, err
		}

		return struct{}{}, nil
	})
	return err
}

func (c *EvaluationClient) handleStream(ctx context.Context, r io.ReadCloser) error {
	reader := bufio.NewReaderSize(r, 32*1024)
	var buffer bytes.Buffer

	readErrCh := make(chan error, 1)
	readCh := make(chan []byte, 1)

	// Start a goroutine to handle reading
	go func() {
		for {
			chunk, err := reader.ReadBytes('\n')
			if err != nil {
				readErrCh <- err
				return
			}
			readCh <- chunk
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-readErrCh:
			if err == io.EOF {
				return nil
			}
			return err
		case chunk := <-readCh:
			if len(chunk) > 0 {
				buffer.Write(chunk)
				if json.Valid(buffer.Bytes()) {
					var chunk streamChunk
					if err := json.Unmarshal(buffer.Bytes(), &chunk); err != nil {
						return fmt.Errorf("failed to unmarshal stream chunk: %w", err)
					}

					for ns, payload := range chunk.Result.Namespaces {
						if ns == c.namespace {
							select {
							case <-ctx.Done():
								return ctx.Err()
							case c.snapshotChan <- snapshot{payload: payload}:
							}
						}
					}
					buffer.Reset()
				}
			}
		}
	}
}

// evaluateWASM evaluates the given function name with the given request.
func (c *EvaluationClient) evaluateWASM(ctx context.Context, funcName string, request any) ([]byte, error) {
	c.mu.RLock()
	if c.err != nil && c.errorStrategy == ErrorStrategyFail {
		c.mu.RUnlock()
		return nil, c.err
	}
	c.mu.RUnlock()

	reqBytes, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var (
		allocFunc   = c.exportedFuncs[fAllocate]
		deallocFunc = c.exportedFuncs[fDeallocate]
	)

	c.mu.Lock()
	defer c.mu.Unlock()

	reqPtr, err := allocFunc.Call(ctx, uint64(len(reqBytes)))
	if err != nil {
		return nil, fmt.Errorf("failed to allocate memory for request: %w", err)
	}

	if !c.mod.Memory().Write(uint32(reqPtr[0]), reqBytes) {
		_, _ = deallocFunc.Call(ctx, reqPtr[0], uint64(len(reqBytes)))
		return nil, fmt.Errorf("failed to write request to memory")
	}

	evalFunc, ok := c.exportedFuncs[funcName]
	if !ok {
		_, _ = deallocFunc.Call(ctx, reqPtr[0], uint64(len(reqBytes)))
		return nil, fmt.Errorf("failed to call %s: function not found", funcName)
	}

	res, err := evalFunc.Call(ctx, uint64(c.engine), reqPtr[0], uint64(len(reqBytes)))
	if err != nil {
		_, _ = deallocFunc.Call(ctx, reqPtr[0], uint64(len(reqBytes)))
		return nil, fmt.Errorf("failed to call %s: %w", funcName, err)
	}

	// clean up request memory
	if _, err := deallocFunc.Call(ctx, reqPtr[0], uint64(len(reqBytes))); err != nil {
		return nil, fmt.Errorf("failed to deallocate memory: %w", err)
	}

	if len(res) < 1 {
		return nil, fmt.Errorf("failed to call %s: no result returned", funcName)
	}

	ptr, length := decodePtr(res[0])
	b, ok := c.mod.Memory().Read(ptr, length)
	if !ok {
		_, _ = deallocFunc.Call(ctx, uint64(ptr), uint64(length))
		return nil, fmt.Errorf("failed to read result from memory")
	}

	// make a copy of the result before deallocating
	result := make([]byte, len(b))
	copy(result, b)

	// clean up result memory
	if _, err := deallocFunc.Call(ctx, uint64(ptr), uint64(length)); err != nil {
		return nil, fmt.Errorf("failed to deallocate memory: %w", err)
	}

	return result, nil
}

// isTransientError determines if an error is transient and should be retried
func isTransientError(err error) bool {
	if err == nil {
		return false
	}

	// check for specific HTTP status codes if the error is a fetchError
	var fetchErr fetchError
	if errors.As(err, &fetchErr) {
		if fetchErr.code > 400 && (fetchErr.code == http.StatusTooManyRequests ||
			fetchErr.code == http.StatusBadGateway ||
			fetchErr.code == http.StatusServiceUnavailable ||
			fetchErr.code == http.StatusGatewayTimeout) {
			return true
		}
		return false
	}

	// check for network errors
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout()
	}

	// check for specific network operation errors
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return opErr.Temporary() || opErr.Timeout()
	}

	// check for DNS temporary failures
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return dnsErr.Temporary()
	}

	// some errors are always considered temporary
	if errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}

	// Check for HTTP/2 stream errors
	var streamErr *http2.StreamError
	if errors.As(err, &streamErr) {
		switch streamErr.Code {
		case http2.ErrCodeInternal, // INTERNAL_ERROR
			http2.ErrCodeProtocol,        // PROTOCOL_ERROR
			http2.ErrCodeEnhanceYourCalm: // ENHANCE_YOUR_CALM
			return true
		}
	}

	// Check for HTTP/2 GoAway errors
	var goAwayErr *http2.GoAwayError
	if errors.As(err, &goAwayErr) {
		// Consider connection-level issues as transient
		switch goAwayErr.ErrCode {
		case http2.ErrCodeNo,
			http2.ErrCodeInternal,
			http2.ErrCodeFlowControl,
			http2.ErrCodeEnhanceYourCalm:
			return true
		}
	}

	return false
}

// getSnapshotURL returns the URL for fetching snapshots based on the fetch mode
func (c *EvaluationClient) getSnapshotURL() string {
	switch c.fetchMode {
	case FetchModeStreaming:
		// Streaming uses a different endpoint that accepts multiple namespaces
		return fmt.Sprintf(c.baseURL+streamingPath, c.namespace)
	case FetchModePolling:
		// Polling uses the single namespace snapshot endpoint
		url := fmt.Sprintf(c.baseURL+snapshotPath, c.namespace)
		if c.ref != "" {
			url += "?reference=" + c.ref
		}
		return url
	default:
		// This should never happen as we validate fetch mode during initialization
		return ""
	}
}
