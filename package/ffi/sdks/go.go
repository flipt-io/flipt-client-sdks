package sdks

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"

	"dagger.io/dagger"
	"go.flipt.io/flipt/client-sdks/package/ffi/platform"
)

type GoSDK struct {
	BaseSDK
	Libc platform.Libc
}

func (s *GoSDK) Build(ctx context.Context, client *dagger.Client, hostDirectory *dagger.Directory, opts BuildOpts) error {

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

	libc := platform.Glibc
	if s.Libc != "" {
		libc = s.Libc
	}

	repository := git.
		WithExec([]string{"git", "clone", "https://github.com/flipt-io/flipt-client-sdks.git", "/src"}).
		WithWorkdir("/src").
		WithFile("/tmp/ext/flipt_engine.h", hostDirectory.File("flipt-engine-ffi/include/flipt_engine.h")).
		WithDirectory("/tmp/ext", hostDirectory.Directory(fmt.Sprintf("tmp/%s", libc)), dagger.ContainerWithDirectoryOpts{Include: defaultInclude})

	filtered := repository.
		WithEnvVariable("FILTER_BRANCH_SQUELCH_WARNING", "1").
		WithExec([]string{"git", "filter-branch", "-f", "--prune-empty",
			"--subdirectory-filter", "flipt-client-go",
			"--tree-filter", "cp -r /tmp/ext .",
			"--", opts.Tag})

	_, err := filtered.Sync(ctx)
	if !opts.Push {
		return err
	}

	if opts.Tag == "" {
		return fmt.Errorf("tag is not set")
	}

	const tagPrefix = "refs/tags/flipt-client-go-"
	if !strings.HasPrefix(opts.Tag, tagPrefix) {
		return fmt.Errorf("tag %q must start with %q", opts.Tag, tagPrefix)
	}

	// because of how Go modules work, we need to create a new repo that contains
	// only the go client code. This is because the go client code is in a subdirectory
	// we also need to copy the ext directory into the tag of this new repo so that it can be used
	targetRepo := os.Getenv("TARGET_REPO")
	if targetRepo == "" {
		targetRepo = "https://github.com/flipt-io/flipt-client-go.git"
	}

	targetTag := strings.TrimPrefix(opts.Tag, tagPrefix)

	// push to target repo/tag
	_, err = filtered.WithExec([]string{
		"git",
		"push",
		"-f",
		targetRepo,
		fmt.Sprintf("%s:%s", opts.Tag, targetTag)}).
		Sync(ctx)

	return err
}
