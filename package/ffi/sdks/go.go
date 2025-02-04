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

func (s *GoSDK) SupportedPlatforms() []platform.Platform {
	switch s.Libc {
	case platform.Musl:
		return []platform.Platform{
			platform.LinuxArm64Musl,
			platform.LinuxX86_64Musl,
			platform.DarwinArm64,
			platform.DarwinX86_64,
			platform.WindowsX86_64,
		}
	default:
		return s.BaseSDK.SupportedPlatforms()
	}
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

	repository := git.
		WithExec([]string{"git", "clone", "https://github.com/flipt-io/flipt-client-sdks.git", "/src"}).
		WithWorkdir("/src").
		WithFile("/tmp/ext/flipt_engine.h", hostDirectory.File("flipt-engine-ffi/include/flipt_engine.h")).
		WithDirectory("/tmp/ext", hostDirectory.Directory(fmt.Sprintf("tmp/%s", s.Libc)), dagger.ContainerWithDirectoryOpts{Include: defaultInclude})

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

	targetRepo := os.Getenv("TARGET_REPO")
	if targetRepo == "" {
		targetRepo = "https://github.com/flipt-io/flipt-client-go.git"
	}

	targetTag := strings.TrimPrefix(opts.Tag, tagPrefix)

	// push to target repo/tag
	if _, err := filtered.WithExec([]string{
		"git",
		"push",
		"-f",
		targetRepo,
		fmt.Sprintf("%s:%s", opts.Tag, targetTag)}).
		Sync(ctx); err != nil {
		return err
	}

	// create GitHub release via API
	releasePayload := fmt.Sprintf(`{"tag_name":"%s","name":"%s","body":"Release %s of the Flipt Go Client SDK"}`, targetTag, targetTag, targetTag)
	_, err = filtered.WithExec([]string{
		"curl",
		"-X", "POST",
		"-H", fmt.Sprintf("Authorization: token %s", pat),
		"-H", "Accept: application/vnd.github.v3+json",
		"-d", releasePayload,
		fmt.Sprintf("https://api.github.com/repos/flipt-io/flipt-client-go/releases"),
	}).Sync(ctx)

	return err
}
