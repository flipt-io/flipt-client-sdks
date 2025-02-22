package evaluation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

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

// EvaluationClient wraps the functionality of making variant and boolean evaluation of Flipt feature flags.
type EvaluationClient struct {
	mod api.Module

	engine    uint32
	namespace string

	url            string
	authentication any
	ref            string
	updateInterval int
	fetchMode      FetchMode
	errorStrategy  ErrorStrategy
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

// WithUpdateInterval allows for specifying how often flag state data should be fetched from an upstream Flipt instance.
func WithUpdateInterval(updateInterval int) clientOption {
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
		return nil, fmt.Errorf("failed to compile and load evaluation engine: %w", err)
	}

	cfg := wazero.NewModuleConfig().WithName("flipt_engine")
	mod, err := runtime.InstantiateModule(ctx, compiled, cfg)
	if err != nil {
		return nil, fmt.Errorf("can't instantiate Wasm module: %w", err)
	}

	client := &EvaluationClient{
		mod:       mod,
		namespace: "default",
	}

	for _, opt := range opts {
		opt(client)
	}

	if client.namespace == "" {
		return nil, fmt.Errorf("namespace cannot be empty")
	}

	alloc := mod.ExportedFunction("allocate")
	initializeEngine := mod.ExportedFunction("initialize_engine")

	nsPtr, err := alloc.Call(ctx, uint64(len(client.namespace)))
	if err != nil {
		return nil, fmt.Errorf("failed to allocate memory for namespace: %w", err)
	}

	if !mod.Memory().Write(uint32(nsPtr[0]), []byte(client.namespace)) {
		return nil, fmt.Errorf("failed to write namespace to memory")
	}

	// TODO: implement fetching
	payload := "{\"namespace\": {\"key\": \"default\"}, \"flags\": []}"
	pmPtr, err := alloc.Call(ctx, uint64(len(payload)))
	if err != nil {
		return nil, fmt.Errorf("failed to allocate memory for payload: %w", err)
	}

	if !mod.Memory().Write(uint32(pmPtr[0]), []byte(payload)) {
		return nil, fmt.Errorf("failed to write payload to memory")
	}

	res, err := initializeEngine.Call(ctx, nsPtr[0], uint64(len(client.namespace)), pmPtr[0], uint64(len(payload)))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize engine: %w", err)
	}
	enginePtr := res[0]

	// keep enginePtr in memory so it does not get garbage collected
	client.engine = uint32(enginePtr)

	return client, nil
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
		return nil, err
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
		return nil, err
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
		return nil, err
	}

	if batchResult.Status != statusSuccess {
		return nil, errors.New(batchResult.ErrorMessage)
	}

	return batchResult.Result, nil
}

// Common WASM evaluation logic
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

// func (e *EvaluationClient) ListFlags(_ context.Context) ([]Flag, error) {
// 	if e.engine == nil {
// 		return nil, errors.New("engine not initialized")
// 	}

// 	flags := C.list_flags(e.engine)
// 	defer C.destroy_string(flags)

// 	b := C.GoBytes(unsafe.Pointer(flags), (C.int)(C.strlen(flags)))

// 	var fl *ListFlagsResult

// 	if err := json.Unmarshal(b, &fl); err != nil {
// 		return nil, err
// 	}

// 	if fl.Status != statusSuccess {
// 		return nil, errors.New(fl.ErrorMessage)
// 	}

// 	return *fl.Result, nil
// }

// Close cleans up the allocated engine as it was initialized in the constructor.
func (e *EvaluationClient) Close(ctx context.Context) error {
	if e.engine != 0 {
		dealloc := e.mod.ExportedFunction("destroy_engine")
		dealloc.Call(ctx, uint64(e.engine))
	}
	return e.mod.Close(ctx)
}

func decodePtr(ptr uint64) (uint32, uint32) {
	return uint32(ptr >> 32), uint32(ptr)
}
