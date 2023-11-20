package main

import (
	"context"
	"log"
	"os"

	"dagger.io/dagger"
)

func main() {
	ctx := context.Background()
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stderr))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	dir := client.Host().Directory(".")

	flipt, dynamicLibrary := getTestDependencies(ctx, client, dir)

	if err := pythonTests(context.Background(), client, flipt, dynamicLibrary, dir); err != nil {
		log.Fatal(err)
	}
}

func getTestDependencies(ctx context.Context, client *dagger.Client, hostDirectory *dagger.Directory) (*dagger.Container, *dagger.File) {
	// Dynamic Library
	rust := client.Container().From("rust:1.73.0-buster").
		WithWorkdir("/app").
		WithDirectory("/app/flipt-engine", hostDirectory.Directory("flipt-engine")).
		WithFile("/app/Cargo.toml", hostDirectory.File("Cargo.toml")).
		WithExec([]string{"cargo", "build", "--release"})

	// Flipt
	flipt := client.Container().From("flipt/flipt:latest").
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
