package main

import (
	"context"
	"os"

	"dagger.io/dagger"
	"golang.org/x/sync/errgroup"
)

var sema = make(chan struct{}, 5)

func main() {
	err := run()
	if err != nil {
		panic(err)
	}
}

type buildFn func(context.Context, *dagger.Client, *dagger.Directory) error

func run() error {
	ctx := context.Background()

	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stdout))
	if err != nil {
		return err
	}

	defer client.Close()

	dir := client.Host().Directory(".", dagger.HostDirectoryOpts{
		Exclude: []string{"diagrams/", "build/", "test/"},
	})

	var g errgroup.Group

	for _, fn := range []buildFn{
		buildPython,
	} {
		fn := fn
		g.Go(take(func() error {
			return fn(ctx, client, dir)
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

func buildPython(ctx context.Context, client *dagger.Client, hostDirectory *dagger.Directory) error {
	_, err := client.Container().From("python:3.11-bookworm").
		WithExec([]string{"pip", "install", "poetry==1.7.0"}).
		WithDirectory("/app", hostDirectory.Directory("flipt-client-python")).
		WithDirectory("/app/ext", hostDirectory.Directory("tmp")).
		WithWorkdir("/app").
		WithExec([]string{"poetry", "config", "installer.parallel", "false"}).
		WithExec([]string{"poetry", "install", "-v"}).
		WithExec([]string{"poetry", "build", "-v"}).
		Sync(ctx)

	return err
}
