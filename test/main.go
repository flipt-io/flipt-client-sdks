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
	setup     []string
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
	sdk      string
	base     string
	err      error
	platform testPlatform
	skipped  bool
}

const (
	headerFile = "/src/flipt-engine-ffi/include/flipt_engine.h"
	wasmDir    = "/src/flipt-engine-wasm/pkg"
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
		{base: "gradle:8-jdk11-alpine", platforms: []testPlatform{linuxAMD64}},
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

// Define test configurations
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
			skipped := false
			wg.Add(1) // Increment WaitGroup for each test

			if len(c.platforms) > 0 {
				// filter out containers that don't match the current platform if specified
				if !slices.Contains(c.platforms, platform) {
					skipped = true
				}
			}

			g.Go(take(func() error {
				defer wg.Done() // Decrement WaitGroup when test is done

				// if the container is skipped, send a result with the skipped flag set
				if skipped {
					resultsChan <- testResult{
						sdk:      lang,
						base:     c.base,
						platform: platform,
						skipped:  true,
					}
					return nil
				}

				testCase := &testCase{
					sdk:     lang,
					hostDir: hostDir,
					flipt:   flipt,
				}

				switch lang {
				case "node":
					testCase.engine = getWasmBuildContainer(ctx, client, hostDir, "nodejs")
				case "browser", "react":
					testCase.engine = getWasmBuildContainer(ctx, client, hostDir, "web")
				default:
					testCase.engine = getFFIBuildContainer(ctx, client, hostDir)
				}

				container = createBaseContainer(client, c)

				err := t.fn(ctx, container, testCase)
				resultsChan <- testResult{
					sdk:      lang,
					base:     c.base,
					err:      err,
					platform: platform,
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

// detectPlatform detects the platform and architecture of the host machine
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
		Platform: dagger.Platform(platform),
	}).From(config.base)

	for _, cmd := range config.setup {
		container = container.WithExec(args(cmd))
	}

	return container
}

// getFFIBuildContainer builds the shared object library for the Rust core for the client libraries to run
// their tests against.
func getFFIBuildContainer(_ context.Context, client *dagger.Client, hostDirectory *dagger.Directory) *dagger.Container {
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
		WithWorkdir("/src").
		WithDirectory("/src/flipt-engine-ffi", hostDirectory.Directory("flipt-engine-ffi")).
		WithDirectory("/src/flipt-engine-wasm", hostDirectory.Directory("flipt-engine-wasm"), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{"pkg/", ".gitignore"},
		}).
		WithDirectory("/src/flipt-evaluation", hostDirectory.Directory("flipt-evaluation")).
		WithFile("/src/Cargo.toml", hostDirectory.File("Cargo.toml")).
		WithFile("/src/.cargo/config.toml", hostDirectory.File(".cargo/config.toml")).
		WithExec(args("chmod +x /src/flipt-engine-ffi/build.sh")).
		WithExec(args("/src/flipt-engine-ffi/build.sh linux-" + string(architecture)))
}

// getWasmBuildContainer builds the wasm module for the Rust core for the client libraries to run
// their tests against.
func getWasmBuildContainer(_ context.Context, client *dagger.Client, hostDirectory *dagger.Directory, target string) *dagger.Container {
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
		WithDirectory(fmt.Sprintf("/src/ext/linux_%s", architecture), t.engine.Directory("/tmp/ffi")).
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
		WithDirectory(fmt.Sprintf("/src/ext/linux_%s", architecture), t.engine.Directory("/tmp/ffi")).
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
		WithDirectory(fmt.Sprintf("/src/lib/ext/linux_%s", architecture), t.engine.Directory("/tmp/ffi")).
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
		WithDirectory(fmt.Sprintf("/src/src/main/resources/linux-%s", path), t.engine.Directory("/tmp/ffi")).
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
		WithDirectory(fmt.Sprintf("/src/lib/src/ffi/linux_%s", architecture), t.engine.Directory("/tmp/ffi")).
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
		WithDirectory(fmt.Sprintf("/src/src/FliptClient/ext/ffi/linux_%s", architecture), t.engine.Directory("/tmp/ffi")).
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
		skipped []testResult
		success []testResult
		failed  []testResult

		isCI        = os.Getenv("CI") != ""
		summaryFile = os.Getenv("GITHUB_STEP_SUMMARY")
	)

	for _, result := range results {
		if result.skipped {
			skipped = append(skipped, result)
		} else if result.err != nil {
			failed = append(failed, result)
		} else {
			success = append(success, result)
		}
	}

	// Sort each slice by sdk + base + platform
	sortResults := func(results []testResult) {
		slices.SortFunc(results, func(a, b testResult) int {
			aKey := fmt.Sprintf("%s-%s-%s", a.sdk, a.base, a.platform)
			bKey := fmt.Sprintf("%s-%s-%s", b.sdk, b.base, b.platform)
			return strings.Compare(aKey, bKey)
		})
	}

	sortResults(skipped)
	sortResults(success)
	sortResults(failed)

	if isCI && summaryFile != "" {
		// Format results as GitHub Actions workflow summary
		var summary strings.Builder

		summary.WriteString("# Integration Test Results\n\n")

		// Successful Tests Table
		if len(success) > 0 {
			summary.WriteString(fmt.Sprintf("## ✅ Successful Tests (%d/%d)\n\n", len(success), len(results)))
			summary.WriteString("| SDK | Base Image | Platform |\n")
			summary.WriteString("|-----|------------|----------|\n")
			for _, result := range success {
				summary.WriteString(fmt.Sprintf("| %s | %s | %s |\n", result.sdk, result.base, result.platform))
			}
		}

		// Failed Tests Table
		if len(failed) > 0 {
			summary.WriteString(fmt.Sprintf("\n## ❌ Failed Tests (%d/%d)\n\n", len(failed), len(results)))
			summary.WriteString("| SDK | Base Image | Platform |\n")
			summary.WriteString("|-----|------------|----------|\n")
			for _, result := range failed {
				summary.WriteString(fmt.Sprintf("| %s | %s | %s |\n",
					result.sdk, result.base, result.platform))
			}
		}

		// Skipped Tests Table
		if len(skipped) > 0 {
			summary.WriteString(fmt.Sprintf("\n## ⚠️ Skipped Tests (%d/%d)\n\n", len(skipped), len(results)))
			summary.WriteString("| Base Image | Platform |\n")
			summary.WriteString("|------------|----------|\n")
			for _, c := range skipped {
				summary.WriteString(fmt.Sprintf("| %s | %s |\n", c.base, platform))
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

	if len(skipped) > 0 {
		fmt.Printf("\nSkipped tests (%d/%d)\n", len(skipped), len(results))
		fmt.Println("--------------------------------")
		for _, c := range skipped {
			fmt.Printf("⚠️ %s for platform %s\n", c.base, platform)
		}
	}

	fmt.Println("\n=========================")
}
