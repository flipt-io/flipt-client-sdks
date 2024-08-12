package evaluation

import (
	"context"
	"encoding/json"
	"fmt"

	_ "embed"

	"github.com/bytecodealliance/wasmtime-go/v23"
)

var (
	//go:embed ext/flipt_evaluation_wasi.wasm
	wasm []byte
)

// Client wraps the functionality of making variant and boolean evaluation of Flipt feature flags.
type Client struct {
	namespace string
	instance  *wasmtime.Instance
	store     *wasmtime.Store
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

	f := instance.GetFunc(store, "new")
	if f == nil {
		return nil, fmt.Errorf("new function not found")
	}

	if _, err := f.Call(store, "default", "{}"); err != nil {
		return nil, err
	}

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

	f := e.instance.GetFunc(e.store, "evaluate-variant")
	if f == nil {
		return nil, fmt.Errorf("evaluate-variant function not found")
	}

	result, err := f.Call(e.store, ereq)
	if err != nil {
		return nil, err
	}

	var vr *VariantResult

	if err := json.Unmarshal([]byte(result.(string)), &vr); err != nil {
		return nil, err
	}

	return vr, nil
}
