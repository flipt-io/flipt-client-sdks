package sdks

import (
	"context"
	"fmt"
	"os"

	"dagger.io/dagger"
)

type RubySDK struct {
	BaseSDK
}

func (s *RubySDK) Build(ctx context.Context, client *dagger.Client, hostDirectory *dagger.Directory, opts BuildOpts) error {
	container := client.Container().From("ruby:3.1-bookworm").
		WithWorkdir("/src").
		WithDirectory("/src", hostDirectory.Directory("flipt-client-ruby")).
		WithDirectory("/src/lib/ext", hostDirectory.Directory("tmp/glibc"), dagger.ContainerWithDirectoryOpts{
			Include: defaultInclude,
		}).
		WithFile("/src/lib/ext/flipt_engine.h", hostDirectory.File("flipt-engine-ffi/include/flipt_engine.h")).
		WithExec([]string{"bundle", "install"}).
		WithExec([]string{"bundle", "exec", "rake", "build"})

	var err error

	if !opts.Push {
		_, err = container.Sync(ctx)
		return err
	}

	if os.Getenv("RUBYGEMS_API_KEY") == "" {
		return fmt.Errorf("RUBYGEMS_API_KEY is not set")
	}

	gemHostAPIKeySecret := client.SetSecret("rubygems-api-key", os.Getenv("RUBYGEMS_API_KEY"))

	_, err = container.
		WithSecretVariable("GEM_HOST_API_KEY", gemHostAPIKeySecret).
		WithExec([]string{"sh", "-c", "gem push pkg/flipt_client-*.gem"}).
		Sync(ctx)

	return err
}
