package sdks

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"dagger.io/dagger"
	"go.flipt.io/flipt/client-sdks/package/ffi/platform"
)

type DartSDK struct {
	BaseSDK
}

func (s *DartSDK) SupportedPlatforms() []platform.Platform {
	return []platform.Platform{
		platform.AndroidArm64,
	}
}

func (s *DartSDK) Build(ctx context.Context, client *dagger.Client, container *dagger.Container, hostDirectory *dagger.Directory, tmpDirectory *dagger.Directory, opts BuildOpts) error {
	// we need to download the FliptEngineFFI.xcframework.tar.gz file from the release and extract it into the tmp directory to support the iOS SDK
	var (
		url = fmt.Sprintf("https://github.com/flipt-io/flipt-client-sdks/releases/download/flipt-engine-ffi-%s/%s.%s", opts.EngineTag, "FliptEngineFFI.xcframework", "tar.gz")
	)

	_, err := client.Container().From("debian:bookworm-slim").
		WithExec(args("apt-get update")).
		WithExec(args("apt-get install -y wget p7zip-full")).
		WithExec(args("mkdir -p /tmp/dl")).
		WithExec(args("wget --spider %s", url)).
		WithExec(args("wget %s -O /tmp/dl/%s.%s", url, "FliptEngineFFI.xcframework", "tar.gz")).
		WithExec(args("mkdir -p /out/ios")).
		WithExec(args("tar xzf /tmp/dl/%s.%s -C /out/ios", "FliptEngineFFI.xcframework", "tar.gz")).
		Directory("/out").
		Export(ctx, "tmp")
	if err != nil {
		return err
	}

	// the directory structure of the tmp directory is as follows:
	// tmp/android_x86_64/
	// tmp/android_aarch64/

	// we need to convert it to the following structure:
	// staging/android/src/main/jniLibs/x86_64/
	// staging/android/src/main/jniLibs/arm64-v8a/

	// this is because the Android SDK and iOS SDKs expects the library to be in a specific directory structure based on host OS and architecture
	// the other platforms just pass through to the native directory
	type rename struct {
		old string
		new string
	}

	renames := []rename{
		{old: "android_aarch64", new: "android/src/main/jniLibs/arm64-v8a"},
		{old: "ios", new: "ios"},
	}

	if err := os.RemoveAll("staging"); err != nil {
		return err
	}

	if err := os.MkdirAll("staging", 0o755); err != nil {
		return err
	}

	for _, rename := range renames {
		tmp := fmt.Sprintf("tmp/%s", rename.old)
		// if the directory does not exist or is empty, skip it
		if isEmpty, err := isDirEmptyOrNotExist(tmp); err != nil {
			return fmt.Errorf("error checking directory %s: %w", tmp, err)
		} else if isEmpty {
			fmt.Printf("directory %s is empty or does not exist, skipping\n", tmp)
			continue
		}

		// staging directory to copy the files into
		stg := fmt.Sprintf("staging/%s", rename.new)

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

	include := []string{"**/FliptEngineFFI.xcframework/**"}
	include = append(include, dynamicInclude...)

	container = container.From("ghcr.io/cirruslabs/flutter:stable").
		WithDirectory("/src", hostDirectory.Directory("flipt-client-dart"), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{".gitignore", ".dart_tool/"},
		}).
		WithDirectory("/src", hostDirectory.Directory("staging"), dagger.ContainerWithDirectoryOpts{
			Include: include,
		}).
		WithFile("/src/ffi/flipt_engine.h", hostDirectory.File("flipt-engine-ffi/include/flipt_engine.h")).
		WithWorkdir("/src").
		WithExec(args("flutter pub get"))

	if !opts.Push {
		out, err := container.WithExec(args("dart pub publish --dry-run")).
			WithExec(args("apt-get update")).
			WithExec(args("apt-get install -y tree")).
			WithExec(args("tree /src")).
			Stdout(ctx)
		if err != nil {
			return err
		}
		fmt.Println(out)
		return nil
	}

	// get oidc token for publishing to pub.dev
	// https://dart.dev/tools/pub/automated-publishing#publishing-packages-using-github-actions
	if os.Getenv("PUB_TOKEN") == "" {
		return fmt.Errorf("PUB_TOKEN is not set")
	}

	pubToken := client.SetSecret("pub-token", os.Getenv("PUB_TOKEN"))

	_, err = container.WithSecretVariable("PUB_TOKEN", pubToken).
		WithExec(args("dart pub token add https://pub.dev --env-var PUB_TOKEN")).
		WithExec(args("dart pub publish --force")).
		Sync(ctx)

	return err
}
