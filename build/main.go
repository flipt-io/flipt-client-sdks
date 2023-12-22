package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"maps"
	"os"
	"strings"

	"dagger.io/dagger"
	"golang.org/x/sync/errgroup"
)

var (
	languages    string
	push         bool
	languageToFn = map[string]buildFn{
		"python": pythonBuild,
		"go":     goBuild,
		"node":   nodeBuild,
		"ruby":   rubyBuild,
	}
	sema = make(chan struct{}, 5)
)

func init() {
	flag.StringVar(&languages, "languages", "", "comma separated list of which language(s) to run integration tests for")
	flag.BoolVar(&push, "push", false, "push built artifacts to registry")
}

func main() {
	flag.Parse()

	if err := run(); err != nil {
		log.Fatal(err)
	}
}

type buildFn func(context.Context, *dagger.Client, *dagger.Directory) error

func run() error {
	var tests = make(map[string]buildFn, len(languageToFn))

	maps.Copy(tests, languageToFn)

	if languages != "" {
		l := strings.Split(languages, ",")
		subset := make(map[string]buildFn, len(l))
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
		Exclude: []string{"diagrams/", "build/", "test/", ".git/"},
	})

	var g errgroup.Group

	for _, fn := range tests {
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
	gemAPIKey := os.Getenv("RUBYGEMS_API_KEY")
	if gemAPIKey == "" {
		return fmt.Errorf("RUBYGEMS_API_KEY is not set")
	}

	gemHost := os.Getenv("RUBYGEMS_HOST")
	if gemHost == "" {
		// TODO: relax this when we push to rubygems.org
		return fmt.Errorf("RUBYGEMS_HOST is not set")
	}

	gemHostAPIKeySecret := client.SetSecret("rubygems-api-key", gemAPIKey)

	container := client.Container().From("ruby:3.1-bookworm").
		WithWorkdir("/app").
		WithDirectory("/app", hostDirectory.Directory("flipt-client-ruby")).
		WithDirectory("/app/lib/ext", hostDirectory.Directory("tmp")).
		WithExec([]string{"bundle", "install"}).
		WithExec([]string{"bundle", "exec", "rake", "build"})

	var err error

	if !push {
		_, err = container.File("/app/pkg/flipt_client-0.1.0.gem").
			Export(ctx, "tmp/flipt_client-0.1.0.gem")
		return err
	}

	_, err = container.
		WithSecretVariable("GEM_HOST_API_KEY", gemHostAPIKeySecret).
		WithExec([]string{"gem", "push", "--host", gemHost, "/app/pkg/flipt_client-0.1.0.gem"}).
		Sync(ctx)

	return err
}
