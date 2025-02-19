package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"maps"
	"os"
	"slices"
	"strings"
	"sync"

	"dagger.io/dagger"
	"golang.org/x/sync/errgroup"
)

type testPlatform string

const (
	linuxAMD64 testPlatform = "linux/amd64"
	linuxARM64 testPlatform = "linux/arm64"
)

func (p testPlatform) String() string {
	return string(p)
}

type testArchitecture string

const (
	amd64 testArchitecture = "x86_64"
	arm64 testArchitecture = "aarch64"
)

func (a testArchitecture) String() string {
	return string(a)
}

var (
	architecture = amd64
	platform     = linuxAMD64
	sdks         string
	sema         = make(chan struct{}, 10)
)

// containerConfig holds the base configuration for a test container
type containerConfig struct {
	base      string
	commands  []string
	platforms []testPlatform
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
	"python": {containers: []containerConfig{
		{base: "python:3.9-bookworm"},
		{base: "python:3.9-bullseye"},
		{base: "python:3.9-alpine"},
	}, fn: pythonTests},
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
	"node": {containers: []containerConfig{{base: "node:21.2-bookworm"}}, fn: nodeTests},
	"ruby": {containers: []containerConfig{
		{base: "ruby:3.1-bookworm"},
		{base: "ruby:3.1-bullseye"},
		{base: "ruby:3.1-alpine"},
	}, fn: rubyTests},
	"java": {containers: []containerConfig{
		{base: "gradle:8-jdk11"},
		{base: "gradle:8-jdk11-focal"},
		{base: "gradle:8-jdk11-alpine", platforms: []testPlatform{linuxAMD64}},
	}, fn: javaTests},
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

	var skipped []containerConfig

	for lang, t := range tests {
		lang, t := lang, t
		for _, c := range t.containers {
			c := c

			if len(c.platforms) > 0 {
				// filter out containers that don't match the current platform if specified
				if !slices.Contains(c.platforms, platform) {
					fmt.Printf("skipping %s for platform %s\n", c.base, platform)
					skipped = append(skipped, c)
					continue
				}
			}

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

	fmt.Println("\nSkipped tests:")
	for _, c := range skipped {
		fmt.Printf("⚠️ %s for platform %s\n", c.base, platform)
	}

	fmt.Println("\nFailed tests:")
	for _, result := range results {
		if result.err != nil {
			fmt.Printf("❌ %s %s\n", result.sdk, result.base)
		}
	}

	fmt.Println("\n=========================")

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

	if strings.Contains(string(daggerPlatform), "arm64") || strings.Contains(string(daggerPlatform), "aarch64") {
		platform = linuxARM64
		architecture = arm64
	}
	return nil
}

func setupFliptContainer(client *dagger.Client, hostDir *dagger.Directory) *dagger.Container {
	return client.Container().From("flipt/flipt:latest").
		WithUser("root").
		WithExec(args("mkdir -p /var/data/flipt")).
		WithDirectory("/var/data/flipt", hostDir.Directory("test/fixtures/testdata")).
		WithExec(args("chown -R flipt:flipt /var/data/flipt")).
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
	var (
		target = fmt.Sprintf("%s-unknown-linux-musl", architecture)

		// Create cache volumes for Cargo
		cargoRegistry = client.CacheVolume("cargo-registry-" + target)
		cargoGit      = client.CacheVolume("cargo-git-" + target)
		targetCache   = client.CacheVolume("cargo-target-" + target)
	)

	return client.Container(dagger.ContainerOpts{
		Platform: dagger.Platform(platform),
	}).From("rust:1.83.0-bullseye").
		// Mount cargo caches
		WithMountedCache("/usr/local/cargo/registry", cargoRegistry).
		WithMountedCache("/usr/local/cargo/git", cargoGit).
		WithMountedCache("/src/target", targetCache).
		WithExec(args("apt-get update")).
		WithExec(args("apt-get install -y build-essential musl-dev musl-tools")).
		WithExec([]string{"rustup", "target", "add", target}).
		WithWorkdir("/src").
		WithDirectory("/src/flipt-engine-ffi", hostDirectory.Directory("flipt-engine-ffi")).
		WithDirectory("/src/flipt-engine-wasm", hostDirectory.Directory("flipt-engine-wasm"), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{"pkg/", ".gitignore"},
		}).
		WithDirectory("/src/flipt-evaluation", hostDirectory.Directory("flipt-evaluation")).
		WithFile("/src/Cargo.toml", hostDirectory.File("Cargo.toml")).
		WithFile("/src/.cargo/config.toml", hostDirectory.File(".cargo/config.toml")).
		WithExec(args("cargo build -p flipt-engine-ffi --release --target " + target)).
		// Copy the static lib to where the wrapper build script can find it
		WithExec([]string{"cp", fmt.Sprintf("/src/target/%s/release/libfliptengine.a", target), "/src/flipt-engine-ffi/"}).
		// Make and run the build script
		WithExec(args("chmod +x /src/flipt-engine-ffi/build.sh")).
		WithExec(args("/src/flipt-engine-ffi/build.sh")).
		// Copy .so to output directory
		WithExec(args("mkdir -p /output")).
		WithExec(args("cp /src/flipt-engine-ffi/libfliptengine.so /output/"))
}

// getWasmTestContainer builds the wasm module for the Rust core, and the Flipt container for the client libraries to run
// their tests against.
func getWasmTestContainer(_ context.Context, client *dagger.Client, hostDirectory *dagger.Directory, target string) *dagger.Container {
	container := client.Container(dagger.ContainerOpts{
		Platform: dagger.Platform(platform),
	}).From("rust:1.83.0-bullseye").
		WithWorkdir("/src").
		WithDirectory("/src/flipt-engine-ffi", hostDirectory.Directory("flipt-engine-ffi")).
		WithDirectory("/src/flipt-engine-wasm", hostDirectory.Directory("flipt-engine-wasm"), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{"pkg/", ".gitignore"},
		}).
		WithDirectory("/src/flipt-evaluation", hostDirectory.Directory("flipt-evaluation")).
		WithFile("/src/Cargo.toml", hostDirectory.File("Cargo.toml")).
		WithExec(args("cargo build -p flipt-engine-wasm --release")) // Build the wasm module

	if architecture == "aarch64" {
		container = container.WithExec(args("apt-get update")).
			WithExec(args("apt-get -y install binaryen"))
	}

	return container.WithExec(args("cargo install wasm-pack")). // Install wasm-pack
									WithWorkdir("/src/flipt-engine-wasm").
									WithExec(args("wasm-pack build --target " + target)) // Build the wasm package
}

