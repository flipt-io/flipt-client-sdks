package evaluation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
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

	engine    uint32
	namespace string

	url            string
	authentication any
	ref            string
	updateInterval time.Duration
	fetchMode      FetchMode
	errorStrategy  ErrorStrategy

	httpClient  *http.Client
	etag        string
	stopPolling chan struct{}
}

// clientOption adds additional configuration for Client parameters
type clientOption func(*EvaluationClient)

// WithNamespace allows for specifying which namespace the clients wants to make evaluations from.
func WithNamespace(namespace string) clientOption {
	return func(c *EvaluationClient) {
		c.namespace = namespace
	}
}

// WithURL allows for configuring the URL of an upstream Flipt instance to fetch feature data.
func WithURL(url string) clientOption {
	return func(c *EvaluationClient) {
		c.url = url
	}
}

// WithRef allows for specifying a reference to fetch feature data from.
func WithRef(ref string) clientOption {
	return func(c *EvaluationClient) {
		c.ref = ref
	}
}

// WithUpdateInterval allows for specifying how often flag state data should be fetched from an upstream Flipt instance.
func WithUpdateInterval(updateInterval time.Duration) clientOption {
	return func(c *EvaluationClient) {
		c.updateInterval = updateInterval
	}
}

// WithClientTokenAuthentication allows authenticating with Flipt using a static client token.
func WithClientTokenAuthentication(token string) clientOption {
	return func(c *EvaluationClient) {
		c.authentication = clientTokenAuthentication{
			Token: token,
		}
	}
}

// WithJWTAuthentication allows authenticating with Flipt using a JSON Web Token.
func WithJWTAuthentication(token string) clientOption {
	return func(c *EvaluationClient) {
		c.authentication = jwtAuthentication{
			Token: token,
		}
	}
}

// WithFetchMode allows for specifying the fetch mode for the Flipt client (e.g. polling, streaming).
// Note: Streaming is currently only supported when using the SDK with Flipt Cloud (https://flipt.io/cloud).
func WithFetchMode(fetchMode FetchMode) clientOption {
	return func(c *EvaluationClient) {
		c.fetchMode = fetchMode
	}
}

// WithErrorStrategy allows for specifying the error strategy for the Flipt client when fetching flag state (e.g. fail, fallback).
func WithErrorStrategy(errorStrategy ErrorStrategy) clientOption {
	return func(c *EvaluationClient) {
		c.errorStrategy = errorStrategy
	}
}

// NewEvaluationClient constructs a Client.
func NewEvaluationClient(ctx context.Context, opts ...clientOption) (*EvaluationClient, error) {
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

	client := &EvaluationClient{
		runtime:        runtime,
		mod:            mod,
		namespace:      "default",
		url:            "http://localhost:8080",
		httpClient:     &http.Client{},
		updateInterval: 2 * time.Minute, // default 120 seconds
		stopPolling:    make(chan struct{}),
	}

	for _, opt := range opts {
		opt(client)
	}

	if client.namespace == "" {
		mod.Close(ctx)
		runtime.Close(ctx)
		return nil, fmt.Errorf("namespace cannot be empty")
	}

	var (
		alloc            = mod.ExportedFunction("allocate")
		dealloc          = mod.ExportedFunction("deallocate")
		initializeEngine = mod.ExportedFunction("initialize_engine")
	)

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
				dealloc.Call(ctx, a.ptr, a.size)
			}
			client.Close(ctx)
		}
	}()

	// allocate namespace
	nsPtr, err := alloc.Call(ctx, uint64(len(client.namespace)))
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
	payload, err := client.fetchSnapshot(ctx)
	if err != nil {
		cleanup = true
		return nil, fmt.Errorf("failed to fetch initial state: %w", err)
	}

	// allocate payload
	pmPtr, err := alloc.Call(ctx, uint64(len(payload)))
	if err != nil {
		cleanup = true
		return nil, fmt.Errorf("failed to allocate memory for payload: %w", err)
	}
	allocations = append(allocations, allocation{pmPtr[0], uint64(len(payload))})

	if !mod.Memory().Write(uint32(pmPtr[0]), []byte(payload)) {
		cleanup = true
		return nil, fmt.Errorf("failed to write payload to memory")
	}

	// initialize engine
	res, err := initializeEngine.Call(ctx, nsPtr[0], uint64(len(client.namespace)), pmPtr[0], uint64(len(payload)))
	if err != nil {
		cleanup = true
		return nil, fmt.Errorf("failed to initialize engine: %w", err)
	}

	enginePtr := res[0]
	client.engine = uint32(enginePtr)

	// start background polling if interval > 0
	if client.updateInterval > 0 {
		go client.startPolling(ctx)
	}

	return client, nil
}

