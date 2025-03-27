package sdks

import (
	"context"

	"dagger.io/dagger"
)

type WebSDK struct {
	BaseSDK
}

func (s *WebSDK) Build(ctx context.Context, client *dagger.Client, container *dagger.Container, hostDirectory *dagger.Directory, opts BuildOpts) error {
	return buildWasmJS(ctx, client, container, hostDirectory, opts, "web", "flipt-client-web")
}
