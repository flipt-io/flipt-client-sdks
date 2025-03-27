package sdks

import (
	"context"

	"dagger.io/dagger"
)

type BrowserSDK struct {
	BaseSDK
}

func (s *BrowserSDK) Build(ctx context.Context, client *dagger.Client, container *dagger.Container, hostDirectory *dagger.Directory, opts BuildOpts) error {
	return buildWasmJS(ctx, client, container, hostDirectory, opts, "web", "flipt-client-browser")
}
