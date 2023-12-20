package main

import (
	"context"
	"os"

	"dagger.io/dagger"
)

func main() {
	err := run()
	if err != nil {
		panic(err)
	}
}

func run() error {
	ctx := context.Background()

	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stdout))
	if err != nil {
		return err
	}

	defer client.Close()

	hostDirectory := client.Host().Directory(".", dagger.HostDirectoryOpts{
		Exclude: []string{"diagrams/", "build/", "test/"},
	})

	_, err = client.Container().From("python:3.11-bookworm").
		WithExec([]string{"pip", "install", "poetry==1.7.0"}).
		WithDirectory("/app", hostDirectory.Directory("flipt-client-python")).
		WithDirectory("/app/ext", hostDirectory.Directory("tmp")).
		WithWorkdir("/app").
		WithExec([]string{"ls", "-la", "/app/ext"}).
		WithExec([]string{"poetry", "install", "--no-dev"}).
		Sync(ctx)

	return err
}
