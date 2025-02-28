package sdks

import (
	"context"

	"dagger.io/dagger"
)

type NodeSDK struct {
	BaseSDK
}

func (s *NodeSDK) Build(ctx context.Context, client *dagger.Client, hostDirectory *dagger.Directory, opts BuildOpts) error {
	return buildWasmJS(ctx, client, hostDirectory, opts, "nodejs", "flipt-client-node")
}
