package flipt

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
	defaultEnvironment = "default"
	defaultNamespace   = "default"
	statusSuccess      = "success"

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

// Client wraps the functionality of evaluating Flipt feature flags.
type Client struct {
	runtime wazero.Runtime
	mod     api.Module
	mu      sync.RWMutex

	engine uint32
	err    error

	cfg  config
	etag string

	wg     sync.WaitGroup
	cancel context.CancelFunc

	errChan      chan error
	snapshotChan chan snapshot

	closeOnce sync.Once

	exportedFuncs map[string]api.Function
}

// NewClient constructs a Client and performs an initial fetch of flag state.
func NewClient(ctx context.Context, opts ...Option) (_ *Client, err error) {
	cfg := config{
		Environment:    defaultEnvironment,
		Namespace:      defaultNamespace,
		URL:            "http://localhost:8080",
		UpdateInterval: 2 * time.Minute,
		FetchMode:      FetchModePolling,
		ErrorStrategy:  ErrorStrategyFail,
		HTTPClient:     defaultHTTPClient,
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("failed to validate configuration: %w", err)
	}

	runtime := wazero.NewRuntime(ctx)

	defer func() {
		if err != nil {
			_ = runtime.Close(ctx)
		}
	}()

	_, err = wasi_snapshot_preview1.Instantiate(ctx, runtime)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate WASI: %w", err)
	}

	compiled, err := runtime.CompileModule(ctx, wasm)
	if err != nil {
		return nil, fmt.Errorf("failed to compile and load evaluation engine: %w", err)
	}

	mod, err := runtime.InstantiateModule(ctx, compiled, wazero.NewModuleConfig().WithName("flipt_engine"))
	if err != nil {
		return nil, fmt.Errorf("can't instantiate Wasm module: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)

	c := &Client{
		runtime: runtime,
		mod:     mod,

		cfg: cfg,

		cancel:       cancel,
		errChan:      make(chan error, 1),
		snapshotChan: make(chan snapshot, 1),
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
	nsPtr, err := allocFunc.Call(ctx, uint64(len(c.cfg.Namespace)))
	if err != nil {
		return nil, fmt.Errorf("failed to allocate memory for namespace: %w", err)
	}

	if !mod.Memory().Write(uint32(nsPtr[0]), []byte(c.cfg.Namespace)) {
		return nil, fmt.Errorf("failed to write namespace to memory")
	}

	// Get initial snapshot URL (always uses polling endpoint)
	initialURL := fmt.Sprintf(c.cfg.URL+snapshotPath, c.cfg.Namespace)
	if c.cfg.Reference != "" {
		initialURL += "?reference=" + c.cfg.Reference
	}

	// fetch initial state
	snapshot, err := c.fetch(ctx, initialURL, "")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch initial state: %w", err)
	}

	// set initial etag
	c.etag = snapshot.etag

	// allocate payload
	pmPtr, err := allocFunc.Call(ctx, uint64(len(snapshot.payload)))
	if err != nil {
		return nil, fmt.Errorf("failed to allocate memory for payload: %w", err)
	}

	if !mod.Memory().Write(uint32(pmPtr[0]), []byte(snapshot.payload)) {
		return nil, fmt.Errorf("failed to write payload to memory")
	}

	// initialize engine
	res, err := initializeEngine.Call(ctx, nsPtr[0], uint64(len(c.cfg.Namespace)), pmPtr[0], uint64(len(snapshot.payload)))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize engine: %w", err)
	}

	if len(res) < 1 || res[0] == 0 {
		return nil, fmt.Errorf("failed to initialize engine: invalid pointer returned")
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
				if err != nil {
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
		switch cfg.FetchMode {
		case FetchModeStreaming:
			c.startStreaming(fctx)
		case FetchModePolling:
			c.startPolling(fctx)
		}
	}()

	return c, nil
}

// Err returns the last error that occurred.
func (c *Client) Err() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.err
}

