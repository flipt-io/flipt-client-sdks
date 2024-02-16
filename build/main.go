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
		"java":   javaBuild,
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
		Exclude: []string{".github/", "build/", "test/", ".git/"},
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

	const tagPrefix = "refs/tags/flipt-client-go-"
	if !strings.HasPrefix(tag, tagPrefix) {
		return fmt.Errorf("tag %q must start with %q", tag, tagPrefix)
	}

	// because of how Go modules work, we need to create a new repo that contains
	// only the go client code. This is because the go client code is in a subdirectory
	// we also need to copy the ext directory into the tag of this new repo so that it can be used
	targetRepo := os.Getenv("TARGET_REPO")
	if targetRepo == "" {
		targetRepo = "https://github.com/flipt-io/flipt-client-go.git"
	}

	targetTag := strings.TrimPrefix(tag, tagPrefix)

	pat := os.Getenv("GITHUB_TOKEN")
	if pat == "" {
		return errors.New("GITHUB_TOKEN environment variable must be set")
	}

	var (
		encodedPAT = base64.URLEncoding.EncodeToString([]byte("pat:" + pat))
		ghToken    = client.SetSecret("gh-token", encodedPAT)
	)

	var gitUserName = os.Getenv("GIT_USER_NAME")
	if gitUserName == "" {
		gitUserName = "flipt-bot"
	}

	var gitUserEmail = os.Getenv("GIT_USER_EMAIL")
	if gitUserEmail == "" {
		gitUserEmail = "dev@flipt.io"
	}

	git := client.Container().From("golang:1.21.3-bookworm").
		WithSecretVariable("GITHUB_TOKEN", ghToken).
		WithExec([]string{"git", "config", "--global", "user.email", gitUserEmail}).
		WithExec([]string{"git", "config", "--global", "user.name", gitUserName}).
		WithExec([]string{"sh", "-c", `git config --global http.https://github.com/.extraheader "AUTHORIZATION: Basic ${GITHUB_TOKEN}"`})

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

func javaBuild(ctx context.Context, client *dagger.Client, hostDirectory *dagger.Directory) error {
	// the directory structure of the tmp directory is as follows:
	// tmp/linux_x86_64/
	// tmp/linux_arm64/
	// tmp/darwin_arm64/

	// we need to convert it to the following structure:
	// tmp/linux-x86-64/
	// tmp/linux-aarch64/
	// tmp/darwin-aarch64/

	// this is because JNA expects the library to be in a specific directory structure based on host OS and architecture
	// see: https://github.com/java-native-access/jna/blob/master/test/com/sun/jna/PlatformTest.java
	// we can do this on the host and then copy the files into the container

	if err := os.Rename("tmp/linux_x86_64", "tmp/linux-x86-64"); err != nil {
		return err
	}

	if err := os.Rename("tmp/linux_arm64", "tmp/linux-aarch64"); err != nil {
		return err
	}

	if err := os.Rename("tmp/darwin_arm64", "tmp/darwin-aarch64"); err != nil {
		return err
	}

	container := client.Container().From("gradle:8.5.0-jdk11").
		WithDirectory("/src", hostDirectory.Directory("flipt-client-java")).
		WithDirectory("/src/src/main/resources", hostDirectory.Directory("tmp")).
		WithWorkdir("/src").
		WithExec([]string{"gradle", "-x", "test", "build"})

	var err error

	if !push {
		_, err = container.Sync(ctx)
		return err
	}

	if os.Getenv("MAVEN_USERNAME") == "" {
		return fmt.Errorf("MAVEN_USERNAME is not set")
	}
	if os.Getenv("MAVEN_PASSWORD") == "" {
		return fmt.Errorf("MAVEN_PASSWORD is not set")
	}
	if os.Getenv("MAVEN_PUBLISH_REGISTRY_URL") == "" {
		return fmt.Errorf("MAVEN_PUBLISH_REGISTRY_URL is not set")
	}
	if os.Getenv("PGP_PRIVATE_KEY") == "" {
		return fmt.Errorf("PGP_PRIVATE_KEY is not set")
	}
	if os.Getenv("PGP_PASSPHRASE") == "" {
		return fmt.Errorf("PGP_PASSPHRASE is not set")
	}

	var (
		mavenUsername    = client.SetSecret("maven-username", os.Getenv("MAVEN_USERNAME"))
		mavenPassword    = client.SetSecret("maven-password", os.Getenv("MAVEN_PASSWORD"))
		mavenRegistryUrl = client.SetSecret("maven-registry-url", os.Getenv("MAVEN_PUBLISH_REGISTRY_URL"))
		pgpPrivateKey    = client.SetSecret("pgp-private-key", os.Getenv("PGP_PRIVATE_KEY"))
		pgpPassphrase    = client.SetSecret("pgp-passphrase", os.Getenv("PGP_PASSPHRASE"))
	)

	_, err = container.WithSecretVariable("MAVEN_USERNAME", mavenUsername).
		WithSecretVariable("MAVEN_PASSWORD", mavenPassword).
		WithSecretVariable("MAVEN_PUBLISH_REGISTRY_URL", mavenRegistryUrl).
		WithSecretVariable("PGP_PRIVATE_KEY", pgpPrivateKey).
		WithSecretVariable("PGP_PASSPHRASE", pgpPassphrase).
		WithExec([]string{"gradle", "publish"}).
		Sync(ctx)

	return err
}
