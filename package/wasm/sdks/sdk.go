package sdks

import (
	"context"
	"fmt"
	"os"
	"strings"

	"dagger.io/dagger"
)

type BuildOpts struct {
	Push bool
	Tag  string
}

type SDK interface {
	Build(ctx context.Context, client *dagger.Client, container *dagger.Container, hostDirectory *dagger.Directory, opts BuildOpts) error
}

type BaseSDK struct{}

func buildWasmJS(ctx context.Context, client *dagger.Client, container *dagger.Container, hostDirectory *dagger.Directory, opts BuildOpts, target string, clientDir string) error {
	rust := container.From("rust:1.83.0-bookworm").
		WithWorkdir("/src").
		WithDirectory("/src/flipt-engine-ffi", hostDirectory.Directory("flipt-engine-ffi")).
		WithDirectory("/src/flipt-engine-wasm", hostDirectory.Directory("flipt-engine-wasm"), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{"pkg/", ".gitignore"},
		}).
		WithDirectory("/src/flipt-engine-wasm-js", hostDirectory.Directory("flipt-engine-wasm-js"), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{"pkg/", ".gitignore"},
		}).
		WithDirectory("/src/flipt-evaluation", hostDirectory.Directory("flipt-evaluation")).
		WithFile("/src/Cargo.toml", hostDirectory.File("Cargo.toml"))

	var err error

	rust, err = rust.
		WithExec(args("cargo install wasm-pack")). // Install wasm-pack
		WithWorkdir("/src/flipt-engine-wasm-js").
		WithExec(args("wasm-pack build --target %s", target)). // Build the wasm package
		Sync(ctx)

	if err != nil {
		return err
	}

	container = container.From("node:21.2-bookworm").
		WithDirectory("/src", hostDirectory.Directory(clientDir), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{".node_modules/", ".gitignore", "pkg/", "dist/", "wasm/"},
		}).
		WithDirectory("/src/src/wasm", rust.Directory("/src/flipt-engine-wasm-js/pkg"), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{".node_modules/", ".gitignore", "package.json", "README.md", "LICENSE"},
		}).
		WithWorkdir("/src").
		WithExec(args("npm install")).  // Install dependencies
		WithExec(args("npm run build")) // Build the package

	if !opts.Push {
		out, err := container.WithExec(args("apt-get update")).
			WithExec(args("apt-get install -y tree")).
			WithExec(args("tree -I node_modules /src")).
			Stdout(ctx)
		if err != nil {
			return err
		}
		fmt.Println(out)
		return nil
	}

	if os.Getenv("NPM_API_KEY") == "" {
		return fmt.Errorf("NPM_API_KEY is not set")
	}

	npmAPIKeySecret := client.SetSecret("npm-api-key", os.Getenv("NPM_API_KEY"))

	_, err = container.WithSecretVariable("NPM_TOKEN", npmAPIKeySecret).
		WithExec(args("npm config set -- //registry.npmjs.org/:_authToken ${NPM_TOKEN}")).
		WithExec(args("npm publish --access public")).
		Sync(ctx)

	return err
}

func args(args string, a ...any) []string {
	if len(a) == 0 {
		return strings.Split(args, " ")
	}
	return strings.Split(fmt.Sprintf(args, a...), " ")
}
