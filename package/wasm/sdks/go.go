package sdks

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"

	"dagger.io/dagger"
)

type GoSDK struct {
	BaseSDK
}

func (s *GoSDK) Build(ctx context.Context, client *dagger.Client, hostDirectory *dagger.Directory, opts BuildOpts) error {
	const (
		target   = "wasm32-wasip1"
		wasmFile = "/src/target/wasm32-wasip1/release/flipt_engine_wasm.wasm"
	)

	rust := client.Container().From("rust:1.83.0-bookworm").
		WithWorkdir("/src").
		WithDirectory("/src/flipt-engine-ffi", hostDirectory.Directory("flipt-engine-ffi")).
		WithDirectory("/src/flipt-engine-wasm", hostDirectory.Directory("flipt-engine-wasm"), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{"pkg/", ".gitignore"},
		}).
		WithDirectory("/src/flipt-engine-wasm-js", hostDirectory.Directory("flipt-engine-wasm-js"), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{"pkg/", ".gitignore"},
		}).
		WithDirectory("/src/flipt-evaluation", hostDirectory.Directory("flipt-evaluation")).
		WithFile("/src/Cargo.toml", hostDirectory.File("Cargo.toml")).
		WithExec(args("rustup target add " + target)).
		WithExec(args("cargo build -p flipt-engine-wasm --release --target " + target)).
		WithExec(args("cargo install wasm-opt")).
		WithExec(args("wasm-opt --converge --flatten --rereloop -Oz -Oz --gufa -o %s %s", wasmFile, wasmFile))

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

	container := git.
		WithExec(args("git clone https://github.com/flipt-io/flipt-client-sdks.git /src")).
		WithWorkdir("/src").
		WithFile("/ext/flipt_engine_wasm.wasm", rust.File(wasmFile))

	if !opts.Push {
		out, err := container.WithExec(args("apt-get update")).
			WithExec(args("apt-get install -y tree")).
			WithExec(args("tree /src/flipt-client-go")).
			WithExec(args("tree /ext")).
			Stdout(ctx)
		if err != nil {
			return err
		}
		fmt.Println(out)
		return nil
	}

	filtered := container.
		WithEnvVariable("FILTER_BRANCH_SQUELCH_WARNING", "1").
		WithExec([]string{"git", "filter-branch", "-f", "--prune-empty",
			"--subdirectory-filter", "flipt-client-go",
			"--tree-filter", "cp -r /ext .",
			"--", opts.Tag})

	if _, err := filtered.Sync(ctx); err != nil {
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
	token, _ := secretPAT.Plaintext(ctx)
	_, err := filtered.
		WithExec([]string{
			"curl",
			"-X", "POST",
			"-H", fmt.Sprintf("Authorization: token %s", token),
			"-H", "Accept: application/vnd.github.v3+json",
			"-d", releasePayload,
			"https://api.github.com/repos/flipt-io/flipt-client-go/releases",
		}).Sync(ctx)

	return err
}
