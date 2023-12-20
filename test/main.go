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
	languages    string
	languageToFn = map[string]integrationTestFn{
		"python": pythonTests,
		"go":     goTests,
		"node":   nodeTests,
		"ruby":   rubyTests,
	}
	sema = make(chan struct{}, len(languageToFn))
)

type testArgs struct {
	arch       string
	libFile    *dagger.File
	headerFile *dagger.File
	hostDir    *dagger.Directory
}

type integrationTestFn func(context.Context, *dagger.Client, *dagger.Container, testArgs) error

func init() {
	flag.StringVar(&languages, "languages", "", "comma separated list of which language(s) to run integration tests for")
}

func main() {
	flag.Parse()

	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	var tests = make(map[string]integrationTestFn, len(languageToFn))

	maps.Copy(tests, languageToFn)

	if languages != "" {
		l := strings.Split(languages, ",")
		subset := make(map[string]integrationTestFn, len(l))
		for _, language := range l {
			testFn, ok := languageToFn[language]
			if !ok {
				return fmt.Errorf("language %s is not supported", language)
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
		Exclude: []string{"diagrams/", "build/", "tmp/"},
	})

	flipt, args := getTestDependencies(ctx, client, dir)

	var g errgroup.Group

	for _, testFn := range tests {
		testFn := testFn

		g.Go(take(func() error {
			return testFn(ctx, client, flipt, args)
		}))
	}

	err = g.Wait()

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

// getTestDependencies builds the dynamic library for the Rust core, and the Flipt container for the client libraries to run
// their tests against.
func getTestDependencies(ctx context.Context, client *dagger.Client, hostDirectory *dagger.Directory) (*dagger.Container, testArgs) {
	arch := "x86_64"
	if strings.Contains(runtime.GOARCH, "arm64") || strings.Contains(runtime.GOARCH, "aarch64") {
		arch = "arm64"
	}

	// Dynamic Library
	rust := client.Container().From("rust:1.73.0-bookworm").
		WithWorkdir("/app").
		WithDirectory("/app/flipt-engine", hostDirectory.Directory("flipt-engine")).
		WithFile("/app/Cargo.toml", hostDirectory.File("Cargo.toml")).
		WithExec([]string{"cargo", "build", "--release"})

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
		libFile:    rust.File("/app/target/release/libfliptengine.so"),
		headerFile: rust.File("/app/target/release/flipt_engine.h"),
		hostDir:    hostDirectory,
	}
}

// pythonTests runs the python integration test suite against a container running Flipt.
func pythonTests(ctx context.Context, client *dagger.Client, flipt *dagger.Container, args testArgs) error {

	_, err := client.Container().From("python:3.11-bookworm").
		WithExec([]string{"pip", "install", "poetry==1.7.0"}).
		WithWorkdir("/app").
		WithDirectory("/app", args.hostDir.Directory("flipt-client-python")).
		WithFile(fmt.Sprintf("/app/ext/libfliptengine.linux.%s.so", args.arch), args.libFile).
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
	_, err := client.Container().From("golang:1.21.3-bookworm").
		WithExec([]string{"apt-get", "update"}).
		WithExec([]string{"apt-get", "-y", "install", "build-essential"}).
		WithWorkdir("/app").
		WithDirectory("/app", args.hostDir.Directory("flipt-client-go")).
		WithFile("/app/ext/libfliptengine.so", args.libFile).
		WithFile("/app/ext/flipt_engine.h", args.headerFile).
		WithServiceBinding("flipt", flipt.WithExec(nil).AsService()).
		WithEnvVariable("FLIPT_URL", "http://flipt:8080").
		WithEnvVariable("FLIPT_AUTH_TOKEN", "secret").
		// Since the dynamic library is being sourced from a "non-standard location" we can
		// modify the LD_LIBRARY_PATH variable to inform the linker different locations for
		// dynamic libraries.
		WithEnvVariable("LD_LIBRARY_PATH", "/app/ext:$LD_LIBRARY_PATH").
		WithExec([]string{"go", "mod", "download"}).
		WithExec([]string{"go", "test", "./..."}).
		Sync(ctx)

	return err
}

// nodeTests runs the node integration test suite against a container running Flipt.
func nodeTests(ctx context.Context, client *dagger.Client, flipt *dagger.Container, args testArgs) error {
	_, err := client.Container().From("node:21.2-bookworm").
		WithWorkdir("/app").
		// The node_modules should never be version controlled, but we will exclude it here
		// just to be safe.
		WithDirectory("/app", args.hostDir.Directory("flipt-client-node"), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{"./node_modules/"},
		}).
		WithFile("/app/ext/libfliptengine.so", args.libFile).
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
	_, err := client.Container().From("ruby:3.1-bookworm").
		WithWorkdir("/app").
		WithDirectory("/app", args.hostDir.Directory("flipt-client-ruby")).
		WithFile("/app/lib/ext/libfliptengine.so", args.libFile).
		WithServiceBinding("flipt", flipt.WithExec(nil).AsService()).
		WithEnvVariable("FLIPT_URL", "http://flipt:8080").
		WithEnvVariable("FLIPT_AUTH_TOKEN", "secret").
		WithExec([]string{"bundle", "install"}).
		WithExec([]string{"bundle", "exec", "rspec"}).
		Sync(ctx)

	return err
}
