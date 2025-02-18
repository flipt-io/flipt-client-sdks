package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"maps"
	"os"
	"strings"
	"sync"

	"dagger.io/dagger"
	"golang.org/x/sync/errgroup"
)

var (
	architecture = "x86_64"
	platform     = "linux/amd64"
	sdks         string
	sema         = make(chan struct{}, 10)
)

// containerConfig holds the base configuration for a test container
type containerConfig struct {
	base     string
	commands []string
}

// testConfig holds the complete test configuration
type testConfig struct {
	containers []containerConfig
	fn         testFn
}

// testCase represents a single test execution context
type testCase struct {
	sdk     string
	hostDir *dagger.Directory
	flipt   *dagger.Container
	engine  *dagger.Container
}

type testFn func(context.Context, *dagger.Container, *testCase) error

// testResult holds the result of a test execution
type testResult struct {
	sdk  string
	base string
	err  error
}

const (
	headerFile = "/src/flipt-engine-ffi/include/flipt_engine.h"
	wasmDir    = "/src/flipt-engine-wasm/pkg"
)

// Define test configurations
var sdkToConfig = map[string]testConfig{
	"python": {containers: []containerConfig{{base: "python:3.9-bookworm"}}, fn: pythonTests},
	"go": {containers: []containerConfig{
		{
			base:     "golang:1.23-bookworm",
			commands: []string{"apt-get update", "apt-get install -y build-essential"},
		},
		{
			base:     "golang:1.23-bullseye",
			commands: []string{"apt-get update", "apt-get install -y build-essential"},
		},
		{
			base:     "golang:1.23-alpine",
			commands: []string{"apk update", "apk add --no-cache build-base"},
		},
	}, fn: goTests},
	"node":    {containers: []containerConfig{{base: "node:21.2-bookworm"}}, fn: nodeTests},
	"ruby":    {containers: []containerConfig{{base: "ruby:3.1-bookworm"}}, fn: rubyTests},
	"java":    {containers: []containerConfig{{base: "gradle:8.5.0-jdk11"}}, fn: javaTests},
	"browser": {containers: []containerConfig{{base: "node:21.2-bookworm"}}, fn: browserTests},
	"dart":    {containers: []containerConfig{{base: "dart:stable"}}, fn: dartTests},
	"react":   {containers: []containerConfig{{base: "node:21.2-bookworm"}}, fn: reactTests},
	"csharp":  {containers: []containerConfig{{base: "mcr.microsoft.com/dotnet/sdk:8.0"}}, fn: csharpTests},
}

func init() {
	flag.StringVar(&sdks, "sdks", "", "comma separated list of which language(s) to run integration tests for")
}

