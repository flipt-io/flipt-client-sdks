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
	sdks    string
	sdkToFn = map[string]integrationTestFn{
		"python": pythonTests,
		"go":     goTests,
		"node":   nodeTests,
		"ruby":   rubyTests,
		"java":   javaTests,
		//"browser": browserTests,
	}
	sema = make(chan struct{}, 5)
)

type testArgs struct {
	os         string
	arch       string
	libFile    *dagger.File
	headerFile *dagger.File
	hostDir    *dagger.Directory
}

type integrationTestFn func(context.Context, *dagger.Client, *dagger.Container, testArgs) error

func init() {
	flag.StringVar(&sdks, "sdks", "", "comma separated list of which language(s) to run integration tests for")
}

func main() {
	flag.Parse()

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

	flipt, args := getTestDependencies(ctx, client, dir)

	var g errgroup.Group

	for _, fn := range tests {
		fn := fn
		g.Go(take(func() error {
			return fn(ctx, client, flipt, args)
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

// getTestDependencies builds the dynamic library for the Rust core, and the Flipt container for the client libraries to run
// their tests against.
func getTestDependencies(_ context.Context, client *dagger.Client, hostDirectory *dagger.Directory) (*dagger.Container, testArgs) {
	arch := "x86_64"
	if strings.Contains(runtime.GOARCH, "arm64") || strings.Contains(runtime.GOARCH, "aarch64") {
		arch = "arm64"
	}

	// Engines
	rust := client.Container().From("rust:1.74.0-bookworm").
		WithWorkdir("/src").
		WithDirectory("/src/flipt-engine-ffi", hostDirectory.Directory("flipt-engine-ffi")).
		WithDirectory("/src/flipt-client-browser", hostDirectory.Directory("flipt-client-browser")).
		WithDirectory("/src/flipt-evaluation", hostDirectory.Directory("flipt-evaluation")).
		WithFile("/src/Cargo.toml", hostDirectory.File("Cargo.toml")).
		WithExec([]string{"cargo", "build", "--release"}) // Build the dynamic library

	if arch == "arm64" {
		rust = rust.WithExec([]string{"apt-get", "update"}).
			WithExec([]string{"apt-get", "-y", "install", "binaryen"})
	}

	rust = rust.
		WithExec([]string{"cargo", "install", "wasm-pack"}). // Install wasm-pack
		WithWorkdir("/src/flipt-client-browser").
		WithExec([]string{"wasm-pack", "build", "--target", "web"}) // Build the wasm package

	// Flipt
	flipt := client.Container().From("flipt/flipt:latest").
		WithUser("root").
		WithExec([]string{"mkdir", "-p", "/var/data/flipt"}).
		WithDirectory("/var/data/flipt", hostDirectory.Directory("test/fixtures/testdata")).
		WithExec([]string{"chown", "-R", "flipt:flipt", "/var/data/flipt"}).
		WithUser("flipt").
		WithEnvVariable("FLIPT_STORAGE_TYPE", "local").
		WithEnvVariable("FLIPT_STORAGE_LOCAL_PATH", "/var/data/flipt").
		WithEnvVariable("FLIPT_AUTHENTICATION_METHODS_TOKEN_ENABLED", "1").
		WithEnvVariable("FLIPT_AUTHENTICATION_METHODS_TOKEN_BOOTSTRAP_TOKEN", "secret").
		WithEnvVariable("FLIPT_AUTHENTICATION_REQUIRED", "1").
		WithExposedPort(8080)

	return flipt, testArgs{
		arch:       arch,
		libFile:    rust.File("/src/target/release/libfliptengine.so"),
		headerFile: rust.File("/src/flipt-engine-ffi/include/flipt_engine.h"),
		hostDir:    hostDirectory,
	}
}

// pythonTests runs the python integration test suite against a container running Flipt.
func pythonTests(ctx context.Context, client *dagger.Client, flipt *dagger.Container, args testArgs) error {
	_, err := client.Pipeline("python").Container().From("python:3.11-bookworm").
		WithExec([]string{"pip", "install", "poetry==1.7.0"}).
		WithWorkdir("/src").
		WithDirectory("/src", args.hostDir.Directory("flipt-client-python")).
		WithFile(fmt.Sprintf("/src/ext/linux_%s/libfliptengine.so", args.arch), args.libFile).
		WithServiceBinding("flipt", flipt.WithExec(nil).AsService()).
		WithEnvVariable("FLIPT_URL", "http://flipt:8080").
		WithEnvVariable("FLIPT_AUTH_TOKEN", "secret").
		WithExec([]string{"poetry", "install", "--without=dev"}).
		WithExec([]string{"poetry", "run", "test"}).
		Sync(ctx)

	return err
}

// goTests runs the golang integration test suite against a container running Flipt.
func goTests(ctx context.Context, client *dagger.Client, flipt *dagger.Container, args testArgs) error {
	_, err := client.Pipeline("go").Container().From("golang:1.21.3-bookworm").
		WithExec([]string{"apt-get", "update"}).
		WithExec([]string{"apt-get", "-y", "install", "build-essential"}).
		WithWorkdir("/src").
		WithDirectory("/src", args.hostDir.Directory("flipt-client-go")).
		WithFile(fmt.Sprintf("/src/ext/linux_%s/libfliptengine.so", args.arch), args.libFile).
		WithFile("/src/ext/flipt_engine.h", args.headerFile).
		WithServiceBinding("flipt", flipt.WithExec(nil).AsService()).
		WithEnvVariable("FLIPT_URL", "http://flipt:8080").
		WithEnvVariable("FLIPT_AUTH_TOKEN", "secret").
		// Since the dynamic library is being sourced from a "non-standard location" we can
		// modify the LD_LIBRARY_PATH variable to inform the linker different locations for
		// dynamic libraries.
		WithEnvVariable("LD_LIBRARY_PATH", fmt.Sprintf("/src/ext/linux_%s:$LD_LIBRARY_PATH", args.arch)).
		WithExec([]string{"go", "mod", "download"}).
		WithExec([]string{"go", "test", "./..."}).
		Sync(ctx)

	return err
}

// nodeTests runs the node integration test suite against a container running Flipt.
func nodeTests(ctx context.Context, client *dagger.Client, flipt *dagger.Container, args testArgs) error {
	_, err := client.Pipeline("node").Container().From("node:21.2-bookworm").
		WithWorkdir("/src").
		// The node_modules should never be version controlled, but we will exclude it here
		// just to be safe.
		WithDirectory("/src", args.hostDir.Directory("flipt-client-node"), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{"./node_modules/"},
		}).
		WithFile(fmt.Sprintf("/src/ext/linux_%s/libfliptengine.so", args.arch), args.libFile).
		WithServiceBinding("flipt", flipt.WithExec(nil).AsService()).
		WithEnvVariable("FLIPT_URL", "http://flipt:8080").
		WithEnvVariable("FLIPT_AUTH_TOKEN", "secret").
		WithExec([]string{"npm", "install"}).
		WithExec([]string{"npm", "test"}).
		Sync(ctx)

	return err
}

// rubyTests runs the ruby integration test suite against a container running Flipt.
func rubyTests(ctx context.Context, client *dagger.Client, flipt *dagger.Container, args testArgs) error {
	_, err := client.Pipeline("ruby").Container().From("ruby:3.1-bookworm").
		WithWorkdir("/src").
		WithDirectory("/src", args.hostDir.Directory("flipt-client-ruby")).
		WithFile(fmt.Sprintf("/src/lib/ext/linux_%s/libfliptengine.so", args.arch), args.libFile).
		WithServiceBinding("flipt", flipt.WithExec(nil).AsService()).
		WithEnvVariable("FLIPT_URL", "http://flipt:8080").
		WithEnvVariable("FLIPT_AUTH_TOKEN", "secret").
		WithExec([]string{"bundle", "install"}).
		WithExec([]string{"bundle", "exec", "rspec"}).
		Sync(ctx)

	return err
}

// javaTests run the java integration tests suite against a container running Flipt.
func javaTests(ctx context.Context, client *dagger.Client, flipt *dagger.Container, args testArgs) error {
	path := "x86-64"

	if args.arch == "arm64" {
		path = "arm"
	}

	_, err := client.Pipeline("java").Container().From("gradle:8.5.0-jdk11").
		WithWorkdir("/src").
		WithDirectory("/src", args.hostDir.Directory("flipt-client-java"), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{"./.idea/"},
		}).
		WithFile(fmt.Sprintf("/src/src/main/resources/linux-%s/libfliptengine.so", path), args.libFile).
		WithServiceBinding("flipt", flipt.WithExec(nil).AsService()).
		WithEnvVariable("FLIPT_URL", "http://flipt:8080").
		WithEnvVariable("FLIPT_AUTH_TOKEN", "secret").
		WithExec([]string{"gradle", "test"}).
		Sync(ctx)

	return err
}

func browserTests(ctx context.Context, client *dagger.Client, flipt *dagger.Container, args testArgs) error {
	_, err := client.Pipeline("browser").Container().From("node:21.2-bookworm").
		WithWorkdir("/src").
		WithDirectory("/src", args.hostDir.Directory("flipt-client-browser")).
		WithServiceBinding("flipt", flipt.WithExec(nil).AsService()).
		WithEnvVariable("FLIPT_URL", "http://flipt:8080").
		WithEnvVariable("FLIPT_AUTH_TOKEN", "secret").
		WithExec([]string{"npm", "install"}).
		WithExec([]string{"npm", "test"}).
		Sync(ctx)

	return err
}
