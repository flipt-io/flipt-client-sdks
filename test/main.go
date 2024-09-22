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
	architecture string
	sdks         string
	sdkToFn      = map[string]integrationTestFn{
		"python":  pythonTests,
		"go":      goTests,
		"node":    nodeTests,
		"ruby":    rubyTests,
		"java":    javaTests,
		"browser": browserTests,
		"dart":    dartTests,
		"react":   reactTests,
	}
	sema = make(chan struct{}, 5)
)

type integrationTestFn func(context.Context, *dagger.Client, *testCase) error

func init() {
	flag.StringVar(&sdks, "sdks", "", "comma separated list of which language(s) to run integration tests for")
}

func main() {
	flag.Parse()

	architecture = "x86_64"
	if strings.Contains(runtime.GOARCH, "arm64") || strings.Contains(runtime.GOARCH, "aarch64") {
		architecture = "arm64"
	}

	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	var tests = make(map[string]integrationTestFn, len(sdkToFn))

	maps.Copy(tests, sdkToFn)

	if sdks != "" {
		l := strings.Split(sdks, ",")
		subset := make(map[string]integrationTestFn, len(l))
		for _, language := range l {
			testFn, ok := sdkToFn[language]
			if !ok {
				return fmt.Errorf("sdk %s is not supported", language)
			}
			subset[language] = testFn
		}

		tests = subset
	}

	ctx := context.Background()

	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stdout))
	if err != nil {
		return err
	}
	defer client.Close()

	dir := client.Host().Directory(".", dagger.HostDirectoryOpts{
		Exclude: []string{".github/", "build/", "tmp/", ".git/"},
	})

	flipt := client.Container().From("flipt/flipt:latest").
		WithUser("root").
		WithExec([]string{"mkdir", "-p", "/var/data/flipt"}).
		WithDirectory("/var/data/flipt", dir.Directory("test/fixtures/testdata")).
		WithExec([]string{"chown", "-R", "flipt:flipt", "/var/data/flipt"}).
		WithUser("flipt").
		WithEnvVariable("FLIPT_STORAGE_TYPE", "local").
		WithEnvVariable("FLIPT_STORAGE_LOCAL_PATH", "/var/data/flipt").
		WithEnvVariable("FLIPT_AUTHENTICATION_METHODS_TOKEN_ENABLED", "true").
		WithEnvVariable("FLIPT_AUTHENTICATION_METHODS_TOKEN_BOOTSTRAP_TOKEN", "secret").
		WithEnvVariable("FLIPT_AUTHENTICATION_REQUIRED", "true").
		WithExposedPort(8080)

	var g errgroup.Group

	for lang, fn := range tests {
		lang, fn := lang, fn
		g.Go(take(func() error {
			testCase := &testCase{
				sdk:   lang,
				dir:   dir,
				flipt: flipt,
			}

			switch lang {
			case "node":
				testCase.test = getWasmTestContainer(ctx, client, dir, "nodejs")
			case "browser", "react":
				testCase.test = getWasmTestContainer(ctx, client, dir, "web")
			default:
				testCase.test = getFFITestContainer(ctx, client, dir)
			}

			return fn(ctx, client, testCase)
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

const (
	libFile    = "/src/target/release/libfliptengine.so"
	headerFile = "/src/flipt-engine-ffi/include/flipt_engine.h"
	wasmDir    = "/src/flipt-engine-wasm/pkg"
)

type testCase struct {
	sdk   string
	dir   *dagger.Directory
	flipt *dagger.Container
	test  *dagger.Container
}

// getFFITestContainer builds the dynamic library for the Rust core, and the Flipt container for the client libraries to run
// their tests against.
func getFFITestContainer(_ context.Context, client *dagger.Client, hostDirectory *dagger.Directory) *dagger.Container {
	return client.Container().From("rust:1.74.0-bookworm").
		WithWorkdir("/src").
		WithDirectory("/src/flipt-engine-ffi", hostDirectory.Directory("flipt-engine-ffi")).
		WithDirectory("/src/flipt-engine-wasm", hostDirectory.Directory("flipt-engine-wasm"), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{"pkg/", ".gitignore"},
		}).
		WithDirectory("/src/flipt-evaluation", hostDirectory.Directory("flipt-evaluation")).
		WithFile("/src/Cargo.toml", hostDirectory.File("Cargo.toml")).
		WithExec([]string{"cargo", "build", "-p", "flipt-engine-ffi", "--release"}) // Build the dynamic library
}

// getWasmTestContainer builds the wasm module for the Rust core, and the Flipt container for the client libraries to run
// their tests against.
func getWasmTestContainer(_ context.Context, client *dagger.Client, hostDirectory *dagger.Directory, target string) *dagger.Container {
	container := client.Container().From("rust:1.74.0-bookworm").
		WithWorkdir("/src").
		WithDirectory("/src/flipt-engine-ffi", hostDirectory.Directory("flipt-engine-ffi")).
		WithDirectory("/src/flipt-engine-wasm", hostDirectory.Directory("flipt-engine-wasm"), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{"pkg/", ".gitignore"},
		}).
		WithDirectory("/src/flipt-evaluation", hostDirectory.Directory("flipt-evaluation")).
		WithFile("/src/Cargo.toml", hostDirectory.File("Cargo.toml")).
		WithExec([]string{"cargo", "build", "-p", "flipt-engine-wasm", "--release"}) // Build the wasm module

	if architecture == "arm64" {
		container = container.WithExec([]string{"apt-get", "update"}).
			WithExec([]string{"apt-get", "-y", "install", "binaryen"})
	}

	return container.WithExec([]string{"cargo", "install", "wasm-pack"}). // Install wasm-pack
										WithWorkdir("/src/flipt-engine-wasm").
										WithExec([]string{"wasm-pack", "build", "--target", target}) // Build the wasm package
}

// pythonTests runs the python integration test suite against a container running Flipt.
func pythonTests(ctx context.Context, client *dagger.Client, t *testCase) error {
	_, err := client.Pipeline("python").Container().From("python:3.11-bookworm").
		WithExec([]string{"pip", "install", "poetry==1.7.0"}).
		WithWorkdir("/src").
		WithDirectory("/src", t.dir.Directory("flipt-client-python")).
		WithFile(fmt.Sprintf("/src/ext/linux_%s/libfliptengine.so", architecture), t.test.File(libFile)).
		WithServiceBinding("flipt", t.flipt.WithExec(nil).AsService()).
		WithEnvVariable("FLIPT_URL", "http://flipt:8080").
		WithEnvVariable("FLIPT_AUTH_TOKEN", "secret").
		WithExec([]string{"poetry", "install", "--without=dev"}).
		WithExec([]string{"poetry", "run", "test"}).
		Sync(ctx)

	return err
}

// goTests runs the golang integration test suite against a container running Flipt.
func goTests(ctx context.Context, client *dagger.Client, t *testCase) error {
	_, err := client.Pipeline("go").Container().From("golang:1.21.3-bookworm").
		WithExec([]string{"apt-get", "update"}).
		WithExec([]string{"apt-get", "-y", "install", "build-essential"}).
		WithWorkdir("/src").
		WithDirectory("/src", t.dir.Directory("flipt-client-go")).
		WithFile(fmt.Sprintf("/src/ext/linux_%s/libfliptengine.so", architecture), t.test.File(libFile)).
		WithFile("/src/ext/flipt_engine.h", t.test.File(headerFile)).
		WithServiceBinding("flipt", t.flipt.WithExec(nil).AsService()).
		WithEnvVariable("FLIPT_URL", "http://flipt:8080").
		WithEnvVariable("FLIPT_AUTH_TOKEN", "secret").
		// Since the dynamic library is being sourced from a "non-standard location" we can
		// modify the LD_LIBRARY_PATH variable to inform the linker different locations for
		// dynamic libraries.
		WithEnvVariable("LD_LIBRARY_PATH", fmt.Sprintf("/src/ext/linux_%s:$LD_LIBRARY_PATH", architecture)).
		WithExec([]string{"go", "mod", "download"}).
		WithExec([]string{"go", "test", "./..."}).
		Sync(ctx)

	return err
}

// nodeTests runs the node integration test suite against a container running Flipt.
func nodeTests(ctx context.Context, client *dagger.Client, t *testCase) error {
	_, err := client.Pipeline("node").Container().From("node:21.2-bookworm").
		WithWorkdir("/src").
		WithDirectory("/src", t.dir.Directory("flipt-client-node"), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{".node_modules/", ".gitignore", "dist/"},
		}).
		WithDirectory("/src/dist", t.test.Directory(wasmDir), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{".node_modules/", ".gitignore", "package.json", "README.md", "LICENSE"},
		}).
		WithServiceBinding("flipt", t.flipt.WithExec(nil).AsService()).
		WithEnvVariable("FLIPT_URL", "http://flipt:8080").
		WithEnvVariable("FLIPT_AUTH_TOKEN", "secret").
		WithExec([]string{"npm", "install"}).
		WithExec([]string{"npm", "run", "build"}).
		WithExec([]string{"npm", "test"}).
		Sync(ctx)

	return err
}

