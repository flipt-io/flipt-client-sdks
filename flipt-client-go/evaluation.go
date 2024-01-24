package evaluation

/*
#cgo darwin,arm64 LDFLAGS: -L${SRCDIR}/ext/darwin_arm64/. -lfliptengine
#cgo linux,arm64 LDFLAGS: -L${SRCDIR}/ext/linux_arm64/. -lfliptengine
#cgo linux,amd64 LDFLAGS: -L${SRCDIR}/ext/linux_x86_64/. -lfliptengine
#include <string.h>
#include <stdlib.h>
#include "./ext/flipt_engine.h"
*/
import "C"
import (
	"context"
	"encoding/json"
	"unsafe"
)

// Client wraps the functionality of making variant and boolean evaluation of Flipt feature flags.
type Client struct {
	engine         unsafe.Pointer
	namespace      string
	url            string
	authentication any
	ref            string
	updateInterval int
}

// NewClient constructs a Client.
func NewClient(opts ...clientOption) (*Client, error) {
	client := &Client{
		namespace: "default",
	}

	for _, opt := range opts {
		opt(client)
	}

	engOpts := &EngineOpts[any]{
		URL:            client.url,
		UpdateInterval: client.updateInterval,
		Authentication: &client.authentication,
		Reference:      client.ref,
	}

	b, err := json.Marshal(engOpts)
	if err != nil {
		return nil, err
	}

	ns := []*C.char{C.CString(client.namespace)}

	// Free the memory of the C Strings that were created to initialize the engine.
	defer func() {
		for _, n := range ns {
			C.free(unsafe.Pointer(n))
		}
	}()

	nsPtr := (**C.char)(unsafe.Pointer(&ns[0]))

	eng := C.initialize_engine(nsPtr, C.CString(string(b)))

	client.engine = eng

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

// WithURL allows for configuring the URL of an upstream Flipt instance to fetch feature data.
func WithURL(url string) clientOption {
	return func(c *Client) {
		c.url = url
	}
}

// WithUpdateInterval allows for specifying how often flag state data should be fetched from an upstream Flipt instance.
func WithUpdateInterval(updateInterval int) clientOption {
	return func(c *Client) {
		c.updateInterval = updateInterval
	}
}

// WithClientTokenAuthentication allows authenticating with Flipt using a static client token.
func WithClientTokenAuthentication(token string) clientOption {
	return func(c *Client) {
		c.authentication = clientTokenAuthentication{
			Token: token,
		}
	}
}

// WithJWTAuthentication allows authenticating with Flipt using a JSON Web Token.
func WithJWTAuthentication(token string) clientOption {
	return func(c *Client) {
		c.authentication = jwtAuthentication{
			Token: token,
		}
	}
}

// EvaluateVariant makes an evaluation on a variant flag.
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

	variant := C.evaluate_variant(e.engine, C.CString(string(ereq)))
	defer C.free(unsafe.Pointer(variant))

	b := C.GoBytes(unsafe.Pointer(variant), (C.int)(C.strlen(variant)))

	var vr *VariantResult

	if err := json.Unmarshal(b, &vr); err != nil {
		return nil, err
	}

	return vr, nil
}

// EvaluateBoolean makes an evaluation on a boolean flag.
func (e *Client) EvaluateBoolean(_ context.Context, flagKey, entityID string, evalContext map[string]string) (*BooleanResult, error) {
	ereq, err := json.Marshal(evaluationRequest{
		NamespaceKey: e.namespace,
		FlagKey:      flagKey,
		EntityId:     entityID,
		Context:      evalContext,
	})
	if err != nil {
		return nil, err
	}

	boolean := C.evaluate_boolean(e.engine, C.CString(string(ereq)))
	defer C.free(unsafe.Pointer(boolean))

	b := C.GoBytes(unsafe.Pointer(boolean), (C.int)(C.strlen(boolean)))

	var br *BooleanResult

	if err := json.Unmarshal(b, &br); err != nil {
		return nil, err
	}

	return br, nil
}

// Close cleans up the allocated engine as it was initialized in the constructor.
func (e *Client) Close() error {
	// Destroy the engine to clean up allocated memory on dynamic library side.
	C.destroy_engine(e.engine)
	return nil
}
