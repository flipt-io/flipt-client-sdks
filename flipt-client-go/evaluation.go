package evaluation

/*
#cgo CFLAGS: -I./ext
#cgo darwin,arm64 LDFLAGS: -L${SRCDIR}/ext/darwin_arm64 -lfliptengine  -Wl,-rpath,${SRCDIR}/ext/darwin_arm64
#cgo darwin,amd64 LDFLAGS: -L${SRCDIR}/ext/darwin_x86_64 -lfliptengine -Wl,-rpath,${SRCDIR}/ext/darwin_x86_64
#cgo linux,arm64 LDFLAGS: -L${SRCDIR}/ext/linux_arm64 -lfliptengine  -Wl,-rpath,${SRCDIR}/ext/linux_arm64
#cgo linux,amd64 LDFLAGS: -L${SRCDIR}/ext/linux_x86_64 -lfliptengine -Wl,-rpath,${SRCDIR}/ext/linux_x86_64
#include <string.h>
#include <stdlib.h>
#include "flipt_engine.h"
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

	cn := C.CString(client.namespace)
	defer C.free(unsafe.Pointer(cn))
	co := C.CString(string(b))
	defer C.free(unsafe.Pointer(co))

	eng := C.initialize_engine(cn, co)
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

	cr := C.CString(string(ereq))
	defer C.free(unsafe.Pointer(cr))

	variant := C.evaluate_variant(e.engine, cr)
	defer C.destroy_string(variant)

	b := C.GoBytes(unsafe.Pointer(variant), (C.int)(C.strlen(variant)))

	var vr *VariantResult

	if err := json.Unmarshal(b, &vr); err != nil {
		return nil, err
	}

	return vr, nil
}

// EvaluateBoolean performs evaluation for a boolean flag.
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

	cr := C.CString(string(ereq))
	defer C.free(unsafe.Pointer(cr))

	boolean := C.evaluate_boolean(e.engine, cr)
	defer C.destroy_string(boolean)

	b := C.GoBytes(unsafe.Pointer(boolean), (C.int)(C.strlen(boolean)))

	var br *BooleanResult

	if err := json.Unmarshal(b, &br); err != nil {
		return nil, err
	}

	return br, nil
}

// EvaluateBatch performs evaluation for a batch of flags.
func (e *Client) EvaluateBatch(_ context.Context, requests []*EvaluationRequest) (*BatchResult, error) {
	evaluationRequests := make([]*evaluationRequest, 0, len(requests))

	for _, ir := range requests {
		evaluationRequests = append(evaluationRequests, &evaluationRequest{
			NamespaceKey: e.namespace,
			FlagKey:      ir.FlagKey,
			EntityId:     ir.EntityId,
			Context:      ir.Context,
		})
	}

	requestsBytes, err := json.Marshal(evaluationRequests)
	if err != nil {
		return nil, err
	}

	cr := C.CString(string(requestsBytes))
	defer C.free(unsafe.Pointer(cr))

	batch := C.evaluate_batch(e.engine, cr)
	defer C.destroy_string(batch)

	b := C.GoBytes(unsafe.Pointer(batch), (C.int)(C.strlen(batch)))

	var br *BatchResult

	if err := json.Unmarshal(b, &br); err != nil {
		return nil, err
	}

	return br, nil
}

func (e *Client) ListFlags(_ context.Context) (*ListFlagsResult, error) {
	flags := C.list_flags(e.engine)
	defer C.destroy_string(flags)

	b := C.GoBytes(unsafe.Pointer(flags), (C.int)(C.strlen(flags)))

	var fl *ListFlagsResult

	if err := json.Unmarshal(b, &fl); err != nil {
		return nil, err
	}

	return fl, nil
}

// Close cleans up the allocated engine as it was initialized in the constructor.
func (e *Client) Close() error {
	// Destroy the engine to clean up allocated memory on dynamic library side.
	C.destroy_engine(e.engine)
	return nil
}