// rubyTests runs the ruby integration test suite against a container running Flipt.
func rubyTests(ctx context.Context, client *dagger.Client, t *testCase) error {
	_, err := client.Pipeline("ruby").Container().From("ruby:3.1-bookworm").
		WithWorkdir("/src").
		WithDirectory("/src", t.dir.Directory("flipt-client-ruby")).
		WithFile(fmt.Sprintf("/src/lib/ext/linux_%s/libfliptengine.so", architecture), t.test.File(libFile)).
		WithServiceBinding("flipt", t.flipt.WithExec(nil).AsService()).
		WithEnvVariable("FLIPT_URL", "http://flipt:8080").
		WithEnvVariable("FLIPT_AUTH_TOKEN", "secret").
		WithExec([]string{"bundle", "install"}).
		WithExec([]string{"bundle", "exec", "rspec"}).
		Sync(ctx)

	return err
}

// javaTests run the java integration tests suite against a container running Flipt.
func javaTests(ctx context.Context, client *dagger.Client, t *testCase) error {
	path := "x86-64"

	if architecture == "arm64" {
		path = "aarch64"
	}

	_, err := client.Pipeline("java").Container().From("gradle:8.5.0-jdk11").
		WithWorkdir("/src").
		WithDirectory("/src", t.dir.Directory("flipt-client-java"), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{"./.idea/", ".gradle/", "build/"},
		}).
		WithFile(fmt.Sprintf("/src/src/main/resources/linux-%s/libfliptengine.so", path), t.test.File(libFile)).
		WithServiceBinding("flipt", t.flipt.WithExec(nil).AsService()).
		WithEnvVariable("FLIPT_URL", "http://flipt:8080").
		WithEnvVariable("FLIPT_AUTH_TOKEN", "secret").
		WithExec([]string{"gradle", "test"}).
		Sync(ctx)

	return err
}

