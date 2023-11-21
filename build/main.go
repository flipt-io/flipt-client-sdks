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
	languages    string
	languageToFn = map[string]integrationTestFn{
		"python": pythonTests,
	}
)

type integrationTestFn func(context.Context, *dagger.Client, *dagger.Container, *dagger.File, *dagger.Directory) error

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
	var languagesTestsMap = map[string]integrationTestFn{
		"python": pythonTests,
	}

	if languages != "" {
		l := strings.Split(languages, ",")
		tlm := make(map[string]integrationTestFn, len(l))
		for _, language := range l {
			if testFn, ok := languageToFn[language]; !ok {
				return fmt.Errorf("language %s is not supported", language)
			} else {
				tlm[language] = testFn
			}
		}

		languagesTestsMap = tlm
	}

	ctx := context.Background()

	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stderr))
	if err != nil {
		return err
	}
	defer client.Close()

	dir := client.Host().Directory(".")

	flipt, dynamicLibrary := getTestDependencies(ctx, client, dir)

	for _, testFn := range languagesTestsMap {
		err := testFn(ctx, client, flipt, dynamicLibrary, dir)
		if err != nil {
			return err
		}
	}

	return nil
}

// getTestDependencies builds the dynamic library for the Rust core, and the Flipt container for the client libraries to run
// their tests against.
func getTestDependencies(ctx context.Context, client *dagger.Client, hostDirectory *dagger.Directory) (*dagger.Container, *dagger.File) {
	// Dynamic Library
	rust := client.Container().From("rust:1.73.0-buster").
		WithWorkdir("/app").
		WithDirectory("/app/flipt-engine", hostDirectory.Directory("flipt-engine")).
		WithFile("/app/Cargo.toml", hostDirectory.File("Cargo.toml")).
		WithExec([]string{"cargo", "build", "--release"})

	// Flipt
	flipt := client.Container().From("flipt/flipt:nightly").
		WithUser("root").
		WithExec([]string{"mkdir", "-p", "/var/data/flipt"}).
		WithDirectory("/var/data/flipt", hostDirectory.Directory("build/fixtures/testdata")).
		WithExec([]string{"chown", "-R", "flipt:flipt", "/var/data/flipt"}).
		WithUser("flipt").
		WithEnvVariable("FLIPT_STORAGE_TYPE", "local").
		WithEnvVariable("FLIPT_STORAGE_LOCAL_PATH", "/var/data/flipt").
		WithExposedPort(8080)

	return flipt, rust.File("/app/target/release/libfliptengine.so")
}

// pythonTests runs the Poetry test suite against a container running Flipt through the dynamic library.
func pythonTests(ctx context.Context, client *dagger.Client, flipt *dagger.Container, dynamicLibrary *dagger.File, hostDirectory *dagger.Directory) error {
	// Poetry tests
	_, err := client.Container().From("python:3.11-buster").
		WithExec([]string{"pip", "install", "poetry==1.7.0"}).
		WithWorkdir("/app").
		WithDirectory("/app", hostDirectory.Directory("flipt-client-python")).
		WithFile("/app/libfliptengine.so", dynamicLibrary).
		WithServiceBinding("flipt", flipt.WithExec(nil).AsService()).
		WithEnvVariable("FLIPT_URL", "http://flipt:8080").
		WithEnvVariable("FLIPT_ENGINE_LIB_PATH", "/app/libfliptengine.so").
		WithExec([]string{"poetry", "install", "--without=dev"}).
		WithExec([]string{"poetry", "run", "test"}).
		Sync(ctx)
	if err != nil {
		return err
	}

	return nil
}
