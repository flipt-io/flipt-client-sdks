package main

import (
	"context"
	"encoding/base64"
	"errors"
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
	tag          string
	languageToFn = map[string]buildFn{
		"python": pythonBuild,
		"go":     goBuild,
		"node":   nodeBuild,
		"ruby":   rubyBuild,
	}
	sema = make(chan struct{}, 5)
)

func init() {
	flag.StringVar(&languages, "languages", "", "comma separated list of which language(s) to run builds for")
	flag.BoolVar(&push, "push", false, "push built artifacts to registry")
	flag.StringVar(&tag, "tag", "", "tag to use for release")
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
	container := client.Container().From("python:3.11-bookworm").
		WithExec([]string{"pip", "install", "poetry==1.7.0"}).
		WithDirectory("/src", hostDirectory.Directory("flipt-client-python")).
		WithDirectory("/src/ext", hostDirectory.Directory("tmp")).
		WithWorkdir("/src").
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

	_, err = container.WithSecretVariable("POETRY_PYPI_TOKEN_PYPI", pypiAPIKeySecret).
		WithExec([]string{"poetry", "publish", "-v"}).
		Sync(ctx)

	return err
}

func goBuild(ctx context.Context, client *dagger.Client, hostDirectory *dagger.Directory) error {
	if tag == "" {
		return fmt.Errorf("tag is not set")
	}

	// because of how Go modules work, we need to create a new repo that contains
	// only the go client code. This is because the go client code is in a subdirectory
	// we also need to copy the ext directory into the tag of this new repo so that it can be used
	targetRepo := os.Getenv("TARGET_REPO")
	if targetRepo == "" {
		targetRepo = "https://github.com/flipt-io/flipt-client-go.git"
	}

	targetTag := strings.TrimPrefix(tag, "refs/tags/flipt-client-go-")

	pat := os.Getenv("GITHUB_TOKEN")
	if pat == "" {
		return errors.New("GITHUB_TOKEN environment variable must be set")
	}

	encodedPAT := base64.URLEncoding.EncodeToString([]byte("pat:" + pat))

	var gitUserName = os.Getenv("GIT_USER_NAME")
	if gitUserName == "" {
		gitUserName = "flipt-bot"
	}

	var gitUserEmail = os.Getenv("GIT_USER_EMAIL")
	if gitUserEmail == "" {
		gitUserEmail = "dev@flipt.io"
	}

	git := client.Container().From("golang:1.21.3-bookworm").
		WithExec([]string{"git", "config", "--global", "user.email", gitUserEmail}).
		WithExec([]string{"git", "config", "--global", "user.name", gitUserName}).
		WithExec([]string{"git", "config", "--global",
			"http.https://github.com/.extraheader",
			fmt.Sprintf("AUTHORIZATION: Basic %s", encodedPAT)})

	repository := git.
		WithExec([]string{"git", "clone", "https://github.com/flipt-io/flipt-client-sdks.git", "/src"}).
		WithWorkdir("/src").
		WithDirectory("/tmp/ext", hostDirectory.Directory("tmp"))

	filtered := repository.
		WithEnvVariable("FILTER_BRANCH_SQUELCH_WARNING", "1").
		WithExec([]string{"git", "filter-branch", "-f", "--prune-empty",
			"--subdirectory-filter", "flipt-client-go",
			"--index-filter", "cp -r /tmp/ext .",
			"--", tag})

	if !push {
		_, err := filtered.Sync(ctx)
		return err
	}

	// push to target repo/tag
	_, err := filtered.WithExec([]string{
		"git",
		"push",
		"-f",
		targetRepo,
		fmt.Sprintf("%s:%s", tag, targetTag)}).
		Sync(ctx)

	return err
}

func nodeBuild(ctx context.Context, client *dagger.Client, hostDirectory *dagger.Directory) error {

	container := client.Container().From("node:21.2-bookworm").
		WithDirectory("/src", hostDirectory.Directory("flipt-client-node"), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{"./node_modules/"},
		}).
		WithDirectory("/src/ext", hostDirectory.Directory("tmp")).
		WithWorkdir("/src").
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
		WithWorkdir("/src").
		WithDirectory("/src", hostDirectory.Directory("flipt-client-ruby")).
		WithDirectory("/src/lib/ext", hostDirectory.Directory("tmp")).
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
