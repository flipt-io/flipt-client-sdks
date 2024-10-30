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

type SwiftSDK struct {
	BaseSDK
}

func (s *SwiftSDK) SupportedPlatforms() []platform.Platform {
	return []platform.Platform{
		platform.IOSArm64,
		platform.IOSSimArm64,
	}
}

func (s *SwiftSDK) Build(ctx context.Context, client *dagger.Client, hostDirectory *dagger.Directory, opts BuildOpts) error {
	pat := os.Getenv("GITHUB_TOKEN")
	if pat == "" && opts.Push {
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
		WithFile("Sources/FliptEngineFFI.xcframework/ios-arm64/Headers/flipt_engine.h", hostDirectory.File("flipt-engine-ffi/include/flipt_engine.h")).
		WithFile("Sources/FliptEngineFFI.xcframework/ios-arm64-simulator/Headers/flipt_engine.h", hostDirectory.File("flipt-engine-ffi/include/flipt_engine.h")).
		WithFile("Sources/FliptEngineFFI.xcframework/ios-arm64/libfliptengine.a", hostDirectory.File("tmp/glibc/ios_arm64/libfliptengine.a")).
		WithFile("Sources/FliptEngineFFI.xcframework/ios-arm64-simulator/libfliptengine.a", hostDirectory.File("tmp/glibc/ios_arm64_sim/libfliptengine.a"))

	filtered := repository.
		WithEnvVariable("FILTER_BRANCH_SQUELCH_WARNING", "1").
		WithExec([]string{"git", "filter-branch", "-f", "--prune-empty",
			"--subdirectory-filter", "flipt-client-swift",
			"--tree-filter", "cp -r tmp/ext .",
			"--", opts.Tag})

	_, err := filtered.Sync(ctx)
	if !opts.Push {
		return err
	}

	if opts.Tag == "" {
		return fmt.Errorf("tag is not set")
	}

	const tagPrefix = "refs/tags/flipt-client-swift-"
	if !strings.HasPrefix(opts.Tag, tagPrefix) {
		return fmt.Errorf("tag %q must start with %q", opts.Tag, tagPrefix)
	}

	targetRepo := os.Getenv("TARGET_REPO")
	if targetRepo == "" {
		targetRepo = "https://github.com/flipt-io/flipt-client-swift.git"
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
