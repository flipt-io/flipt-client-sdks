package sdks

import (
	"context"
	"fmt"
	"os"

	"dagger.io/dagger"
)

type DartSDK struct {
	BaseSDK
}

func (s *DartSDK) Build(ctx context.Context, client *dagger.Client, hostDirectory *dagger.Directory, opts BuildOpts) error {
	container := client.Container().From("dart:stable").
		WithDirectory("/src", hostDirectory.Directory("flipt-client-dart"), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{".gitignore", ".dart_tool/"},
		}).
		WithDirectory("/src/lib/src/ffi", hostDirectory.Directory("tmp"), dagger.ContainerWithDirectoryOpts{
			Include: defaultInclude,
		}).
		WithWorkdir("/src").
		WithExec([]string{"dart", "pub", "get"})

	var err error

	if !opts.Push {
		_, err = container.WithExec([]string{"dart", "pub", "publish", "--dry-run"}).Sync(ctx)
		return err
	}

	// get oidc token for publishing to pub.dev
	// https://dart.dev/tools/pub/automated-publishing#publishing-packages-using-github-actions
	if os.Getenv("PUB_TOKEN") == "" {
		return fmt.Errorf("PUB_TOKEN is not set")
	}

	pubToken := client.SetSecret("pub-token", os.Getenv("PUB_TOKEN"))

	_, err = container.WithSecretVariable("PUB_TOKEN", pubToken).
		WithExec([]string{"dart", "pub", "token", "add", "https://pub.dev", "--env-var", "PUB_TOKEN"}).
		WithExec([]string{"dart", "pub", "publish", "--force"}).
		Sync(ctx)

	return err
}
