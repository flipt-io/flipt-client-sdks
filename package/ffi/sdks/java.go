package sdks

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"dagger.io/dagger"
	"go.flipt.io/flipt/client-sdks/package/ffi/platform"
)

type JavaSDK struct {
	BaseSDK
	Libc platform.Libc
}

func (s *JavaSDK) SupportedPlatforms() []platform.Platform {
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

func (s *JavaSDK) Build(ctx context.Context, client *dagger.Client, hostDirectory *dagger.Directory, opts BuildOpts) error {
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

	libc := platform.Glibc
	if s.Libc != "" {
		libc = s.Libc
	}

	for _, rename := range renames {
		tmp := fmt.Sprintf("tmp/%s/%s", libc, rename.old)
		// if the directory does not exist or is empty, skip it
		if isEmpty, err := isDirEmptyOrNotExist(tmp); err != nil {
			return fmt.Errorf("error checking directory %s: %w", tmp, err)
		} else if isEmpty {
			continue
		}

		// staging directory to copy the files into
		stg := fmt.Sprintf("staging/%s/%s", libc, rename.new)

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
		WithDirectory("/src/src/main/resources", hostDirectory.Directory(fmt.Sprintf("staging/%s", libc)), dagger.ContainerWithDirectoryOpts{
			Include: defaultInclude,
		}).
		WithWorkdir("/src")

	if libc == platform.Musl {
		container = container.
			WithExec([]string{"gradle", "-x", "test", "--build-file", "build.musl.gradle", "build"})
	} else {
		container = container.WithExec([]string{"gradle", "-x", "test", "build"})
	}

	var err error

	if !opts.Push {
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

	if libc == platform.Musl {
		_, err = container.WithExec([]string{"gradle", "-b", "build.musl.gradle", "publish"}).Sync(ctx)
	} else {
		_, err = container.WithExec([]string{"gradle", "publish"}).Sync(ctx)
	}

	return err
}
