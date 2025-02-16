package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"maps"
	"os"
	"strings"

	"dagger.io/dagger"
	"golang.org/x/sync/errgroup"
)

var (
	architecture string
	platform     string
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
		"csharp":  csharpTests,
	}
	sema = make(chan struct{}, 5)
)

type integrationTestFn func(context.Context, *dagger.Container, *testCase) error

func init() {
	flag.StringVar(&sdks, "sdks", "", "comma separated list of which language(s) to run integration tests for")
}

func main() {
	flag.Parse()

	if err := run(); err != nil {
		log.Fatal(err)
	}
}

type testResult struct {
	sdk     string
	success bool
	err     error
}

func run() error {
	tests := make(map[string]integrationTestFn, len(sdkToFn))
	results := make([]testResult, 0, len(sdkToFn))
	resultsChan := make(chan testResult, len(sdkToFn))

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

	// Detect the architecture and platform of the host machine
	architecture = "x86_64"
	platform = "linux/amd64"
	daggerPlatform, err := client.DefaultPlatform(ctx)
	if err != nil {
		fmt.Println("Error detecting host platform from Dagger: ", err)
		fmt.Println("Defaulting to platform: linux/amd64, architecture: x86_64")
	}
	platform = string(daggerPlatform)
	if strings.Contains(platform, "arm64") || strings.Contains(platform, "aarch64") {
		architecture = "arm64"
		platform = "linux/arm64"
	}

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

	container := client.Container(dagger.ContainerOpts{
		Platform: dagger.Platform(platform),
	})

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

			err := fn(ctx, container, testCase)
			resultsChan <- testResult{
				sdk:     lang,
				success: err == nil,
				err:     err,
			}
			return err
		}))
	}

	go func() {
		for result := range resultsChan {
			results = append(results, result)
		}
	}()

	err = g.Wait()
	close(resultsChan)

	fmt.Println("\n=== Test Results Summary ===")
	fmt.Println("Successful tests:")
	for _, result := range results {
		if result.success {
			fmt.Printf("✅ %s\n", result.sdk)
		}
	}

	fmt.Println("\nFailed tests:")
	for _, result := range results {
		if !result.success {
			fmt.Printf("❌ %s: %v\n", result.sdk, result.err)
		}
	}
	fmt.Println("=========================")

	return err
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

// getFFITestContainer builds the shared object library for the Rust core, and the Flipt container for the client libraries to run
// their tests against.
func getFFITestContainer(_ context.Context, client *dagger.Client, hostDirectory *dagger.Directory) *dagger.Container {
	target := "x86_64-unknown-linux-musl"
	if architecture == "arm64" {
		target = "aarch64-unknown-linux-musl"
	}

	return client.Container(dagger.ContainerOpts{
		Platform: dagger.Platform(platform),
	}).From("rust:1.83.0-slim").
		WithExec([]string{"apt-get", "update"}).
		WithExec([]string{"apt-get", "install", "-y", "build-essential", "musl-tools"}).
		WithExec([]string{"rustup", "target", "add", "aarch64-unknown-linux-musl", "x86_64-unknown-linux-musl"}).
		WithWorkdir("/src").
		WithDirectory("/src/flipt-engine-ffi", hostDirectory.Directory("flipt-engine-ffi")).
		WithDirectory("/src/flipt-engine-wasm", hostDirectory.Directory("flipt-engine-wasm"), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{"pkg/", ".gitignore"},
		}).
		WithDirectory("/src/flipt-evaluation", hostDirectory.Directory("flipt-evaluation")).
		WithFile("/src/Cargo.toml", hostDirectory.File("Cargo.toml")).
		WithEnvVariable("RUSTFLAGS", "-C target-feature=-crt-static").
		WithExec([]string{"cargo", "build", "-p", "flipt-engine-ffi", "--release", "--target", target}).                  // Cross-compile for musl
		WithExec([]string{"mkdir", "-p", "/src/target/release"}).                                                         // Ensure directory exists
		WithExec([]string{"cp", fmt.Sprintf("/src/target/%s/release/libfliptengine.so", target), "/src/target/release/"}) // Copy to expected location
}

// getWasmTestContainer builds the wasm module for the Rust core, and the Flipt container for the client libraries to run
// their tests against.
func getWasmTestContainer(_ context.Context, client *dagger.Client, hostDirectory *dagger.Directory, target string) *dagger.Container {
	container := client.Container(dagger.ContainerOpts{
		Platform: dagger.Platform(platform),
	}).From("rust:1.83.0-bookworm").
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
func pythonTests(ctx context.Context, root *dagger.Container, t *testCase) error {
	_, err := root.From("python:3.9-bookworm").
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
func goTests(ctx context.Context, root *dagger.Container, t *testCase) error {
	_, err := root.From("golang:1.21.3-bookworm").
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
func nodeTests(ctx context.Context, root *dagger.Container, t *testCase) error {
	_, err := root.From("node:21.2-bookworm").
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
func rubyTests(ctx context.Context, root *dagger.Container, t *testCase) error {
	_, err := root.From("ruby:3.1-bookworm").
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
func javaTests(ctx context.Context, root *dagger.Container, t *testCase) error {
	path := "x86-64"

	if architecture == "arm64" {
		path = "aarch64"
	}

	_, err := root.From("gradle:8.5.0-jdk11").
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
func browserTests(ctx context.Context, root *dagger.Container, t *testCase) error {
	_, err := root.From("node:21.2-bookworm").
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
func reactTests(ctx context.Context, root *dagger.Container, t *testCase) error {
	_, err := root.From("node:21.2-bookworm").
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
func dartTests(ctx context.Context, root *dagger.Container, t *testCase) error {
	_, err := root.From("dart:stable").
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

// csharpTests runs the csharp integration test suite against a container running Flipt.
func csharpTests(ctx context.Context, root *dagger.Container, t *testCase) error {
	_, err := root.From("mcr.microsoft.com/dotnet/sdk:8.0").
		WithWorkdir("/src").
		WithDirectory("/src", t.dir.Directory("flipt-client-csharp"), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{".gitignore", "obj/", "bin/"},
		}).
		WithFile(fmt.Sprintf("src/FliptClient/ext/ffi/linux_%s/libfliptengine.so", architecture), t.test.File(libFile)).
		WithServiceBinding("flipt", t.flipt.WithExec(nil).AsService()).
		WithEnvVariable("FLIPT_URL", "http://flipt:8080").
		WithEnvVariable("FLIPT_AUTH_TOKEN", "secret").
		WithExec([]string{"dotnet", "restore"}).
		WithExec([]string{"dotnet", "build"}).
		WithExec([]string{"dotnet", "test"}).
		Sync(ctx)

	return err
}
