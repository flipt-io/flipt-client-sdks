package sdks

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"dagger.io/dagger"
)

type JavaSDK struct {
	BaseSDK
}

func (s *JavaSDK) Build(ctx context.Context, client *dagger.Client, container *dagger.Container, hostDirectory *dagger.Directory, opts BuildOpts) error {
	// the directory structure of the tmp directory is as follows:
	// tmp/linux_x86_64/
	// tmp/linux_aarch64/
	// tmp/darwin_x86_64/
	// tmp/darwin_aarch64/

	// we need to convert it to the following structure:
	// staging/linux-x86-64/
	// staging/linux-aarch64/
	// staging/darwin-x86-64/
	// staging/darwin-aarch64/

	// this is because JNA expects the library to be in a specific directory structure based on host OS and architecture
	// see: https://github.com/java-native-access/jna/blob/master/test/com/sun/jna/PlatformTest.java
	// we can do this on the host and then copy the files into the container

	type rename struct {
		old string
		new string
	}

	renames := []rename{
		{old: "linux_x86_64", new: "linux-x86-64"},
		{old: "linux_aarch64", new: "linux-aarch64"},
		{old: "darwin_x86_64", new: "darwin-x86-64"},
		{old: "darwin_aarch64", new: "darwin-aarch64"},
		{old: "windows_x86_64", new: "win32-x86-64"},
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

	container = container.From("gradle:8-jdk17").
		WithDirectory("/src", hostDirectory.Directory("flipt-client-java")).
		WithDirectory("/src/src/main/resources", hostDirectory.Directory("staging"), dagger.ContainerWithDirectoryOpts{
			Include: dynamicInclude,
		}).
		WithWorkdir("/src").
		WithExec(args("chown -R gradle:gradle /src")).
		WithExec(args("gradle clean")).
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

	if os.Getenv("SONATYPE_PORTAL_USERNAME") == "" {
		return fmt.Errorf("SONATYPE_PORTAL_USERNAME is not set")
	}
	if os.Getenv("SONATYPE_PORTAL_PASSWORD") == "" {
		return fmt.Errorf("SONATYPE_PORTAL_PASSWORD is not set")
	}
	if os.Getenv("PGP_PRIVATE_KEY") == "" {
		return fmt.Errorf("PGP_PRIVATE_KEY is not set")
	}
	if os.Getenv("PGP_PASSPHRASE") == "" {
		return fmt.Errorf("PGP_PASSPHRASE is not set")
	}

	var (
		sonatypePortalUsername = client.SetSecret("sonatype-portal-username", os.Getenv("SONATYPE_PORTAL_USERNAME"))
		sonatypePortalPassword = client.SetSecret("sonatype-portal-password", os.Getenv("SONATYPE_PORTAL_PASSWORD"))
		pgpPrivateKey          = client.SetSecret("pgp-private-key", os.Getenv("PGP_PRIVATE_KEY"))
		pgpPassphrase          = client.SetSecret("pgp-passphrase", os.Getenv("PGP_PASSPHRASE"))
	)

	_, err = container.WithSecretVariable("SONATYPE_PORTAL_USERNAME", sonatypePortalUsername).
		WithSecretVariable("SONATYPE_PORTAL_PASSWORD", sonatypePortalPassword).
		WithSecretVariable("PGP_PRIVATE_KEY", pgpPrivateKey).
		WithSecretVariable("PGP_PASSPHRASE", pgpPassphrase).
		WithExec(args("gradle publishToSonatype closeAndReleaseSonatypeStagingRepository")).
		Sync(ctx)

	return err
}
