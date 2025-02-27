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

var (
	sdks   string
	groups string
	sema   = make(chan struct{}, 10)
)

// containerConfig holds the base configuration for a test container
type containerConfig struct {
	base  string
	setup []string
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
	wasmJSDir  = "/src/flipt-engine-wasm-js/pkg"
)

var (
	pythonVersions = []containerConfig{
		{base: "python:3.9-bookworm"},
		{base: "python:3.9-bullseye"},
		{base: "python:3.9-alpine"},
	}

	goVersions = []containerConfig{
		{base: "golang:1.23-bookworm", setup: []string{"apt-get update", "apt-get install -y build-essential"}},
		{base: "golang:1.23-bullseye", setup: []string{"apt-get update", "apt-get install -y build-essential"}},
		{base: "golang:1.23-alpine", setup: []string{"apk update", "apk add --no-cache build-base"}},
	}

	nodeVersions = []containerConfig{
		{base: "node:21.2-bookworm"},
		{base: "node:21.2-bullseye"},
		{base: "node:21.2-alpine"},
	}

	rubyVersions = []containerConfig{
		{base: "ruby:3.1-bookworm"},
		{base: "ruby:3.1-bullseye"},
		{base: "ruby:3.1-alpine", setup: []string{"apk update", "apk add --no-cache build-base"}},
	}

	javaVersions = []containerConfig{
		{base: "gradle:8-jdk11"},
		{base: "gradle:8-jdk11-focal"},
		{base: "gradle:8-jdk11-alpine"},
	}

	browserVersions = []containerConfig{
		{base: "node:21.2-bookworm"},
	}

	dartVersions = []containerConfig{
		{base: "dart:stable"},
	}

	reactVersions = []containerConfig{
		{base: "node:21.2-bookworm"},
	}

	csharpVersions = []containerConfig{
		{base: "mcr.microsoft.com/dotnet/sdk:8.0"},
		{base: "mcr.microsoft.com/dotnet/sdk:8.0-alpine"},
	}
)

// sdkToConfig defines the test configurations for each SDK
var sdkToConfig = map[string]testConfig{
	"python":  {containers: pythonVersions, fn: pythonTests},
	"go":      {containers: goVersions, fn: goTests},
	"node":    {containers: nodeVersions, fn: nodeTests},
	"ruby":    {containers: rubyVersions, fn: rubyTests},
	"java":    {containers: javaVersions, fn: javaTests},
	"browser": {containers: browserVersions, fn: browserTests},
	"dart":    {containers: dartVersions, fn: dartTests},
	"react":   {containers: reactVersions, fn: reactTests},
	"csharp":  {containers: csharpVersions, fn: csharpTests},
}

// sdkGroups defines which SDKs belong to each group
var sdkGroups = map[string][]string{
	"ffi":  {"python", "ruby", "java", "dart", "csharp"},
	"wasm": {"go"},
	"js":   {"node", "browser", "react"},
}

