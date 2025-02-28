package sdks

import (
	"context"

	"dagger.io/dagger"
)

type BrowserSDK struct {
	BaseSDK
}

func (s *BrowserSDK) Build(ctx context.Context, client *dagger.Client, hostDirectory *dagger.Directory, opts BuildOpts) error {
	return buildWasmJS(ctx, client, hostDirectory, opts, "web", "flipt-client-browser")
}
