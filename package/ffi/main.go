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
	sdks      string
	push      bool
	tag       string
	engineTag string
	sdksToFn  = map[string]buildFn{
		"python":  pythonBuild,
		"go":      goBuild,
		"go-musl": goMuslBuild,
		"node":    nodeBuild,
		"ruby":    rubyBuild,
		"java":    javaBuild,
	}
	sema = make(chan struct{}, 5)
)

func init() {
	flag.StringVar(&sdks, "sdks", "", "comma separated list of which sdks(s) to run builds for")
	flag.BoolVar(&push, "push", false, "push built artifacts to registry")
	flag.StringVar(&tag, "tag", "", "tag to use for release")
	flag.StringVar(&engineTag, "engine-tag", "", "tag to use for the engine ffi")
}

func main() {
	flag.Parse()

	if err := run(); err != nil {
		log.Fatal(err)
	}
}

type libc string

const (
	glibc libc = "glibc"
	musl  libc = "musl"
)

type buildOptions struct {
	libc libc
}

type buildOptionsFn func(*buildOptions)

func withMusl() buildOptionsFn {
	return func(opts *buildOptions) {
		opts.libc = musl
	}
}

type buildFn func(ctx context.Context, client *dagger.Client, hostDirectory *dagger.Directory, opts ...buildOptionsFn) error