func init() {
	flag.StringVar(&sdks, "sdks", "", "comma separated list of which language(s) to run integration tests for")
	flag.StringVar(&groups, "groups", "", "comma separated list of which group(s) to run integration tests for")
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
		wg          sync.WaitGroup // WaitGroup for tracking result senders
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

	if groups != "" {
		l := strings.Split(groups, ",")
		subset := make(map[string]testConfig, len(l))
		for _, group := range l {
			for _, language := range sdkGroups[group] {
				testFn, ok := sdkToConfig[language]
				if !ok {
					return fmt.Errorf("sdk %s is not supported in group %s", language, group)
				}
				subset[language] = testFn
			}
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

	flipt := setupFliptContainer(client, hostDir)

	var g errgroup.Group

	container := client.Container(dagger.ContainerOpts{
		Platform: dagger.Platform("linux/amd64"),
	})

	for lang, t := range tests {
		lang, t := lang, t
		for _, c := range t.containers {
			c := c
			wg.Add(1) // Increment WaitGroup for each test

			g.Go(take(func() error {
				defer wg.Done() // Decrement WaitGroup when test is done

				testCase := &testCase{
					sdk:     lang,
					hostDir: hostDir,
					flipt:   flipt,
				}

				switch lang {
				case "node":
					testCase.engine = getWasmJSBuildContainer(ctx, client, hostDir, "nodejs")
				case "browser", "react":
					testCase.engine = getWasmJSBuildContainer(ctx, client, hostDir, "web")
				case "go":
					testCase.engine = getWasmBuildContainer(ctx, client, hostDir)
				default:
					testCase.engine = getFFIBuildContainer(ctx, client, hostDir)
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

	var resultsWg sync.WaitGroup
	resultsWg.Add(1)

	go func() {
		defer resultsWg.Done()
		for result := range resultsChan {
			results = append(results, result)
		}
	}()

	err = g.Wait()
	wg.Wait()          // Wait for all tests to complete sending results
	close(resultsChan) // Close channel after all results have been sent

	resultsWg.Wait() // Wait for all results to be processed

	printResults(results)

	return err
}

// take is a helper function to limit the concurrency of the tests
func take(fn func() error) func() error {
	return func() error {
		// insert into semaphore channel to maintain
		// a max concurrency
		sema <- struct{}{}
		defer func() { <-sema }()

		return fn()
	}
}

// setupFliptContainer creates a Flipt container with the test data mounted
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

// createBaseContainer creates a base container with common settings
func createBaseContainer(client *dagger.Client, config containerConfig) *dagger.Container {
	container := client.Container(dagger.ContainerOpts{
		Platform: dagger.Platform("linux/amd64"),
	}).From(config.base)

	for _, cmd := range config.setup {
		container = container.WithExec(args(cmd))
	}

	return container
}

// getFFIBuildContainer builds the shared object library for the Rust core for the client libraries to run
// their tests against.
func getFFIBuildContainer(_ context.Context, client *dagger.Client, hostDirectory *dagger.Directory) *dagger.Container {
	const target = "x86_64-unknown-linux-musl"

	return client.Container(dagger.ContainerOpts{
		Platform: dagger.Platform("linux/amd64"),
	}).From("rust:1.83.0-bullseye"). // requires older version of glibc for best compatibility
						WithMountedCache("/usr/local/cargo/registry", client.CacheVolume("cargo-registry")).
						WithMountedCache("/usr/local/cargo/git", client.CacheVolume("cargo-git")).
						WithExec(args("apt-get update")).
						WithExec(args("apt-get install -y build-essential musl-dev musl-tools")).
						WithWorkdir("/src").
						WithDirectory("/src/flipt-engine-ffi", hostDirectory.Directory("flipt-engine-ffi")).
						WithDirectory("/src/flipt-engine-wasm", hostDirectory.Directory("flipt-engine-wasm"), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{"pkg/", ".gitignore"},
		}).
		WithDirectory("/src/flipt-engine-wasm-js", hostDirectory.Directory("flipt-engine-wasm-js"), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{"pkg/", ".gitignore"},
		}).
		WithDirectory("/src/flipt-evaluation", hostDirectory.Directory("flipt-evaluation")).
		WithFile("/src/Cargo.toml", hostDirectory.File("Cargo.toml")).
		WithFile("/src/.cargo/config.toml", hostDirectory.File(".cargo/config.toml")).
		WithExec(args("chmod +x /src/flipt-engine-ffi/build.sh")).
		WithExec(args("/src/flipt-engine-ffi/build.sh linux-x86_64"))
}

// getWasmBuildContainer builds the wasm module for the Rust core for the client libraries to run
// their tests against.
func getWasmBuildContainer(_ context.Context, client *dagger.Client, hostDirectory *dagger.Directory) *dagger.Container {
	const target = "wasm32-wasip1"

	container := client.Container(dagger.ContainerOpts{
		Platform: dagger.Platform("linux/amd64"),
	}).From("rust:1.83.0-bullseye").
		WithMountedCache("/usr/local/cargo/registry", client.CacheVolume("cargo-registry")).
		WithMountedCache("/usr/local/cargo/git", client.CacheVolume("cargo-git")).
		WithWorkdir("/src").
		WithDirectory("/src/flipt-engine-ffi", hostDirectory.Directory("flipt-engine-ffi")).
		WithDirectory("/src/flipt-engine-wasm", hostDirectory.Directory("flipt-engine-wasm"), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{"pkg/", ".gitignore"},
		}).
		WithDirectory("/src/flipt-engine-wasm-js", hostDirectory.Directory("flipt-engine-wasm-js"), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{"pkg/", ".gitignore"},
		}).
		WithDirectory("/src/flipt-evaluation", hostDirectory.Directory("flipt-evaluation")).
		WithFile("/src/Cargo.toml", hostDirectory.File("Cargo.toml")).
		WithExec(args("rustup target add " + target)).
		WithExec(args("cargo build -p flipt-engine-wasm --release --target " + target)) // Build the wasm module

	return container.WithExec(args("cargo install wasm-opt")).
		WithExec(args("wasm-opt --converge --flatten --rereloop -Oz -Oz --gufa -o /src/target/wasm32-wasip1/release/flipt_engine_wasm.wasm /src/target/wasm32-wasip1/release/flipt_engine_wasm.wasm"))
}

// getWasmJSBuildContainer builds the wasm module for the Rust core for the client libraries to run
// their tests against.
func getWasmJSBuildContainer(_ context.Context, client *dagger.Client, hostDirectory *dagger.Directory, target string) *dagger.Container {
	container := client.Container(dagger.ContainerOpts{
		Platform: dagger.Platform("linux/amd64"),
	}).From("rust:1.83.0-bullseye").
		WithMountedCache("/usr/local/cargo/registry", client.CacheVolume("cargo-registry")).
		WithMountedCache("/usr/local/cargo/git", client.CacheVolume("cargo-git")).
		WithWorkdir("/src").
		WithDirectory("/src/flipt-engine-ffi", hostDirectory.Directory("flipt-engine-ffi")).
		WithDirectory("/src/flipt-engine-wasm", hostDirectory.Directory("flipt-engine-wasm"), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{"pkg/", ".gitignore"},
		}).
		WithDirectory("/src/flipt-engine-wasm-js", hostDirectory.Directory("flipt-engine-wasm-js"), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{"pkg/", ".gitignore"},
		}).
		WithDirectory("/src/flipt-evaluation", hostDirectory.Directory("flipt-evaluation")).
		WithFile("/src/Cargo.toml", hostDirectory.File("Cargo.toml")).
		WithExec(args("cargo build -p flipt-engine-wasm-js --release")) // Build the wasm module

	return container.WithExec(args("cargo install wasm-opt wasm-pack")). // Install wasm-pack
										WithWorkdir("/src/flipt-engine-wasm-js").
										WithExec(args("wasm-pack build --target " + target)) // Build the wasm package
}

// pythonTests runs the python integration test suite against a container running Flipt.
func pythonTests(ctx context.Context, root *dagger.Container, t *testCase) error {
	// previously python:3.9-bookworm
	_, err := root.
		WithExec(args("pip install poetry==1.7.0")).
		WithWorkdir("/src").
		WithDirectory("/src", t.hostDir.Directory("flipt-client-python")).
		WithDirectory("/src/ext/linux_x86_64", t.engine.Directory("/tmp/ffi")).
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
		WithFile("/src/ext/flipt_engine_wasm.wasm", t.engine.File("/src/target/wasm32-wasip1/release/flipt_engine_wasm.wasm")).
		WithServiceBinding("flipt", t.flipt.WithExec(nil).AsService()).
		WithEnvVariable("FLIPT_URL", "http://flipt:8080").
		WithEnvVariable("FLIPT_AUTH_TOKEN", "secret").
		WithExec(args("go mod download")).
		WithExec(args("go test -v -timeout 30s ./...")).
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
		WithDirectory("/src/dist", t.engine.Directory(wasmJSDir), dagger.ContainerWithDirectoryOpts{
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
		WithDirectory("/src/lib/ext/linux_x86_64", t.engine.Directory("/tmp/ffi")).
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
	_, err := root.
		WithWorkdir("/src").
		WithDirectory("/src", t.hostDir.Directory("flipt-client-java"), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{"./.idea/", ".gradle/", "build/"},
		}).
		WithDirectory("/src/src/main/resources/linux-x86-64", t.engine.Directory("/tmp/ffi")).
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
		WithDirectory("/src/dist", t.engine.Directory(wasmJSDir), dagger.ContainerWithDirectoryOpts{
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
		WithDirectory("/src/lib/src/ffi/linux_x86_64", t.engine.Directory("/tmp/ffi")).
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
		WithDirectory("/src/src/FliptClient/ext/ffi/linux_x86_64", t.engine.Directory("/tmp/ffi")).
		WithServiceBinding("flipt", t.flipt.WithExec(nil).AsService()).
		WithEnvVariable("FLIPT_URL", "http://flipt:8080").
		WithEnvVariable("FLIPT_AUTH_TOKEN", "secret").
		WithExec(args("dotnet clean")).
		WithExec(args("dotnet restore")).
		WithExec(args("dotnet build")).
		WithExec(args("dotnet test")).
		Sync(ctx)

	return err
}

func printResults(results []testResult) {
	var (
		success []testResult
		failed  []testResult

		isCI        = os.Getenv("CI") != ""
		summaryFile = os.Getenv("GITHUB_STEP_SUMMARY")
	)

	for _, result := range results {
		if result.err != nil {
			failed = append(failed, result)
		} else {
			success = append(success, result)
		}
	}

	// Sort each slice by sdk + base + platform
	sortResults := func(results []testResult) {
		slices.SortFunc(results, func(a, b testResult) int {
			aKey := fmt.Sprintf("%s-%s", a.sdk, a.base)
			bKey := fmt.Sprintf("%s-%s", b.sdk, b.base)
			return strings.Compare(aKey, bKey)
		})
	}

	sortResults(success)
	sortResults(failed)

	if isCI && summaryFile != "" {
		// Format results as GitHub Actions workflow summary
		var summary strings.Builder

		summary.WriteString("# Integration Test Results\n\n")

		// Successful Tests Table
		if len(success) > 0 {
			summary.WriteString(fmt.Sprintf("## ✅ Successful Tests (%d/%d)\n\n", len(success), len(results)))
			summary.WriteString("| SDK | Base Image |\n")
			summary.WriteString("|-----|------------|\n")
			for _, result := range success {
				summary.WriteString(fmt.Sprintf("| %s | %s |\n", result.sdk, result.base))
			}
		}

		// Failed Tests Table
		if len(failed) > 0 {
			summary.WriteString(fmt.Sprintf("\n## ❌ Failed Tests (%d/%d)\n\n", len(failed), len(results)))
			summary.WriteString("| SDK | Base Image |\n")
			summary.WriteString("|-----|------------|\n")
			for _, result := range failed {
				summary.WriteString(fmt.Sprintf("| %s | %s |\n", result.sdk, result.base))
			}
		}

		// Write to GITHUB_STEP_SUMMARY file
		if err := os.WriteFile(summaryFile, []byte(summary.String()), 0644); err != nil {
			fmt.Printf("Failed to write GitHub summary: %v\n", err)
		}
	}

	// Always print to stdout for local runs and CI logging
	fmt.Println("\n=== Test Results Summary ===")

	if len(success) > 0 {
		fmt.Printf("\nSuccessful tests (%d/%d)\n", len(success), len(results))
		fmt.Println("--------------------------------")
		for _, result := range success {
			fmt.Printf("✅ %s %s\n", result.sdk, result.base)
		}
	}

	if len(failed) > 0 {
		fmt.Printf("\nFailed tests (%d/%d)\n", len(failed), len(results))
		fmt.Println("--------------------------------")
		for _, result := range failed {
			fmt.Printf("❌ %s %s\n", result.sdk, result.base)
		}
	}

	fmt.Println("\n=========================")
}