// browserTests runs the browser integration test suite against a container running Flipt.
func browserTests(ctx context.Context, client *dagger.Client, t *testCase) error {
	_, err := client.Pipeline("browser").Container().From("node:21.2-bookworm").
		WithWorkdir("/src").
		WithDirectory("/src", t.dir.Directory("flipt-client-browser"), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{".node_modules/", ".gitignore", "dist/"},
		}).
		WithDirectory("/src/dist", t.test.Directory(wasmDir), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{".node_modules/", ".gitignore", "package.json", "README.md", "LICENSE"},
		}).
		WithServiceBinding("flipt", t.flipt.WithExec(nil).AsService()).
		WithEnvVariable("FLIPT_URL", "http://flipt:8080").
		WithEnvVariable("FLIPT_AUTH_TOKEN", "secret").
		WithExec([]string{"npm", "install"}).
		WithExec([]string{"npm", "run", "build"}).
		WithExec([]string{"npm", "test"}).
		Sync(ctx)

	return err
}

// reactTests runs the react unit test suite against a mocked Flipt client.
// this is because the react client simply uses the browser client under the hood
func reactTests(ctx context.Context, client *dagger.Client, t *testCase) error {
	_, err := client.Pipeline("react").Container().From("node:21.2-bookworm").
		WithWorkdir("/src").
		WithDirectory("/src", t.dir.Directory("flipt-client-react"), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{".node_modules/", ".gitignore", "dist/"},
		}).
		WithExec([]string{"npm", "install"}).
		WithExec([]string{"npm", "run", "build"}).
		WithExec([]string{"npm", "test"}).
		Sync(ctx)

	return err
}

// dartTests runs the dart integration test suite against a container running Flipt.
func dartTests(ctx context.Context, client *dagger.Client, t *testCase) error {
	_, err := client.Pipeline("dart").Container().From("dart:stable").
		WithWorkdir("/src").
		WithDirectory("/src", t.dir.Directory("flipt-client-dart"), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{".gitignore", ".dart_tool/"},
		}).
		WithFile(fmt.Sprintf("/src/lib/src/ffi/linux_%s/libfliptengine.so", architecture), t.test.File(libFile)).
		WithServiceBinding("flipt", t.flipt.WithExec(nil).AsService()).
		WithEnvVariable("FLIPT_URL", "http://flipt:8080").
		WithEnvVariable("FLIPT_AUTH_TOKEN", "secret").
		WithExec([]string{"dart", "pub", "get"}).
		WithExec([]string{"dart", "test"}).
		Sync(ctx)

	return err
}
