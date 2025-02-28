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
		WithDirectory("/src/src/FliptClient/ext/ffi", hostDirectory.Directory("tmp"), dagger.ContainerWithDirectoryOpts{
			Include: defaultInclude,
		}).
		WithExec(args("dotnet restore")).
		WithExec(args("rm -rf **/bin **/obj")).
		WithExec(args("dotnet build -c Release")).
		WithExec(args("dotnet pack -c Release -o bin/Release"))

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

	if os.Getenv("NUGET_API_KEY") == "" {
		return fmt.Errorf("NUGET_API_KEY is not set")
	}

	nugetAPIKeySecret := client.SetSecret("nuget-api-key", os.Getenv("NUGET_API_KEY"))

	_, err = container.WithSecretVariable("NUGET_API_KEY", nugetAPIKeySecret).
		WithExec(args("sh -c 'dotnet nuget push bin/Release/*.nupkg --api-key $NUGET_API_KEY --source https://api.nuget.org/v3/index.json --skip-duplicate'")).
		Sync(ctx)

	return err
}