func main() {
	flag.Parse()

	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func args(args string) []string {
	return strings.Split(args, " ")
}

func run() error {
	var (
		tests       = make(map[string]testConfig, len(sdkToConfig))
		results     = []testResult{}
		resultsChan = make(chan testResult)
	)

	maps.Copy(tests, sdkToConfig)

	if sdks != "" {
		l := strings.Split(sdks, ",")
		subset := make(map[string]testConfig, len(l))
		for _, language := range l {
			testFn, ok := sdkToConfig[language]
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

	hostDir := client.Host().Directory(".", dagger.HostDirectoryOpts{
		Exclude: []string{".github/", "build/", "tmp/", ".git/"},
	})

	// Detect the architecture and platform of the host machine
	err = detectPlatform(client)
	if err != nil {
		return err
	}

	flipt := setupFliptContainer(client, hostDir)

	var g errgroup.Group

	container := client.Container(dagger.ContainerOpts{
		Platform: dagger.Platform(platform),
	})

	for lang, t := range tests {
		lang, t := lang, t
		for _, c := range t.containers {
			c := c

			g.Go(take(func() error {
				testCase := &testCase{
					sdk:     lang,
					hostDir: hostDir,
					flipt:   flipt,
				}

				switch lang {
				case "node":
					testCase.engine = getWasmTestContainer(ctx, client, hostDir, "nodejs")
				case "browser", "react":
					testCase.engine = getWasmTestContainer(ctx, client, hostDir, "web")
				default:
					testCase.engine = getFFITestContainer(ctx, client, hostDir)
				}

				container = createBaseContainer(client, c)

				err := t.fn(ctx, container, testCase)
				resultsChan <- testResult{
					sdk:  lang,
					base: c.base,
					err:  err,
				}
				return err
			}))
		}
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		for result := range resultsChan {
			results = append(results, result)
		}
	}()

	err = g.Wait()
	close(resultsChan)

	wg.Wait()

	fmt.Println("\n=== Test Results Summary ===")

	fmt.Println("\nSuccessful tests:")
	for _, result := range results {
		if result.err == nil {
			fmt.Printf("✅ %s %s\n", result.sdk, result.base)
		}
	}

	fmt.Println("\nFailed tests:")
	for _, result := range results {
		if result.err != nil {
			fmt.Printf("❌ %s %s: %v\n", result.sdk, result.base, result.err)
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

func detectPlatform(client *dagger.Client) error {
	daggerPlatform, err := client.DefaultPlatform(context.Background())
	if err != nil {
		fmt.Println("Error detecting host platform from Dagger: ", err)
		fmt.Println("Defaulting to platform: linux/amd64, architecture: x86_64")
		return nil
	}

	platform = string(daggerPlatform)
	if strings.Contains(platform, "arm64") || strings.Contains(platform, "aarch64") {
		platform = "linux/arm64"
		architecture = "aarch64"
	}
	return nil
}

func setupFliptContainer(client *dagger.Client, hostDir *dagger.Directory) *dagger.Container {
	return client.Container().From("flipt/flipt:latest").
		WithUser("root").
		WithExec([]string{"mkdir", "-p", "/var/data/flipt"}).
		WithDirectory("/var/data/flipt", hostDir.Directory("test/fixtures/testdata")).
		WithExec([]string{"chown", "-R", "flipt:flipt", "/var/data/flipt"}).
		WithUser("flipt").
		WithEnvVariable("FLIPT_STORAGE_TYPE", "local").
		WithEnvVariable("FLIPT_STORAGE_LOCAL_PATH", "/var/data/flipt").
		WithEnvVariable("FLIPT_AUTHENTICATION_METHODS_TOKEN_ENABLED", "true").
		WithEnvVariable("FLIPT_AUTHENTICATION_METHODS_TOKEN_BOOTSTRAP_TOKEN", "secret").
		WithEnvVariable("FLIPT_AUTHENTICATION_REQUIRED", "true").
		WithExposedPort(8080)
}

// Helper function to create a base container with common settings
func createBaseContainer(client *dagger.Client, config containerConfig) *dagger.Container {
	container := client.Container(dagger.ContainerOpts{
		Platform: dagger.Platform(platform),
	}).From(config.base)

	for _, cmd := range config.commands {
		container = container.WithExec(args(cmd))
	}
	return container
}

// getFFITestContainer builds the shared object library for the Rust core, and the Flipt container for the client libraries to run
// their tests against.
func getFFITestContainer(_ context.Context, client *dagger.Client, hostDirectory *dagger.Directory) *dagger.Container {
	target := fmt.Sprintf("%s-unknown-linux-musl", architecture)

	return client.Container(dagger.ContainerOpts{
		Platform: dagger.Platform(platform),
	}).From("rust:alpine3.17").
		WithExec([]string{"apk", "update"}).
		WithExec([]string{"apk", "add", "--no-cache", "build-base", "musl-dev", "binutils", "libgcc", "gcc"}).
		WithExec([]string{"rustup", "target", "add", target}).
		WithWorkdir("/src").
		WithDirectory("/src/flipt-engine-ffi", hostDirectory.Directory("flipt-engine-ffi")).
		WithDirectory("/src/flipt-engine-wasm", hostDirectory.Directory("flipt-engine-wasm"), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{"pkg/", ".gitignore"},
		}).
		WithDirectory("/src/flipt-evaluation", hostDirectory.Directory("flipt-evaluation")).
		WithFile("/src/Cargo.toml", hostDirectory.File("Cargo.toml")).
		WithEnvVariable("RUSTFLAGS", "-C target-feature=+crt-static").
		WithExec([]string{"cargo", "build", "-p", "flipt-engine-ffi", "--release", "--target", target}).
		// Copy the static lib to where the wrapper build script can find it
		WithExec([]string{"cp", fmt.Sprintf("/src/target/%s/release/libfliptengine.a", target), "/src/flipt-engine-ffi/"}).
		// Make and run the build script
		WithExec([]string{"chmod", "+x", "/src/flipt-engine-ffi/build.sh"}).
		WithExec([]string{"/src/flipt-engine-ffi/build.sh"}).
		// Copy both .so and .a files to output
		WithExec([]string{"mkdir", "-p", "/output"}).
		WithExec([]string{"cp", "/src/flipt-engine-ffi/libfliptengine.so", "/output/"}).
		WithExec([]string{"cp", "/src/flipt-engine-ffi/libfliptengine.a", "/output/libfliptengine_static.a"})
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

	if architecture == "aarch64" {
		container = container.WithExec([]string{"apt-get", "update"}).
			WithExec([]string{"apt-get", "-y", "install", "binaryen"})
	}

	return container.WithExec([]string{"cargo", "install", "wasm-pack"}). // Install wasm-pack
										WithWorkdir("/src/flipt-engine-wasm").
										WithExec([]string{"wasm-pack", "build", "--target", target}) // Build the wasm package
}

// pythonTests runs the python integration test suite against a container running Flipt.
func pythonTests(ctx context.Context, root *dagger.Container, t *testCase) error {
	// previously python:3.9-bookworm
	_, err := root.
		WithExec([]string{"pip", "install", "poetry==1.7.0"}).
		WithWorkdir("/src").
		WithDirectory("/src", t.hostDir.Directory("flipt-client-python")).
		WithDirectory(fmt.Sprintf("/src/ext/linux_%s", architecture), t.engine.Directory("/output")).
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
	// previously golang:1.21.3-bookworm
	_, err := root.
		WithWorkdir("/src").
		WithDirectory("/src", t.hostDir.Directory("flipt-client-go")).
		WithDirectory(fmt.Sprintf("/src/ext/linux_%s", architecture), t.engine.Directory("/output")).
		WithFile("/src/ext/flipt_engine.h", t.engine.File(headerFile)).
		WithServiceBinding("flipt", t.flipt.WithExec(nil).AsService()).
		WithEnvVariable("FLIPT_URL", "http://flipt:8080").
		WithEnvVariable("FLIPT_AUTH_TOKEN", "secret").
		WithExec([]string{"go", "mod", "download"}).
		WithExec([]string{"go", "test", "-v", "./..."}).
		Sync(ctx)

	return err
}

// nodeTests runs the node integration test suite against a container running Flipt.

func nodeTests(ctx context.Context, root *dagger.Container, t *testCase) error {
	// previously node:21.2-bookworm
	_, err := root.
		WithWorkdir("/src").
		WithDirectory("/src", t.hostDir.Directory("flipt-client-node"), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{".node_modules/", ".gitignore", "dist/"},
		}).
		WithDirectory("/src/dist", t.engine.Directory(wasmDir), dagger.ContainerWithDirectoryOpts{
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
	// previously ruby:3.1-bookworm
	_, err := root.
		WithWorkdir("/src").
		WithDirectory("/src", t.hostDir.Directory("flipt-client-ruby")).
		WithDirectory(fmt.Sprintf("/src/ext/linux_%s", architecture), t.engine.Directory("/output")).
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
	// previously gradle:8.5.0-jdk11
	path := strings.ReplaceAll(architecture, "_", "-")
	_, err := root.
		WithWorkdir("/src").
		WithDirectory("/src", t.hostDir.Directory("flipt-client-java"), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{"./.idea/", ".gradle/", "build/"},
		}).
		WithDirectory(fmt.Sprintf("/src/src/main/resources/linux-%s", path), t.engine.Directory("/output")).
		WithServiceBinding("flipt", t.flipt.WithExec(nil).AsService()).
		WithEnvVariable("FLIPT_URL", "http://flipt:8080").
		WithEnvVariable("FLIPT_AUTH_TOKEN", "secret").
		WithExec([]string{"gradle", "test"}).
		Sync(ctx)

	return err
}

// browserTests runs the browser integration test suite against a container running Flipt.
func browserTests(ctx context.Context, root *dagger.Container, t *testCase) error {
	// previously node:21.2-bookworm
	_, err := root.
		WithWorkdir("/src").
		WithDirectory("/src", t.hostDir.Directory("flipt-client-browser"), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{".node_modules/", ".gitignore", "dist/"},
		}).
		WithDirectory("/src/dist", t.engine.Directory(wasmDir), dagger.ContainerWithDirectoryOpts{
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
	// previously node:21.2-bookworm
	_, err := root.
		WithWorkdir("/src").
		WithDirectory("/src", t.hostDir.Directory("flipt-client-react"), dagger.ContainerWithDirectoryOpts{
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
	// previously dart:stable
	_, err := root.
		WithWorkdir("/src").
		WithDirectory("/src", t.hostDir.Directory("flipt-client-dart"), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{".gitignore", ".dart_tool/"},
		}).
		WithDirectory(fmt.Sprintf("/src/ext/linux_%s", architecture), t.engine.Directory("/output")).
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
	// previously mcr.microsoft.com/dotnet/sdk:8.0
	_, err := root.
		WithWorkdir("/src").
		WithDirectory("/src", t.hostDir.Directory("flipt-client-csharp"), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{".gitignore", "obj/", "bin/"},
		}).
		WithDirectory(fmt.Sprintf("/src/ext/linux_%s", architecture), t.engine.Directory("/output")).
		WithServiceBinding("flipt", t.flipt.WithExec(nil).AsService()).
		WithEnvVariable("FLIPT_URL", "http://flipt:8080").
		WithEnvVariable("FLIPT_AUTH_TOKEN", "secret").
		WithExec([]string{"dotnet", "restore"}).
		WithExec([]string{"dotnet", "build"}).
		WithExec([]string{"dotnet", "test"}).
		Sync(ctx)

	return err
}