func (e *EvaluationClient) fetchSnapshot(ctx context.Context) (string, error) {
	url := fmt.Sprintf("%s/internal/v1/evaluation/snapshot/namespace/%s", e.url, e.namespace)
	if e.ref != "" {
		url = fmt.Sprintf("%s?reference=%s", url, e.ref)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
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

	if e.etag != "" {
		req.Header.Set("If-None-Match", e.etag)
	}

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch snapshot: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		return "", nil
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// store etag if present
	if etag := resp.Header.Get("ETag"); etag != "" {
		e.etag = etag
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(body), nil
}

func (e *EvaluationClient) startPolling(ctx context.Context) error {
	ticker := time.NewTicker(e.updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-e.stopPolling:
			return nil
		case <-ticker.C:
			payload, err := e.fetchSnapshot(ctx)
			if err != nil {
				return fmt.Errorf("failed to fetch snapshot: %w", err)
			}

			// skip update if no changes (304 response)
			if payload == "" {
				continue
			}

			// update engine with new state
			var (
				alloc    = e.mod.ExportedFunction("allocate")
				dealloc  = e.mod.ExportedFunction("deallocate")
				snapshot = e.mod.ExportedFunction("snapshot") // TODO: implement in Rust
			)

			pmPtr, err := alloc.Call(ctx, uint64(len(payload)))
			if err != nil {
				return fmt.Errorf("failed to allocate memory for payload: %w", err)
			}

			if !e.mod.Memory().Write(uint32(pmPtr[0]), []byte(payload)) {
				dealloc.Call(ctx, uint64(pmPtr[0]), uint64(len(payload)))
				return fmt.Errorf("failed to write payload to memory")
			}

			_, err = snapshot.Call(ctx, uint64(e.engine), pmPtr[0], uint64(len(payload)))
			if err != nil {
				dealloc.Call(ctx, uint64(pmPtr[0]), uint64(len(payload)))
				return fmt.Errorf("failed to update engine: %w", err)
			}

			// dont defer in loop to avoid stack overflow
			dealloc.Call(ctx, uint64(pmPtr[0]), uint64(len(payload)))
		}
	}
}

// EvaluateVariant performs evaluation for a variant flag.
func (e *EvaluationClient) EvaluateVariant(ctx context.Context, flagKey, entityID string, evalContext map[string]string) (*VariantEvaluationResponse, error) {
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

	dealloc := e.mod.ExportedFunction("deallocate")
	ptr, len := decodePtr(res[0])
	defer dealloc.Call(ctx, uint64(ptr), uint64(len))

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
	close(e.stopPolling)

	if e.engine != 0 {
		dealloc := e.mod.ExportedFunction("destroy_engine")
		dealloc.Call(ctx, uint64(e.engine))
	}

	_ = e.mod.Close(ctx)
	return e.runtime.Close(ctx)
}

func decodePtr(ptr uint64) (uint32, uint32) {
	return uint32(ptr >> 32), uint32(ptr)
}

func (e *EvaluationClient) evaluateWASM(ctx context.Context, funcName string, request any) ([]byte, error) {
	if e.engine == 0 {
		return nil, errors.New("engine not initialized")
	}

	reqBytes, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var (
		alloc   = e.mod.ExportedFunction("allocate")
		dealloc = e.mod.ExportedFunction("deallocate")
	)

	reqPtr, err := alloc.Call(ctx, uint64(len(reqBytes)))
	if err != nil {
		return nil, fmt.Errorf("failed to allocate memory for request: %w", err)
	}
	defer dealloc.Call(ctx, reqPtr[0], uint64(len(reqBytes)))

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
	defer dealloc.Call(ctx, uint64(ptr), uint64(len))

	b, ok := e.mod.Memory().Read(ptr, len)
	if !ok {
		return nil, fmt.Errorf("failed to read result from memory")
	}

	return b, nil
}
