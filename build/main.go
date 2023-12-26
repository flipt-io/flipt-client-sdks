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
	secure       bool
	protocol     string = "https"
	version      string
	languageToFn = map[string]buildFn{
		"python": pythonBuild,
		//"go":     goBuild,
		"node": nodeBuild,
		"ruby": rubyBuild,
	}
	sema = make(chan struct{}, 5)
)

func init() {
	flag.StringVar(&languages, "languages", "", "comma separated list of which language(s) to run builds for")
	flag.BoolVar(&push, "push", false, "push built artifacts to registry")
	flag.BoolVar(&secure, "secure", true, "use secure protocol (https) to push artifacts")
	flag.StringVar(&version, "version", "", "version to build")
}

func main() {
	flag.Parse()

	if !secure {
		protocol = "http"
	}

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
	container := client.Container().From("python:3.11-bookworm").
		WithExec([]string{"pip", "install", "poetry==1.7.0"}).
		WithDirectory("/app", hostDirectory.Directory("flipt-client-python")).
		WithDirectory("/app/ext", hostDirectory.Directory("tmp")).
		WithWorkdir("/app").
		WithExec([]string{"poetry", "install", "--without=dev", "-v"}).
		WithExec([]string{"poetry", "build", "-v"})

	var err error

	if !push {
		_, err = container.Sync(ctx)
		return err
	}

	if os.Getenv("PYPI_API_KEY") == "" {
		return fmt.Errorf("PYPI_API_KEY is not set")
	}

	pypiAPIKeySecret := client.SetSecret("pypi-api-key", os.Getenv("PYPI_API_KEY"))

	_, err = container.WithEnvVariable("POETRY_HTTP_BASIC_PUBLISH_USERNAME", "__token__").
		WithSecretVariable("POETRY_HTTP_BASIC_PUBLISH_PASSWORD", pypiAPIKeySecret).
		WithExec([]string{"poetry", "publish", "-v"}).
		Sync(ctx)

	return err
}

func goBuild(ctx context.Context, client *dagger.Client, hostDirectory *dagger.Directory) error {
	if version == "" {
		return fmt.Errorf("version is not set")
	}

	container := client.Container().From("golang:1.21.3-bookworm").
		WithDirectory("/app", hostDirectory.Directory("flipt-client-go")).
		WithDirectory("/app/ext", hostDirectory.Directory("tmp")).
		WithWorkdir("/app").
		WithExec([]string{"go", "install", "-v", "github.com/cloudsmith-io/gopack@latest"}).
		WithExec([]string{"gopack", version, "/app"})

	// TODO: push
	_, err := container.Sync(ctx)
	return err
}

func nodeBuild(ctx context.Context, client *dagger.Client, hostDirectory *dagger.Directory) error {

	container := client.Container().From("node:21.2-bookworm").
		WithDirectory("/app", hostDirectory.Directory("flipt-client-node"), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{"./node_modules/"},
		}).
		WithDirectory("/app/ext", hostDirectory.Directory("tmp")).
		WithWorkdir("/app").
		WithExec([]string{"npm", "install"}).
		WithExec([]string{"npm", "run", "build"}).
		WithExec([]string{"npm", "pack"})

	var err error

	if !push {
		_, err = container.Sync(ctx)
		return err
	}

	if os.Getenv("NPM_API_KEY") == "" {
		return fmt.Errorf("NPM_API_KEY is not set")
	}

	npmAPIKeySecret := client.SetSecret("npm-api-key", os.Getenv("NPM_API_KEY"))

	_, err = container.WithSecretVariable("NPM_TOKEN", npmAPIKeySecret).
		WithExec([]string{"npm", "config", "set", "--", "//registry.npmjs.org/:_authToken", "${NPM_TOKEN}"}).
		WithExec([]string{"npm", "publish", "--access", "public"}).
		Sync(ctx)

	return err
}

func rubyBuild(ctx context.Context, client *dagger.Client, hostDirectory *dagger.Directory) error {

	container := client.Container().From("ruby:3.1-bookworm").
		WithWorkdir("/app").
		WithDirectory("/app", hostDirectory.Directory("flipt-client-ruby")).
		WithDirectory("/app/lib/ext", hostDirectory.Directory("tmp")).
		WithExec([]string{"bundle", "install"}).
		WithExec([]string{"bundle", "exec", "rake", "build"})

	var err error

	if !push {
		_, err = container.Sync(ctx)
		return err
	}

	if os.Getenv("RUBYGEMS_API_KEY") == "" {
		return fmt.Errorf("RUBYGEMS_API_KEY is not set")
	}

	gemHostAPIKeySecret := client.SetSecret("rubygems-api-key", os.Getenv("RUBYGEMS_API_KEY"))

	_, err = container.
		WithSecretVariable("GEM_HOST_API_KEY", gemHostAPIKeySecret).
		WithExec([]string{"sh", "-c", "gem push pkg/flipt_client-*.gem"}).
		Sync(ctx)

	return err
}
