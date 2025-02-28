package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"dagger.io/dagger"
)

var (
	sdks string
	push bool
	tag  string
)

func init() {
	flag.StringVar(&sdks, "sdks", "", "comma separated list of which sdks(s) to run builds for")
	flag.BoolVar(&push, "push", false, "push built artifacts to registry")
	flag.StringVar(&tag, "tag", "", "tag to use for release")
}

func main() {
	flag.Parse()

	if err := run(); err != nil {
		log.Fatal(err)
	}
}

type buildFn func(context.Context, *dagger.Client, *dagger.Directory) error

func run() error {
	var build buildFn

	switch sdks {
	case "browser":
		build = browserBuild
	case "node":
		build = nodeBuild
	default:
		return fmt.Errorf("no builds specified")
	}

	ctx := context.Background()

	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stdout))
	if err != nil {
		return err
	}

	defer client.Close()

	dir := client.Host().Directory(".", dagger.HostDirectoryOpts{
		Exclude: []string{".github/", "package/", "test/", ".git/"},
	})

	return build(ctx, client, dir)
}

func args(args string, a ...any) []string {
	if len(a) == 0 {
		return strings.Split(args, " ")
	}
	return strings.Split(fmt.Sprintf(args, a...), " ")
}

func buildWasm(ctx context.Context, client *dagger.Client, hostDirectory *dagger.Directory, target string, clientDir string) error {
	rust := client.Container().From("rust:1.83.0-bookworm").
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
		WithExec(args("cargo install wasm-pack wasm-opt")). // Install wasm-pack
		WithWorkdir("/src/flipt-engine-wasm-js").
		WithExec(args("wasm-pack build --target %s", target)). // Build the wasm package
		Sync(ctx)

	if err != nil {
		return err
	}

	container := client.Container().From("node:21.2-bookworm").
		WithDirectory("/src", hostDirectory.Directory(clientDir), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{".node_modules/", ".gitignore", "pkg/", "dist/"},
		}).
		WithDirectory("/src/dist", rust.Directory("/src/flipt-engine-wasm-js/pkg"), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{".node_modules/", ".gitignore", "package.json", "README.md", "LICENSE"},
		}).
		WithWorkdir("/src").
		WithExec(args("npm install")).  // Install dependencies
		WithExec(args("npm run build")) // Build the package

	if !push {
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

func nodeBuild(ctx context.Context, client *dagger.Client, hostDirectory *dagger.Directory) error {
	return buildWasm(ctx, client, hostDirectory, "nodejs", "flipt-client-node")
}

func browserBuild(ctx context.Context, client *dagger.Client, hostDirectory *dagger.Directory) error {
	return buildWasm(ctx, client, hostDirectory, "web", "flipt-client-browser")
}
