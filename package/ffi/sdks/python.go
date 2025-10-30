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

func (s *PythonSDK) Build(ctx context.Context, client *dagger.Client, container *dagger.Container, hostDirectory *dagger.Directory, tmpDirectory *dagger.Directory, opts BuildOpts) error {
	container = container.From("python:3.11-bookworm").
		WithExec(args("pip install poetry==1.7.0")).
		WithDirectory("/src", hostDirectory.Directory("flipt-client-python")).
		WithDirectory("/src/ext", tmpDirectory, dagger.ContainerWithDirectoryOpts{
			Include: dynamicInclude,
		}).
		WithFile("/src/ext/flipt_engine.h", hostDirectory.File("flipt-engine-ffi/include/flipt_engine.h")).
		WithWorkdir("/src").
		WithExec(args("poetry install --without=dev -v")).
		WithExec(args("poetry build -v"))

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

	if os.Getenv("PYPI_API_KEY") == "" {
		return fmt.Errorf("PYPI_API_KEY is not set")
	}

	pypiAPIKeySecret := client.SetSecret("pypi-api-key", os.Getenv("PYPI_API_KEY"))

	_, err = container.WithSecretVariable("POETRY_PYPI_TOKEN_PYPI", pypiAPIKeySecret).
		WithExec(args("poetry publish -v")).
		Sync(ctx)

	return err
}
