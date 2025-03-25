package sdks

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"dagger.io/dagger"
	"go.flipt.io/flipt/client-sdks/package/ffi/platform"
)

type AndroidSDK struct {
	BaseSDK
}

func (s *AndroidSDK) SupportedPlatforms() []platform.Platform {
	return []platform.Platform{
		platform.AndroidArm64,
		platform.AndroidX86_64,
	}
}

func (s *AndroidSDK) Build(ctx context.Context, client *dagger.Client, container *dagger.Container, hostDirectory *dagger.Directory, opts BuildOpts) error {
	// the directory structure of the tmp directory is as follows:
	// tmp/android_x86_64/
	// tmp/android_aarch64/

	// we need to convert it to the following structure:
	// staging/x86_64/
	// staging/arm64-v8a/

	// this is because the Android SDK expects the library to be in a specific directory structure based on host OS and architecture

	type rename struct {
		old string
		new string
	}

	renames := []rename{
		{old: "android_x86_64", new: "x86_64"},
		{old: "android_aarch64", new: "arm64-v8a"},
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

	container = container.From("gradle:8.11-jdk17").
		WithExec(args("mkdir -p /android-sdk/cmdline-tools")).
		WithEnvVariable("ANDROID_HOME", "/android-sdk").
		WithWorkdir("/android-sdk/cmdline-tools").
		WithExec(args("wget https://dl.google.com/android/repository/commandlinetools-linux-11076708_latest.zip")).
		WithExec(args("unzip commandlinetools-linux*.zip")).
		WithWorkdir("/android-sdk/cmdline-tools").
		WithExec(args("mv cmdline-tools latest")).
		WithWorkdir("/android-sdk/cmdline-tools/latest").
		WithExec([]string{"sh", "-c", "yes | bin/sdkmanager --licenses"}).
		WithExec([]string{"bin/sdkmanager", "--install", "build-tools;33.0.0"}).
		WithExec([]string{"bin/sdkmanager", "--install", "platforms;android-33"}).
		WithExec([]string{"bin/sdkmanager", "--install", "platform-tools"}).
		WithExec([]string{"bin/sdkmanager", "--install", "cmdline-tools;latest"}).
		WithExec([]string{"bin/sdkmanager", "--install", "cmake;3.22.1"}).
		WithDirectory("/src", hostDirectory.Directory("flipt-client-kotlin-android")).
		WithDirectory("/src/src/main/cpp/libs", hostDirectory.Directory("staging"), dagger.ContainerWithDirectoryOpts{
			Include: staticInclude,
		}).
		WithFile("/src/src/main/cpp/include/flipt_engine.h", hostDirectory.File("flipt-engine-ffi/include/flipt_engine.h")).
		WithWorkdir("/src").
		WithExec(args("gradle -x test build"))

	var err error

	if !opts.Push {
		out, err := container.WithExec(args("apt-get update")).
			WithExec(args("apt-get install -y tree")).
			WithExec(args("tree /src")).
			Stdout(ctx)
		if err != nil {
			return err
		}
		fmt.Println(out)
		return nil
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
		WithExec(args("gradle publishToSonatype closeAndReleaseSonatypeStagingRepository")).
		Sync(ctx)

	return err
}
