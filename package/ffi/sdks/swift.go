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
		platform.DarwinArm64,
	}
}

func (s *SwiftSDK) Build(ctx context.Context, client *dagger.Client, hostDirectory *dagger.Directory, opts BuildOpts) error {
	pat := os.Getenv("GITHUB_TOKEN")
	if pat == "" && opts.Push {
		return errors.New("GITHUB_TOKEN environment variable must be set")
	}

	var (
		secretPAT        = client.SetSecret("gh-token", pat)
		secretEncodedPAT = client.SetSecret("gh-token-encoded", base64.URLEncoding.EncodeToString([]byte("pat:"+pat)))
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
		WithSecretVariable("GITHUB_TOKEN", secretEncodedPAT).
		WithExec(args("git config --global user.email %s", gitUserEmail)).
		WithExec(args("git config --global user.name %s", gitUserName))

	if opts.Push {
		git = git.
			WithExec([]string{"sh", "-c", "git config --global http.https://github.com/.extraheader \"AUTHORIZATION: Basic ${GITHUB_TOKEN}\""})
	}

	repository := git.
		WithExec(args("git clone https://github.com/flipt-io/flipt-client-sdks.git /src")).
		WithWorkdir("/src").
		WithDirectory("/tmp/ext", hostDirectory.Directory("tmp")).
		WithFile("/tmp/ext/flipt_engine.h", hostDirectory.File("flipt-engine-ffi/include/flipt_engine.h"))

	filtered := repository.
		WithEnvVariable("FILTER_BRANCH_SQUELCH_WARNING", "1").
		WithExec([]string{"git", "filter-branch", "-f", "--prune-empty",
			"--subdirectory-filter", "flipt-client-swift",
			"--tree-filter", `
				[ -d Sources/FliptEngineFFI.xcframework/ios-arm64/Headers ] || mkdir -p Sources/FliptEngineFFI.xcframework/ios-arm64/Headers;
				[ -d Sources/FliptEngineFFI.xcframework/ios-arm64-simulator/Headers ] || mkdir -p Sources/FliptEngineFFI.xcframework/ios-arm64-simulator/Headers;
				[ -d Sources/FliptEngineFFI.xcframework/macos-arm64/Headers ] || mkdir -p Sources/FliptEngineFFI.xcframework/macos-arm64/Headers;
				cp /tmp/ext/ios_aarch64/libfliptengine.a Sources/FliptEngineFFI.xcframework/ios-arm64/;
				cp /tmp/ext/ios_aarch64_sim/libfliptengine.a Sources/FliptEngineFFI.xcframework/ios-arm64-simulator/;
				cp /tmp/ext/darwin_aarch64/libfliptengine.a Sources/FliptEngineFFI.xcframework/macos-arm64/;
				cp /tmp/ext/flipt_engine.h Sources/FliptEngineFFI.xcframework/ios-arm64/Headers;
				cp /tmp/ext/flipt_engine.h Sources/FliptEngineFFI.xcframework/ios-arm64-simulator/Headers;
				cp /tmp/ext/flipt_engine.h Sources/FliptEngineFFI.xcframework/macos-arm64/Headers;
			`, "--", opts.Tag})

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
	releasePayload := fmt.Sprintf(`{"tag_name":"%s","name":"%s","body":"Release %s of the Flipt Swift Client SDK"}`, targetTag, targetTag, targetTag)
	_, err = filtered.
		WithSecretVariable("GITHUB_TOKEN", secretPAT).
		WithExec([]string{
			"curl",
			"-X", "POST",
			"-H", "Authorization: token $GITHUB_TOKEN",
			"-H", "Accept: application/vnd.github.v3+json",
			"-d", releasePayload,
			"https://api.github.com/repos/flipt-io/flipt-client-swift/releases",
		}).Sync(ctx)

	return err
}
