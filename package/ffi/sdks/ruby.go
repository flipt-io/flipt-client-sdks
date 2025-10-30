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

func (s *RubySDK) Build(ctx context.Context, client *dagger.Client, container *dagger.Container, hostDirectory *dagger.Directory, tmpDirectory *dagger.Directory, opts BuildOpts) error {
	container = container.From("ruby:3.1-bookworm").
		WithWorkdir("/src").
		WithDirectory("/src", hostDirectory.Directory("flipt-client-ruby")).
		WithDirectory("/src/lib/ext", tmpDirectory, dagger.ContainerWithDirectoryOpts{
			Include: dynamicInclude,
		}).
		WithFile("/src/lib/ext/flipt_engine.h", hostDirectory.File("flipt-engine-ffi/include/flipt_engine.h")).
		WithExec(args("bundle install")).
		WithExec(args("bundle exec rake build"))

	var err error

	if !opts.Push {
		out, err := container.WithExec(args("apt-get update")).
			WithExec(args("apt-get install -y tree")).
			WithExec(args("tree /src")).
			Stdout(ctx)
		if err != nil {
			return err
		}
		fmt.Println(out)
		return nil
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