func run() error {
	var builds = make(map[string]buildFn, len(sdksToFn))

	maps.Copy(builds, sdksToFn)

	if sdks != "" {
		l := strings.Split(sdks, ",")
		subset := make(map[string]buildFn, len(l))
		for _, sdk := range l {
			testFn, ok := sdksToFn[sdk]
			if !ok {
				return fmt.Errorf("sdk %s is not supported", sdk)
			}
			subset[sdk] = testFn
		}

		builds = subset
	}

	ctx := context.Background()

	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stdout))
	if err != nil {
		return err
	}

	defer client.Close()

	if engineTag != "" {
		// download the pre-build libraries for each platform to local tmp directory
		// to be be used later to copy into the sdk directories
		if err := downloadFFI(ctx, client); err != nil {
			return err
		}
	}

	dir := client.Host().Directory(".", dagger.HostDirectoryOpts{
		Exclude: []string{".github/", "package/", "test/", ".git/"},
	})

	var g errgroup.Group

	for _, fn := range builds {
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

var (
	packages = map[string]string{
		"Linux-arm64":       "aarch64-unknown-linux-gnu",
		"Linux-x86_64":      "x86_64-unknown-linux-gnu",
		"Darwin-arm64":      "aarch64-apple-darwin",
		"Darwin-x86_64":     "x86_64-apple-darwin",
		"Linux-arm64-musl":  "aarch64-unknown-linux-musl",
		"Linux-x86_64-musl": "x86_64-unknown-linux-musl",
	}
)

func args(args string, a ...any) []string {
	return strings.Split(fmt.Sprintf(args, a...), " ")
}

func downloadFFI(ctx context.Context, client *dagger.Client) error {
	if engineTag == "" {
		return fmt.Errorf("ENGINE_TAG is not set")
	}

	for pkg, target := range packages {
		var (
			out = strings.ToLower(strings.ReplaceAll(pkg, "-", "_"))
			url = fmt.Sprintf("https://github.com/flipt-io/flipt-client-sdks/releases/download/flipt-engine-ffi-%s/flipt-engine-ffi-%s.tar.gz", engineTag, pkg)
		)

		container := client.Container().From("debian:bookworm-slim").
			WithExec([]string{"apt-get", "update"}).
			WithExec([]string{"apt-get", "install", "-y", "wget"}).
			WithExec([]string{"mkdir", "-p", "/tmp/dl"}).
			WithExec(args("wget %s -O /tmp/dl/%s.tar.gz", url, pkg)).
			WithExec(args("mkdir -p /tmp/%s", out)).
			WithExec(args("mkdir -p /out/%s", out)).
			WithExec(args("tar -xzf /tmp/dl/%s.tar.gz -C /tmp/%s", pkg, out)).
			WithExec([]string{"sh", "-c", "mv /tmp/" + out + "/target/" + target + "/release/* /out/" + out}).
			Directory("/out")

		_, err := container.Export(ctx, "./tmp/")
		if err != nil {
			return err
		}
	}

	return nil
}

func pythonBuild(ctx context.Context, client *dagger.Client, hostDirectory *dagger.Directory, opts ...buildOptionsFn) error {
	container := client.Container().From("python:3.11-bookworm").
		WithExec([]string{"pip", "install", "poetry==1.7.0"}).
		WithDirectory("/src", hostDirectory.Directory("flipt-client-python")).
		WithDirectory("/src/ext", hostDirectory.Directory("tmp")).
		WithFile("/src/ext/flipt_engine.h", hostDirectory.File("flipt-engine-ffi/include/flipt_engine.h")).
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

func goBuild(ctx context.Context, client *dagger.Client, hostDirectory *dagger.Directory, opts ...buildOptionsFn) error {
	buildOpts := buildOptions{
		libc: glibc,
	}

	for _, opt := range opts {
		opt(&buildOpts)
	}

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
		WithFile("/tmp/ext/flipt_engine.h", hostDirectory.File("flipt-engine-ffi/include/flipt_engine.h"))

	defaultExclude := []string{"*.rlib"}

	if buildOpts.libc == glibc {
		exclude := append(defaultExclude, "*_musl")
		repository = repository.
			WithDirectory("/tmp/ext", hostDirectory.Directory("tmp"), dagger.ContainerWithDirectoryOpts{Exclude: exclude})
	} else {
		repository = repository.
			WithDirectory("/tmp/ext/linux_arm64", hostDirectory.Directory("tmp/linux_arm64_musl"), dagger.ContainerWithDirectoryOpts{Exclude: defaultExclude}).
			WithDirectory("/tmp/ext/linux_x86_64", hostDirectory.Directory("tmp/linux_x86_64_musl"), dagger.ContainerWithDirectoryOpts{Exclude: defaultExclude})
	}

	filtered := repository.
		WithEnvVariable("FILTER_BRANCH_SQUELCH_WARNING", "1").
		WithExec([]string{"git", "filter-branch", "-f", "--prune-empty",
			"--subdirectory-filter", "flipt-client-go",
			"--tree-filter", "cp -r /tmp/ext .",
			"--", tag})

	_, err := filtered.Sync(ctx)
	if !push {
		return err
	}

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

	// push to target repo/tag
	_, err = filtered.WithExec([]string{
		"git",
		"push",
		"-f",
		targetRepo,
		fmt.Sprintf("%s:%s", tag, targetTag)}).
		Sync(ctx)

	return err
}

func goMuslBuild(ctx context.Context, client *dagger.Client, hostDirectory *dagger.Directory, opts ...buildOptionsFn) error {
	return goBuild(ctx, client, hostDirectory, append(opts, withMusl())...)
}

func nodeBuild(ctx context.Context, client *dagger.Client, hostDirectory *dagger.Directory, opts ...buildOptionsFn) error {
	container := client.Container().From("node:21.2-bookworm").
		WithDirectory("/src", hostDirectory.Directory("flipt-client-node"), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{"./node_modules/"},
		}).
		WithDirectory("/src/ext", hostDirectory.Directory("tmp")).
		WithFile("/src/ext/flipt_engine.h", hostDirectory.File("flipt-engine-ffi/include/flipt_engine.h")).
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

func rubyBuild(ctx context.Context, client *dagger.Client, hostDirectory *dagger.Directory, opts ...buildOptionsFn) error {
	container := client.Container().From("ruby:3.1-bookworm").
		WithWorkdir("/src").
		WithDirectory("/src", hostDirectory.Directory("flipt-client-ruby")).
		WithDirectory("/src/lib/ext", hostDirectory.Directory("tmp")).
		WithFile("/src/lib/ext/flipt_engine.h", hostDirectory.File("flipt-engine-ffi/include/flipt_engine.h")).
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

func javaBuild(ctx context.Context, client *dagger.Client, hostDirectory *dagger.Directory, opts ...buildOptionsFn) error {
	// the directory structure of the tmp directory is as follows:
	// tmp/linux_x86_64/
	// tmp/linux_arm64/
	// tmp/darwin_x86_64/
	// tmp/darwin_arm64/

	// we need to convert it to the following structure:
	// tmp/linux-x86-64/
	// tmp/linux-arm/
	// tmp/darwin-x86-64/
	// tmp/darwin-aarch64/

	// this is because JNA expects the library to be in a specific directory structure based on host OS and architecture
	// see: https://github.com/java-native-access/jna/blob/master/test/com/sun/jna/PlatformTest.java
	// we can do this on the host and then copy the files into the container

	rename := map[string]string{
		"linux_x86_64":  "linux-x86-64",
		"linux_arm64":   "linux-arm",
		"darwin_x86_64": "darwin-x86-64",
		"darwin_arm64":  "darwin-aarch64",
	}

	for old, new := range rename {
		if err := os.Rename(fmt.Sprintf("tmp/%s", old), fmt.Sprintf("tmp/%s", new)); err != nil {
			return err
		}
	}

	container := client.Container().From("gradle:8.5.0-jdk11").
		WithDirectory("/src", hostDirectory.Directory("flipt-client-java")).
		WithDirectory("/src/src/main/resources", hostDirectory.Directory("tmp")).
		WithFile("/src/main/resources/flipt_engine.h", hostDirectory.File("flipt-engine-ffi/include/flipt_engine.h")).
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