// pythonTests runs the python integration test suite against a container running Flipt.
func pythonTests(ctx context.Context, root *dagger.Container, t *testCase) error {
	// previously python:3.9-bookworm
	_, err := root.
		WithExec(args("pip install poetry==1.7.0")).
		WithWorkdir("/src").
		WithDirectory("/src", t.hostDir.Directory("flipt-client-python")).
		WithDirectory(fmt.Sprintf("/src/ext/linux_%s", architecture), t.engine.Directory("/output")).
		WithServiceBinding("flipt", t.flipt.WithExec(nil).AsService()).
		WithEnvVariable("FLIPT_URL", "http://flipt:8080").
		WithEnvVariable("FLIPT_AUTH_TOKEN", "secret").
		WithExec(args("poetry install --without=dev")).
		WithExec(args("poetry run test")).
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
		WithExec(args("go mod download")).
		WithExec(args("go test -v ./...")).
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
		WithExec(args("npm install")).
		WithExec(args("npm run build")).
		WithExec(args("npm test")).
		Sync(ctx)

	return err
}

// rubyTests runs the ruby integration test suite against a container running Flipt.
func rubyTests(ctx context.Context, root *dagger.Container, t *testCase) error {
	// previously ruby:3.1-bookworm
	_, err := root.
		WithWorkdir("/src").
		WithDirectory("/src", t.hostDir.Directory("flipt-client-ruby")).
		WithDirectory(fmt.Sprintf("/src/lib/ext/linux_%s", architecture), t.engine.Directory("/output")).
		WithServiceBinding("flipt", t.flipt.WithExec(nil).AsService()).
		WithEnvVariable("FLIPT_URL", "http://flipt:8080").
		WithEnvVariable("FLIPT_AUTH_TOKEN", "secret").
		WithExec(args("bundle install")).
		WithExec(args("bundle exec rspec")).
		Sync(ctx)

	return err
}

// javaTests run the java integration tests suite against a container running Flipt.
func javaTests(ctx context.Context, root *dagger.Container, t *testCase) error {
	// previously gradle:8.5.0-jdk11
	path := strings.ReplaceAll(architecture.String(), "_", "-")
	_, err := root.
		WithWorkdir("/src").
		WithDirectory("/src", t.hostDir.Directory("flipt-client-java"), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{"./.idea/", ".gradle/", "build/"},
		}).
		WithDirectory(fmt.Sprintf("/src/src/main/resources/linux-%s", path), t.engine.Directory("/output")).
		WithServiceBinding("flipt", t.flipt.WithExec(nil).AsService()).
		WithEnvVariable("FLIPT_URL", "http://flipt:8080").
		WithEnvVariable("FLIPT_AUTH_TOKEN", "secret").
		WithExec(args("gradle test")).
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
		WithExec(args("npm install")).
		WithExec(args("npm run build")).
		WithExec(args("npm test")).
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
		WithExec(args("npm install")).
		WithExec(args("npm run build")).
		WithExec(args("npm test")).
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
		WithDirectory(fmt.Sprintf("/src/ffi/linux_%s", architecture), t.engine.Directory("/output")).
		WithServiceBinding("flipt", t.flipt.WithExec(nil).AsService()).
		WithEnvVariable("FLIPT_URL", "http://flipt:8080").
		WithEnvVariable("FLIPT_AUTH_TOKEN", "secret").
		WithExec(args("dart pub get")).
		WithExec(args("dart test")).
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
		WithDirectory(fmt.Sprintf("/src/ext/ffi/linux_%s", architecture), t.engine.Directory("/output")).
		WithServiceBinding("flipt", t.flipt.WithExec(nil).AsService()).
		WithEnvVariable("FLIPT_URL", "http://flipt:8080").
		WithEnvVariable("FLIPT_AUTH_TOKEN", "secret").
		WithExec(args("dotnet restore")).
		WithExec(args("dotnet build")).
		WithExec(args("dotnet test")).
		Sync(ctx)

	return err
}
