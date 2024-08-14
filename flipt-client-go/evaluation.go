package evaluation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	_ "embed"

	"github.com/bytecodealliance/wasmtime-go/v23"
)

var (
	//go:embed ext/flipt_evaluation_wasm.wasm
	wasm []byte
)

// Client wraps the functionality of making variant and boolean evaluation of Flipt feature flags.
type Client struct {
	namespace string
	instance  *wasmtime.Instance
	store     *wasmtime.Store
	memory    *wasmtime.Memory
	enginePtr int32
}

// NewClient constructs a Client.
func NewClient(opts ...clientOption) (*Client, error) {
	client := &Client{
		namespace: "default",
	}

	for _, opt := range opts {
		opt(client)
	}

	engine := wasmtime.NewEngine()
	module, err := wasmtime.NewModule(engine, wasm)
	if err != nil {
		return nil, err
	}

	linker := wasmtime.NewLinker(engine)
	if err := linker.DefineWasi(); err != nil {
		return nil, err
	}

	wasiConfig := wasmtime.NewWasiConfig()
	wasiConfig.InheritStdout()
	wasiConfig.InheritStderr()

	store := wasmtime.NewStore(engine)
	store.SetWasi(wasiConfig)
	instance, err := linker.Instantiate(store, module)
	if err != nil {
		return nil, err
	}

	f := instance.GetFunc(store, "initialize_engine")
	if f == nil {
		return nil, fmt.Errorf("initialize_engine function not found")
	}

	namespace := "default"
	lenNamespace := len(namespace)

	payload := "{\"namespace\": {\"key\": \"default\"}, \"flags\": []}"
	lenPayload := len(payload)

	// allocate memory for the strings
	allocate := instance.GetFunc(store, "allocate")
	if allocate == nil {
		return nil, fmt.Errorf("allocate function not found")
	}

	namespacePtr, err := allocate.Call(store, lenNamespace)
	if err != nil {
		return nil, err
	}
	payloadPtr, err := allocate.Call(store, lenPayload)
	if err != nil {
		return nil, err
	}

	// get a pointer to the memory
	memoryInstance := instance.GetExport(store, "memory").Memory()
	if memoryInstance == nil {
		return nil, fmt.Errorf("memory not found in WASM instance")
	}

	// need to keep this around for the lifetime of the client
	// so it doesn't get garbage collected
	client.memory = memoryInstance

	// write namespace and payload to memory
	data := memoryInstance.UnsafeData(store)
	copy(data[uint32(namespacePtr.(int32)):], []byte(namespace))
	copy(data[uint32(payloadPtr.(int32)):], []byte(payload))

	// initialize_engine
	enginePtr, err := f.Call(store, namespacePtr, lenNamespace, payloadPtr, lenPayload)
	if err != nil {
		return nil, err
	}

	client.enginePtr = enginePtr.(int32)

	client.instance = instance
	client.store = store

	return client, nil
}

// clientOption adds additional configuration for Client parameters
type clientOption func(*Client)

// WithNamespace allows for specifying which namespace the clients wants to make evaluations from.
func WithNamespace(namespace string) clientOption {
	return func(c *Client) {
		c.namespace = namespace
	}
}

// // WithURL allows for configuring the URL of an upstream Flipt instance to fetch feature data.
// func WithURL(url string) clientOption {
// 	return func(c *Client) {
// 		c.url = url
// 	}
// }

// // WithUpdateInterval allows for specifying how often flag state data should be fetched from an upstream Flipt instance.
// func WithUpdateInterval(updateInterval int) clientOption {
// 	return func(c *Client) {
// 		c.updateInterval = updateInterval
// 	}
// }

// // WithClientTokenAuthentication allows authenticating with Flipt using a static client token.
// func WithClientTokenAuthentication(token string) clientOption {
// 	return func(c *Client) {
// 		c.authentication = clientTokenAuthentication{
// 			Token: token,
// 		}
// 	}
// }

// // WithJWTAuthentication allows authenticating with Flipt using a JSON Web Token.
// func WithJWTAuthentication(token string) clientOption {
// 	return func(c *Client) {
// 		c.authentication = jwtAuthentication{
// 			Token: token,
// 		}
// 	}
// }

// EvaluateVariant performs evaluation for a variant flag.
func (e *Client) EvaluateVariant(_ context.Context, flagKey, entityID string, evalContext map[string]string) (*VariantResult, error) {
	ereq, err := json.Marshal(evaluationRequest{
		NamespaceKey: e.namespace,
		FlagKey:      flagKey,
		EntityId:     entityID,
		Context:      evalContext,
	})
	if err != nil {
		return nil, err
	}

	lenEreq := len(ereq)
	// allocate memory for the strings
	allocate := e.instance.GetFunc(e.store, "allocate")
	if allocate == nil {
		return nil, fmt.Errorf("allocate function not found")
	}

	deallocate := e.instance.GetFunc(e.store, "deallocate")
	if deallocate == nil {
		return nil, fmt.Errorf("deallocate function not found")
	}

	ereqPtr, err := allocate.Call(e.store, lenEreq)
	if err != nil {
		return nil, err
	}

	defer deallocate.Call(e.store, ereqPtr, lenEreq)

	// copy the evaluation request to the memory
	data := e.memory.UnsafeData(e.store)
	copy(data[uint32(ereqPtr.(int32)):], ereq)

	f := e.instance.GetFunc(e.store, "evaluate_variant")
	if f == nil {
		return nil, fmt.Errorf("evaluate_variant function not found")
	}

	resultPtr, err := f.Call(e.store, e.enginePtr, ereqPtr, lenEreq)
	if err != nil {
		return nil, err
	}

	result := e.memory.UnsafeData(e.store)[uint32(resultPtr.(int32)):]
	n := bytes.IndexByte(result, 0)
	if n < 0 {
		n = 0
	}

	var vr *VariantResult
	if err := json.Unmarshal(result[:n], &vr); err != nil {
		return nil, err
	}

	return vr, nil
}
