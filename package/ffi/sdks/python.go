package sdks

import (
	"context"
	"fmt"
	"os"

	"dagger.io/dagger"
)

type PythonSDK struct {
	BaseSDK
}

func (s *PythonSDK) Build(ctx context.Context, client *dagger.Client, hostDirectory *dagger.Directory, opts BuildOpts) error {
	container := client.Container().From("python:3.11-bookworm").
		WithExec([]string{"pip", "install", "poetry==1.7.0"}).
		WithDirectory("/src", hostDirectory.Directory("flipt-client-python")).
		WithDirectory("/src/ext", hostDirectory.Directory("tmp"), dagger.ContainerWithDirectoryOpts{
			Include: defaultInclude,
		}).
		WithFile("/src/ext/flipt_engine.h", hostDirectory.File("flipt-engine-ffi/include/flipt_engine.h")).
		WithWorkdir("/src").
		WithExec([]string{"poetry", "install", "--without=dev", "-v"}).
		WithExec([]string{"poetry", "build", "-v"})

	var err error

	if !opts.Push {
		_, err = container.Sync(ctx)
		return err
	}

	if os.Getenv("PYPI_API_KEY") == "" {
		return fmt.Errorf("PYPI_API_KEY is not set")
	}

	pypiAPIKeySecret := client.SetSecret("pypi-api-key", os.Getenv("PYPI_API_KEY"))

	_, err = container.WithSecretVariable("POETRY_PYPI_TOKEN_PYPI", pypiAPIKeySecret).
		WithExec([]string{"poetry", "publish", "-v"}).
		Sync(ctx)

	return err
}
