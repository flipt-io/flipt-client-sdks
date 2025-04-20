package sdks

import (
	"context"

	"dagger.io/dagger"
)

type JsSDK struct {
	BaseSDK
}

func (s *JsSDK) Build(ctx context.Context, client *dagger.Client, container *dagger.Container, hostDirectory *dagger.Directory, opts BuildOpts) error {
	return buildWasmJS(ctx, client, container, hostDirectory, opts, "web", "flipt-client-js")
}
