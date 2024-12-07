package sdks

import (
	"context"
	"fmt"
	"os"

	"dagger.io/dagger"
)

type CSharpSDK struct {
	BaseSDK
}

func (s *CSharpSDK) Build(ctx context.Context, client *dagger.Client, hostDirectory *dagger.Directory, opts BuildOpts) error {
	container := client.Container().From("mcr.microsoft.com/dotnet/sdk:8.0").
		WithWorkdir("/src").
		WithDirectory("/src", hostDirectory.Directory("flipt-client-csharp"), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{".gitignore", "obj/", "bin/"},
		}).
		WithDirectory("/src/src/FliptClient/ext/ffi", hostDirectory.Directory("tmp/glibc"), dagger.ContainerWithDirectoryOpts{
			Include: defaultInclude,
		}).
		WithExec([]string{"dotnet", "build", "-c", "Release"}).
		WithExec([]string{"dotnet", "pack", "-c", "Release", "-o", "bin/Release"})

	var err error

	if !opts.Push {
		_, err = container.Sync(ctx)
		return err
	}

	if os.Getenv("NUGET_API_KEY") == "" {
		return fmt.Errorf("NUGET_API_KEY is not set")
	}

	nugetAPIKeySecret := client.SetSecret("nuget-api-key", os.Getenv("NUGET_API_KEY"))

	_, err = container.WithSecretVariable("NUGET_API_KEY", nugetAPIKeySecret).
		WithExec([]string{"sh", "-c", "dotnet nuget push bin/Release/*.nupkg --api-key $NUGET_API_KEY --source https://api.nuget.org/v3/index.json --skip-duplicate"}).
		Sync(ctx)

	return err
}