// EvaluateVariant performs evaluation for a variant flag.
func (c *Client) EvaluateVariant(ctx context.Context, req *EvaluationRequest) (*VariantEvaluationResponse, error) {
	result, err := c.evaluateWASM(ctx, fEvaluateVariant, req)
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
func (c *Client) EvaluateBoolean(ctx context.Context, req *EvaluationRequest) (*BooleanEvaluationResponse, error) {
	result, err := c.evaluateWASM(ctx, fEvaluateBoolean, req)
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
func (c *Client) EvaluateBatch(ctx context.Context, requests []*EvaluationRequest) (*BatchEvaluationResponse, error) {
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
func (c *Client) ListFlags(ctx context.Context) ([]Flag, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.err != nil && c.cfg.ErrorStrategy == ErrorStrategyFail {
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
func (c *Client) Close(ctx context.Context) (err error) {
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

func (c *Client) handleUpdates(ctx context.Context) error {
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
func (c *Client) startPolling(ctx context.Context) {
	ticker := time.NewTicker(c.cfg.UpdateInterval)
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

func (c *Client) newRequest(ctx context.Context, url string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "flipt-client-go/"+Version)
	req.Header.Set("x-flipt-accept-server-version", "1.47.0")

	if c.cfg.Environment != "" {
		req.Header.Set("x-flipt-environment", c.cfg.Environment)
	}

	if c.cfg.Authentication != nil {
		switch auth := c.cfg.Authentication.(type) {
		case clientTokenAuthentication:
			req.Header.Set("Authorization", "Bearer "+auth.Token)
		case jwtAuthentication:
			req.Header.Set("Authorization", "JWT "+auth.Token)
		}
	}
	return req, nil
}

// fetch fetches the snapshot for the given namespace.
func (c *Client) fetch(ctx context.Context, url, etag string) (snapshot, error) {
	return doWithRetry(ctx, func(_ *int) (snapshot, error) {
		req, err := c.newRequest(ctx, url)
		if err != nil {
			return snapshot{}, err
		}
		if etag != "" {
			req.Header.Set("If-None-Match", etag)
		}

		resp, err := c.cfg.HTTPClient.Do(req)
		if resp != nil {
			defer func() {
				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()
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
			return snapshot{}, &networkError{msg: fmt.Sprintf("unexpected status code: %d", resp.StatusCode), status: resp.StatusCode}
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
func (c *Client) startStreaming(ctx context.Context) {
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

func (c *Client) initiateStream(ctx context.Context) error {
	url := c.getSnapshotURL()
	_, err := doWithRetry(ctx, func(attempt *int) (struct{}, error) {
		req, err := c.newRequest(ctx, url)
		if err != nil {
			return struct{}{}, err
		}

		resp, err := c.cfg.HTTPClient.Do(req)
		if err != nil {
			return struct{}{}, err
		}

		// Ensure response body is closed when we're done
		defer func() {
			_ = resp.Body.Close()
		}()

		if resp.StatusCode != http.StatusOK {
			return struct{}{}, &networkError{msg: fmt.Sprintf("unexpected status code: %d", resp.StatusCode), status: resp.StatusCode}
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

func (c *Client) handleStream(ctx context.Context, r io.ReadCloser) error {
	var (
		reader = bufio.NewReaderSize(r, 32*1024)
		buffer bytes.Buffer

		readErrCh = make(chan error, 1)
		readCh    = make(chan []byte, 1)
	)

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
						if ns == c.cfg.Namespace {
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
func (c *Client) evaluateWASM(ctx context.Context, funcName string, request any) ([]byte, error) {
	c.mu.RLock()
	if c.err != nil && c.cfg.ErrorStrategy == ErrorStrategyFail {
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

	// check for specific HTTP status codes if the error is a network error
	var netErr *networkError
	if errors.As(err, &netErr) {
		if netErr.Status() > 400 && (netErr.Status() == http.StatusTooManyRequests ||
			netErr.Status() == http.StatusBadGateway ||
			netErr.Status() == http.StatusServiceUnavailable ||
			netErr.Status() == http.StatusGatewayTimeout) {
			return true
		}
		return false
	}

	// check for network errors
	var netErr2 net.Error
	if errors.As(err, &netErr2) {
		return netErr2.Timeout()
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
func (c *Client) getSnapshotURL() string {
	switch c.cfg.FetchMode {
	case FetchModeStreaming:
		// Streaming uses a different endpoint that accepts multiple namespaces
		return fmt.Sprintf(c.cfg.URL+streamingPath, c.cfg.Namespace)
	case FetchModePolling:
		// Polling uses the single namespace snapshot endpoint
		url := fmt.Sprintf(c.cfg.URL+snapshotPath, c.cfg.Namespace)
		if c.cfg.Reference != "" {
			url += "?reference=" + c.cfg.Reference
		}
		return url
	default:
		// This should never happen as we validate fetch mode during initialization
		return ""
	}
}
