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
		platform.LinuxX86_64,
		platform.LinuxArm64,
		platform.DarwinX86_64,
		platform.DarwinArm64,
		platform.WindowsX86_64,
		platform.AndroidArm64,
		platform.AndroidX86_64,
		//platform.IOSArm64,
	}
}

func (s *DartSDK) Build(ctx context.Context, client *dagger.Client, container *dagger.Container, hostDirectory *dagger.Directory, opts BuildOpts) error {

	// the directory structure of the tmp directory is as follows:
	// tmp/android_x86_64/
	// tmp/android_aarch64/

	// we need to convert it to the following structure:
	// staging/android/x86_64/
	// staging/android/arm64-v8a/

	// this is because the Android SDK and iOS SDKs expects the library to be in a specific directory structure based on host OS and architecture
	// the other platforms just pass through to the binaries directory
	type rename struct {
		old string
		new string
	}

	renames := []rename{
		{old: "android_x86_64", new: "android/x86_64"},
		{old: "android_aarch64", new: "android/arm64-v8a"},
		{old: "linux_x86_64", new: "binaries/linux_x86_64"},
		{old: "linux_aarch64", new: "binaries/linux_aarch64"},
		{old: "darwin_x86_64", new: "binaries/darwin_x86_64"},
		{old: "darwin_aarch64", new: "binaries/darwin_aarch64"},
		{old: "windows_x86_64", new: "binaries/windows_x86_64"},
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

	container = container.From("dart:stable").
		WithDirectory("/src", hostDirectory.Directory("flipt-client-dart"), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{".gitignore", ".dart_tool/"},
		}).
		WithDirectory("/src", hostDirectory.Directory("staging"), dagger.ContainerWithDirectoryOpts{
			Include: dynamicInclude,
		}).
		WithFile("/src/ffi/flipt_engine.h", hostDirectory.File("flipt-engine-ffi/include/flipt_engine.h")).
		WithWorkdir("/src").
		WithExec(args("dart pub get"))

	var err error

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
