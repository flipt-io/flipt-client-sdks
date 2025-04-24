package sdks

import (
	"context"
	"fmt"
	"os"

	"dagger.io/dagger"
	"go.flipt.io/flipt/client-sdks/package/ffi/platform"
)

type DartSDK struct {
	BaseSDK
}

func (s *DartSDK) SupportedPlatforms() []platform.Platform {
	return []platform.Platform{
		platform.LinuxX86_64,
		platform.LinuxArm64,
		platform.DarwinX86_64,
		platform.DarwinArm64,
		platform.WindowsX86_64,
		platform.AndroidArm64,
		platform.IOSArm64,
	}
}

func (s *DartSDK) Build(ctx context.Context, client *dagger.Client, container *dagger.Container, hostDirectory *dagger.Directory, opts BuildOpts) error {
	container = container.From("dart:stable").
		WithDirectory("/src", hostDirectory.Directory("flipt-client-dart"), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{".gitignore", ".dart_tool/"},
		}).
		WithDirectory("/src/lib/src/ffi", hostDirectory.Directory("tmp"), dagger.ContainerWithDirectoryOpts{
			Include: dynamicInclude,
		}).
		WithWorkdir("/src").
		WithExec(args("dart pub get"))

	var err error

	if !opts.Push {
		out, err := container.WithExec(args("dart pub publish --dry-run")).
			WithExec(args("apt-get update")).
			WithExec(args("apt-get install -y tree")).
			WithExec(args("tree /src")).
			Stdout(ctx)
		if err != nil {
			return err
		}
		fmt.Println(out)
		return nil
	}

	// get oidc token for publishing to pub.dev
	// https://dart.dev/tools/pub/automated-publishing#publishing-packages-using-github-actions
	if os.Getenv("PUB_TOKEN") == "" {
		return fmt.Errorf("PUB_TOKEN is not set")
	}

	pubToken := client.SetSecret("pub-token", os.Getenv("PUB_TOKEN"))

	_, err = container.WithSecretVariable("PUB_TOKEN", pubToken).
		WithExec(args("dart pub token add https://pub.dev --env-var PUB_TOKEN")).
		WithExec(args("dart pub publish --force")).
		Sync(ctx)

	return err
}
