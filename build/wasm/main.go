package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"maps"
	"os"
	"runtime"
	"strings"

	"dagger.io/dagger"
	"golang.org/x/sync/errgroup"
)

var (
	sdks     string
	push     bool
	tag      string
	engine   bool
	sdksToFn = map[string]buildFn{
		"browser": browserBuild,
	}
	sema = make(chan struct{}, 5)
)

func init() {
	flag.BoolVar(&engine, "engine", false, "build the engine")
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
	var builds = make(map[string]buildFn, len(sdksToFn))

	maps.Copy(builds, sdksToFn)

	if engine {
		builds["wasm"] = wasmBuild
	} else if sdks != "" {
		l := strings.Split(sdks, ",")
		subset := make(map[string]buildFn, len(l))
		for _, sdk := range l {
			testFn, ok := sdksToFn[sdk]
			if !ok {
				return fmt.Errorf("sdk %s is not supported", sdk)
			}
			subset[sdk] = testFn
		}

		builds = subset
	} else {
		return fmt.Errorf("no builds specified")
	}

	ctx := context.Background()

	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stdout))
	if err != nil {
		return err
	}

	defer client.Close()

	dir := client.Host().Directory(".", dagger.HostDirectoryOpts{
		Exclude: []string{".github/", "build/", "test/", ".git/"},
	})

	var g errgroup.Group

	for _, fn := range builds {
		fn := fn
		g.Go(take(func() error {
			return fn(ctx, client, dir)
		}))
	}

	return g.Wait()
}

func take(fn func() error) func() error {
	return func() error {
		// insert into semaphore channel to maintain
		// a max concurrency
		sema <- struct{}{}
		defer func() { <-sema }()

		return fn()
	}
}

func wasmBuild(ctx context.Context, client *dagger.Client, hostDirectory *dagger.Directory) error {
	arch := "x86_64"
	if strings.Contains(runtime.GOARCH, "arm64") || strings.Contains(runtime.GOARCH, "aarch64") {
		arch = "arm64"
	}

	rust := client.Container().From("rust:1.74.0-bookworm").
		WithWorkdir("/src").
		WithDirectory("/src/flipt-engine-ffi", hostDirectory.Directory("flipt-engine-ffi")).
		WithDirectory("/src/flipt-engine-wasm", hostDirectory.Directory("flipt-engine-wasm")).
		WithDirectory("/src/flipt-evaluation", hostDirectory.Directory("flipt-evaluation")).
		WithFile("/src/Cargo.toml", hostDirectory.File("Cargo.toml"))

	if arch == "arm64" {
		rust = rust.WithExec([]string{"apt-get", "update"}).
			WithExec([]string{"apt-get", "-y", "install", "binaryen"})
	}

	rust = rust.
		WithExec([]string{"cargo", "install", "wasm-pack"}). // Install wasm-pack
		WithWorkdir("/src/flipt-engine-wasm").
		WithExec([]string{"wasm-pack", "build", "--scope", "flipt-io"}) // Build the wasm package

	var err error

	if !push {
		_, err = rust.Sync(ctx)
		return err
	}

	container := client.Container().From("node:21.2-bookworm").
		WithDirectory("/src", rust.Directory("/src/flipt-engine-wasm/pkg"), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{"./node_modules/"},
		}).
		WithWorkdir("/src").
		WithExec([]string{"npm", "version", tag}) // Update the version

	if os.Getenv("NPM_API_KEY") == "" {
		return fmt.Errorf("NPM_API_KEY is not set")
	}

	npmAPIKeySecret := client.SetSecret("npm-api-key", os.Getenv("NPM_API_KEY"))

	_, err = container.WithSecretVariable("NPM_TOKEN", npmAPIKeySecret).
		WithExec([]string{"npm", "config", "set", "--", "//registry.npmjs.org/:_authToken", "${NPM_TOKEN}"}).
		WithExec([]string{"npm", "publish", "--access", "public"}).
		Sync(ctx)

	return err
}

func browserBuild(ctx context.Context, client *dagger.Client, hostDirectory *dagger.Directory) error {
	container := client.Container().From("node:21.2-bookworm").
		WithDirectory("/src", hostDirectory.Directory("flipt-client-browser"), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{"./node_modules/"},
		}).
		WithWorkdir("/src").
		WithExec([]string{"npm", "install"}).
		WithExec([]string{"npm", "run", "build"}).
		WithExec([]string{"npm", "pack"})

	var err error

	if !push {
		_, err = container.Sync(ctx)
		return err
	}

	if os.Getenv("NPM_API_KEY") == "" {
		return fmt.Errorf("NPM_API_KEY is not set")
	}

	npmAPIKeySecret := client.SetSecret("npm-api-key", os.Getenv("NPM_API_KEY"))

	_, err = container.WithSecretVariable("NPM_TOKEN", npmAPIKeySecret).
		WithExec([]string{"npm", "config", "set", "--", "//registry.npmjs.org/:_authToken", "${NPM_TOKEN}"}).
		WithExec([]string{"npm", "publish", "--access", "public"}).
		Sync(ctx)

	return err
}
