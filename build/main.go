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
		pythonBuild,
		goBuild,
		nodeBuild,
		rubyBuild,
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

func pythonBuild(ctx context.Context, client *dagger.Client, hostDirectory *dagger.Directory) error {
	_, err := client.Container().From("python:3.11-bookworm").
		WithExec([]string{"pip", "install", "poetry==1.7.0"}).
		WithDirectory("/app", hostDirectory.Directory("flipt-client-python")).
		WithDirectory("/app/ext", hostDirectory.Directory("tmp")).
		WithWorkdir("/app").
		WithExec([]string{"poetry", "install", "--without=dev", "-v"}).
		WithExec([]string{"poetry", "build", "-v"}).
		Sync(ctx)

	return err
}

func goBuild(ctx context.Context, client *dagger.Client, hostDirectory *dagger.Directory) error {
	_, err := client.Container().From("golang:1.21.3-bookworm").
		WithDirectory("/app", hostDirectory.Directory("flipt-client-go")).
		WithDirectory("/app/ext", hostDirectory.Directory("tmp")).
		WithWorkdir("/app").
		WithExec([]string{"go", "build", "-v", "."}).
		Sync(ctx)

	return err
}

func nodeBuild(ctx context.Context, client *dagger.Client, hostDirectory *dagger.Directory) error {
	_, err := client.Container().From("node:21.2-bookworm").
		WithDirectory("/app", hostDirectory.Directory("flipt-client-node"), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{"./node_modules/"},
		}).
		WithDirectory("/app/ext", hostDirectory.Directory("tmp")).
		WithWorkdir("/app").
		WithExec([]string{"npm", "install"}).
		WithExec([]string{"npm", "run", "build"}).
		WithExec([]string{"npm", "pack"}).
		Sync(ctx)

	return err
}

func rubyBuild(ctx context.Context, client *dagger.Client, hostDirectory *dagger.Directory) error {
	_, err := client.Container().From("ruby:3.1-bookworm").
		WithDirectory("/app", hostDirectory.Directory("flipt-client-ruby")).
		WithDirectory("/app/lib/ext", hostDirectory.Directory("tmp")).
		WithWorkdir("/app").
		WithExec([]string{"bundle", "install"}).
		WithExec([]string{"bundle", "exec", "rake", "build"}).
		Sync(ctx)

	return err
}
