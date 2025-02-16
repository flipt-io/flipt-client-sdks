package evaluation

/*
#cgo CFLAGS: -I./ext
#cgo darwin,arm64 LDFLAGS: -L${SRCDIR}/ext/darwin_arm64 -lfliptengine -Wl,-rpath,${SRCDIR}/ext/darwin_arm64
#cgo darwin,amd64 LDFLAGS: -L${SRCDIR}/ext/darwin_x86_64 -lfliptengine -Wl,-rpath,${SRCDIR}/ext/darwin_x86_64
#cgo linux,arm64 LDFLAGS: -L${SRCDIR}/ext/linux_arm64 -lfliptengine -lm -Wl,-rpath,${SRCDIR}/ext/linux_arm64
#cgo linux,amd64 LDFLAGS: -L${SRCDIR}/ext/linux_x86_64 -lfliptengine -lm -Wl,-rpath,${SRCDIR}/ext/linux_x86_64
#cgo windows,amd64 LDFLAGS: -L${SRCDIR}/ext/windows_x86_64 -lfliptengine -Wl,-rpath,${SRCDIR}/ext/windows_x86_64
#include <string.h>
#include <stdlib.h>
#include "flipt_engine.h"
*/
import "C"
import (
	"context"
	"encoding/json"
	"errors"
	"unsafe"
)

const statusSuccess = "success"

// EvaluationClient wraps the functionality of making variant and boolean evaluation of Flipt feature flags.
type EvaluationClient struct {
	engine         unsafe.Pointer
	namespace      string
	url            string
	authentication any
	ref            string
	updateInterval int
	fetchMode      FetchMode
	errorStrategy  ErrorStrategy
}

// NewEvaluationClient constructs a Client.
func NewEvaluationClient(opts ...clientOption) (*EvaluationClient, error) {
	client := &EvaluationClient{
		namespace: "default",
	}

	for _, opt := range opts {
		opt(client)
	}

	clientOpts := &clientOptions[any]{
		URL:            client.url,
		UpdateInterval: client.updateInterval,
		Authentication: &client.authentication,
		Reference:      client.ref,
		FetchMode:      client.fetchMode,
		ErrorStrategy:  client.errorStrategy,
	}

	b, err := json.Marshal(clientOpts)
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

// EvaluateVariant performs evaluation for a variant flag.
func (e *EvaluationClient) EvaluateVariant(_ context.Context, flagKey, entityID string, evalContext map[string]string) (*VariantEvaluationResponse, error) {
	ereq, err := json.Marshal(EvaluationRequest{
		FlagKey:  flagKey,
		EntityId: entityID,
		Context:  evalContext,
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

	if vr.Status != statusSuccess {
		return nil, errors.New(vr.ErrorMessage)
	}

	return vr.Result, nil
}

// EvaluateBoolean performs evaluation for a boolean flag.
func (e *EvaluationClient) EvaluateBoolean(_ context.Context, flagKey, entityID string, evalContext map[string]string) (*BooleanEvaluationResponse, error) {
	ereq, err := json.Marshal(EvaluationRequest{
		FlagKey:  flagKey,
		EntityId: entityID,
		Context:  evalContext,
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

	if br.Status != statusSuccess {
		return nil, errors.New(br.ErrorMessage)
	}
	return br.Result, nil

}

// EvaluateBatch performs evaluation for a batch of flags.
func (e *EvaluationClient) EvaluateBatch(_ context.Context, requests []*EvaluationRequest) (*BatchEvaluationResponse, error) {
	requestsBytes, err := json.Marshal(requests)
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

	if br.Status != statusSuccess {
		return nil, errors.New(br.ErrorMessage)
	}

	return br.Result, nil
}

func (e *EvaluationClient) ListFlags(_ context.Context) ([]Flag, error) {
	flags := C.list_flags(e.engine)
	defer C.destroy_string(flags)

	b := C.GoBytes(unsafe.Pointer(flags), (C.int)(C.strlen(flags)))

	var fl *ListFlagsResult

	if err := json.Unmarshal(b, &fl); err != nil {
		return nil, err
	}

	if fl.Status != statusSuccess {
		return nil, errors.New(fl.ErrorMessage)
	}

	return *fl.Result, nil
}

// Close cleans up the allocated engine as it was initialized in the constructor.
func (e *EvaluationClient) Close() error {
	// Destroy the engine to clean up allocated memory on dynamic library side.
	C.destroy_engine(e.engine)
	return nil
}
