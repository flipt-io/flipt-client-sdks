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

func (s *SwiftSDK) Build(ctx context.Context, client *dagger.Client, container *dagger.Container, hostDirectory *dagger.Directory, tmpDirectory *dagger.Directory, opts BuildOpts) error {
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

	git := container.From("golang:1.21.3-bookworm").
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
		WithDirectory("/tmp/ext", tmpDirectory).
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
				cat > Sources/FliptEngineFFI.xcframework/Info.plist << 'PLIST'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>AvailableLibraries</key>
	<array>
		<dict>
			<key>BinaryPath</key>
			<string>libfliptengine.a</string>
			<key>HeadersPath</key>
			<string>Headers</string>
			<key>LibraryIdentifier</key>
			<string>ios-arm64-simulator</string>
			<key>LibraryPath</key>
			<string>libfliptengine.a</string>
			<key>SupportedArchitectures</key>
			<array>
				<string>arm64</string>
			</array>
			<key>SupportedPlatform</key>
			<string>ios</string>
			<key>SupportedPlatformVariant</key>
			<string>simulator</string>
		</dict>
		<dict>
			<key>BinaryPath</key>
			<string>libfliptengine.a</string>
			<key>HeadersPath</key>
			<string>Headers</string>
			<key>LibraryIdentifier</key>
			<string>ios-arm64</string>
			<key>LibraryPath</key>
			<string>libfliptengine.a</string>
			<key>SupportedArchitectures</key>
			<array>
				<string>arm64</string>
			</array>
			<key>SupportedPlatform</key>
			<string>ios</string>
		</dict>
		<dict>
			<key>BinaryPath</key>
			<string>libfliptengine.a</string>
			<key>HeadersPath</key>
			<string>Headers</string>
			<key>LibraryIdentifier</key>
			<string>macos-arm64</string>
			<key>LibraryPath</key>
			<string>libfliptengine.a</string>
			<key>SupportedArchitectures</key>
			<array>
				<string>arm64</string>
			</array>
			<key>SupportedPlatform</key>
			<string>macos</string>
		</dict>
	</array>
	<key>CFBundlePackageType</key>
	<string>XFWK</string>
	<key>XCFrameworkFormatVersion</key>
	<string>1.0</string>
</dict>
</plist>
PLIST
				cat > Sources/FliptEngineFFI.xcframework/ios-arm64/Headers/module.modulemap << 'MODULEMAP'
module FliptEngineFFI {
  header "flipt_engine.h"
  export *
}
MODULEMAP
				cat > Sources/FliptEngineFFI.xcframework/ios-arm64-simulator/Headers/module.modulemap << 'MODULEMAP'
module FliptEngineFFI {
  header "flipt_engine.h"
  export *
}
MODULEMAP
				cat > Sources/FliptEngineFFI.xcframework/macos-arm64/Headers/module.modulemap << 'MODULEMAP'
module FliptEngineFFI {
  header "flipt_engine.h"
  export *
}
MODULEMAP
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
	token, _ := secretPAT.Plaintext(ctx)
	_, err = filtered.
		WithExec([]string{
			"curl",
			"-X", "POST",
			"-H", fmt.Sprintf("Authorization: token %s", token),
			"-H", "Accept: application/vnd.github.v3+json",
			"-d", releasePayload,
			"https://api.github.com/repos/flipt-io/flipt-client-swift/releases",
		}).Sync(ctx)

	return err
}
