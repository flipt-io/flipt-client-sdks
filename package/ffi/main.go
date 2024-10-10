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
	"os/exec"
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
		"python":    pythonBuild,
		"go":        goBuild,
		"go-musl":   goMuslBuild,
		"ruby":      rubyBuild,
		"java":      javaBuild,
		"java-musl": javaMuslBuild,
		"dart":      dartBuild,
	}
	sema = make(chan struct{}, 5)
	// defaultInclude is the default include for all builds to copy over the
	// ffi files into the build directory without unecessary files
	defaultInclude = []string{"**/*.so", "**/*.dylib", "**/*.dll"}
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
	both  libc = "both"
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

	// download the pre-build libraries for each platform to local tmp directory
	// to be be used later to copy into the sdk directories
	if err := downloadFFI(ctx, client); err != nil {
		return err
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

type pkg struct {
	id     string
	target string
	ext    string
	libc   libc
}

var (
	packages = []pkg{
		{id: "Linux-arm64", target: "aarch64-unknown-linux-gnu", ext: "tar.gz", libc: glibc},
		{id: "Linux-x86_64", target: "x86_64-unknown-linux-gnu", ext: "tar.gz", libc: glibc},
		{id: "Linux-arm64-musl", target: "aarch64-unknown-linux-musl", ext: "tar.gz", libc: musl},
		{id: "Linux-x86_64-musl", target: "x86_64-unknown-linux-musl", ext: "tar.gz", libc: musl},
		{id: "Darwin-arm64", target: "aarch64-apple-darwin", ext: "tar.gz", libc: both},
		{id: "Darwin-x86_64", target: "x86_64-apple-darwin", ext: "tar.gz", libc: both},
		// {id: "Windows-x86_64", target: "x86_64-pc-windows-msvc", ext: "zip", libc: glibc},
	}
)

func args(args string, a ...any) []string {
	return strings.Split(fmt.Sprintf(args, a...), " ")
}

func getLatestEngineTag() (string, error) {
	// check if jq is installed
	if _, err := exec.LookPath("jq"); err != nil {
		return "", fmt.Errorf("jq is not installed")
	}

	// check if curl is installed
	if _, err := exec.LookPath("curl"); err != nil {
		return "", fmt.Errorf("curl is not installed")
	}

	// use github api to get the latest release of the ffi engine
	cmd := exec.Command("sh", "-c", `
		curl -s "https://api.github.com/repos/flipt-io/flipt-client-sdks/releases" | 
		jq -r '.[] | select(.tag_name | startswith("flipt-engine-ffi-")) | .tag_name' | 
		sort -Vr | 
		head -n 1 | 
		sed "s/^flipt-engine-ffi-//"
	`)

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get latest engine tag: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

func downloadFFI(ctx context.Context, client *dagger.Client) error {
	var err error
	if engineTag == "" {
		engineTag, err = getLatestEngineTag()
		if err != nil {
			return err
		}
	}

	for _, pkg := range packages {
		pkg := pkg
		var (
			out = strings.TrimSuffix(strings.ToLower(strings.ReplaceAll(pkg.id, "-", "_")), "_musl")
			url = fmt.Sprintf("https://github.com/flipt-io/flipt-client-sdks/releases/download/flipt-engine-ffi-%s/flipt-engine-ffi-%s.%s", engineTag, pkg.id, pkg.ext)
		)

		container := client.Container().From("debian:bookworm-slim").
			WithExec([]string{"apt-get", "update"}).
			WithExec([]string{"apt-get", "install", "-y", "wget", "p7zip-full"}).
			WithExec(args("mkdir -p /tmp/dl")).
			WithExec(args("wget %s -O /tmp/dl/%s.%s", url, pkg.id, pkg.ext)).
			WithExec(args("mkdir -p /tmp/%s", out)).
			WithExec(args("mkdir -p /out/glibc/%s /out/musl/%s", out, out))

		if pkg.ext == "tar.gz" {
			container = container.WithExec(args("tar xzf /tmp/dl/%s.%s -C /tmp/%s", pkg.id, pkg.ext, out))
		} else {
			container = container.WithExec(args("7z x /tmp/dl/%s.%s -o/tmp/%s", pkg.id, pkg.ext, out))
		}

		var cmd []string
		switch pkg.libc {
		case both:
			cmd = []string{"sh", "-c", fmt.Sprintf("cp -r /tmp/%s/target/%s/release/* /out/glibc/%s && cp -r /tmp/%s/target/%s/release/* /out/musl/%s", out, pkg.target, out, out, pkg.target, out)}
		case musl:
			cmd = []string{"sh", "-c", fmt.Sprintf("mv /tmp/%s/target/%s/release/* /out/musl/%s", out, pkg.target, out)}
		default: // glibc
			cmd = []string{"sh", "-c", fmt.Sprintf("mv /tmp/%s/target/%s/release/* /out/glibc/%s", out, pkg.target, out)}
		}

		dir := container.
			WithExec(cmd).
			Directory("/out")

		_, err := dir.Export(ctx, "tmp")
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
		WithDirectory("/src/ext", hostDirectory.Directory("tmp/glibc"), dagger.ContainerWithDirectoryOpts{
			Include: defaultInclude,
		}).
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
		WithFile("/tmp/ext/flipt_engine.h", hostDirectory.File("flipt-engine-ffi/include/flipt_engine.h")).
		WithDirectory("/tmp/ext", hostDirectory.Directory("tmp/"+string(buildOpts.libc)), dagger.ContainerWithDirectoryOpts{Include: defaultInclude})

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

func rubyBuild(ctx context.Context, client *dagger.Client, hostDirectory *dagger.Directory, opts ...buildOptionsFn) error {
	container := client.Container().From("ruby:3.1-bookworm").
		WithWorkdir("/src").
		WithDirectory("/src", hostDirectory.Directory("flipt-client-ruby")).
		WithDirectory("/src/lib/ext", hostDirectory.Directory("tmp/glibc"), dagger.ContainerWithDirectoryOpts{
			Include: defaultInclude,
		}).
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
	buildOpts := buildOptions{
		libc: glibc,
	}

	for _, opt := range opts {
		opt(&buildOpts)
	}

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

	type rename struct {
		old string
		new string
	}

	renames := []rename{
		{old: "linux_x86_64", new: "linux-x86-64"},
		{old: "linux_arm64", new: "linux-aarch64"},
		{old: "darwin_x86_64", new: "darwin-x86-64"},
		{old: "darwin_arm64", new: "darwin-aarch64"},
		{old: "windows_x86_64", new: "win32-x86-64"},
	}

	for _, rename := range renames {
		tmp := fmt.Sprintf("tmp/%s/%s", buildOpts.libc, rename.old)
		// if the directory does not exist, skip it
		if _, err := os.Stat(tmp); os.IsNotExist(err) {
			continue
		}

		// staging directory to copy the files into
		stg := fmt.Sprintf("staging/%s/%s", buildOpts.libc, rename.new)

		// remove directory if it exists
		if err := os.RemoveAll(stg); err != nil {
			return err
		}

		// create the directory
		if err := os.MkdirAll(stg, 0o755); err != nil {
			return err
		}

		// copy the directory to the staging directory
		if err := exec.Command("sh", "-c", fmt.Sprintf("cp -r %s/* %s/", tmp, stg)).Run(); err != nil {
			return err
		}
	}

	container := client.Container().From("gradle:8.5.0-jdk11").
		WithDirectory("/src", hostDirectory.Directory("flipt-client-java")).
		WithDirectory("/src/src/main/resources", hostDirectory.Directory("staging/"+string(buildOpts.libc)), dagger.ContainerWithDirectoryOpts{
			Include: defaultInclude,
		}).
		WithWorkdir("/src")

	if buildOpts.libc == glibc {
		container = container.WithExec([]string{"gradle", "-x", "test", "build"})
	} else {
		container = container.
			WithExec([]string{"gradle", "-x", "test", "--build-file", "build.musl.gradle", "build"})
	}

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

	container = container.WithSecretVariable("MAVEN_USERNAME", mavenUsername).
		WithSecretVariable("MAVEN_PASSWORD", mavenPassword).
		WithSecretVariable("MAVEN_PUBLISH_REGISTRY_URL", mavenRegistryUrl).
		WithSecretVariable("PGP_PRIVATE_KEY", pgpPrivateKey).
		WithSecretVariable("PGP_PASSPHRASE", pgpPassphrase)

	if buildOpts.libc == glibc {
		_, err = container.WithExec([]string{"gradle", "publish"}).Sync(ctx)
	} else {
		_, err = container.WithExec([]string{"gradle", "-b", "build.musl.gradle", "publish"}).Sync(ctx)
	}

	return err
}

func javaMuslBuild(ctx context.Context, client *dagger.Client, hostDirectory *dagger.Directory, opts ...buildOptionsFn) error {
	return javaBuild(ctx, client, hostDirectory, append(opts, withMusl())...)
}

func dartBuild(ctx context.Context, client *dagger.Client, hostDirectory *dagger.Directory, opts ...buildOptionsFn) error {
	container := client.Container().From("dart:stable").
		WithDirectory("/src", hostDirectory.Directory("flipt-client-dart"), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{".gitignore", ".dart_tool/"},
		}).
		WithDirectory("/src/lib/src/ffi", hostDirectory.Directory("tmp/glibc"), dagger.ContainerWithDirectoryOpts{
			Include: defaultInclude,
		}).
		WithWorkdir("/src").
		WithExec([]string{"dart", "pub", "get"})

	var err error

	if !push {
		_, err = container.WithExec([]string{"dart", "pub", "publish", "--dry-run"}).Sync(ctx)
		return err
	}

	// get oidc token for publishing to pub.dev
	// https://dart.dev/tools/pub/automated-publishing#publishing-packages-using-github-actions
	if os.Getenv("PUB_TOKEN") == "" {
		return fmt.Errorf("PUB_TOKEN is not set")
	}

	pubToken := client.SetSecret("pub-token", os.Getenv("PUB_TOKEN"))

	_, err = container.WithSecretVariable("PUB_TOKEN", pubToken).
		WithExec([]string{"dart", "pub", "token", "add", "https://pub.dev", "--env-var", "PUB_TOKEN"}).
		WithExec([]string{"dart", "pub", "publish", "--force"}).
		Sync(ctx)

	return err
}
