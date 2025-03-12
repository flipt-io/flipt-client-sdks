package sdks

import (
	"context"

	"dagger.io/dagger"
)

type NodeSDK struct {
	BaseSDK
}

func (s *NodeSDK) Build(ctx context.Context, client *dagger.Client, container *dagger.Container, hostDirectory *dagger.Directory, opts BuildOpts) error {
	return buildWasmJS(ctx, client, container, hostDirectory, opts, "nodejs", "flipt-client-node")
}
