package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"maps"
	"net/http"
	"os"
	"strings"

	"dagger.io/dagger"
	"go.flipt.io/flipt/client-sdks/package/ffi/platform"
	"go.flipt.io/flipt/client-sdks/package/ffi/sdks"
	"golang.org/x/mod/semver"
	"golang.org/x/sync/errgroup"
)

var (
	selected  string
	push      bool
	tag       string
	engineTag string
	sdksToFn  = map[string]sdks.SDK{
		"python":  &sdks.PythonSDK{},
		"ruby":    &sdks.RubySDK{},
		"java":    &sdks.JavaSDK{},
		"dart":    &sdks.DartSDK{},
		"csharp":  &sdks.CSharpSDK{},
		"swift":   &sdks.SwiftSDK{},
		"android": &sdks.AndroidSDK{},
	}
	sema = make(chan struct{}, 5)
)

func init() {
	flag.StringVar(&selected, "sdks", "", "comma separated list of which sdks(s) to run builds for")
	flag.BoolVar(&push, "push", false, "push built artifacts to registry")
	flag.StringVar(&tag, "tag", "", "tag to use for release")
	flag.StringVar(&engineTag, "engine-tag", "", "tag to use for the engine ffi")
}

func main() {
	flag.Parse()

	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	builds := make(map[string]sdks.SDK, len(sdksToFn))

	maps.Copy(builds, sdksToFn)

	if selected != "" {
		l := strings.Split(selected, ",")
		subset := make(map[string]sdks.SDK, len(l))
		for _, sdk := range l {
			testFn, ok := sdksToFn[sdk]
			if !ok {
				return fmt.Errorf("sdk %s is not supported", sdk)
			}
			subset[sdk] = testFn
		}

		builds = subset
	}

	ctx := context.Background()

	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stdout))
	if err != nil {
		return err
	}

	defer client.Close()

	dir := client.Host().Directory(".", dagger.HostDirectoryOpts{
		Exclude:   []string{".github/", "package/", "test/", ".git/"},
		Gitignore: false,
	})

	// Get absolute path for tmp directory to work with Dagger 0.18.17+
	// NoGitAutoIgnore is required because tmp/ is in .gitignore and Dagger 0.18.17
	// automatically applies gitignore patterns when loading directories
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}
	tmpPath := fmt.Sprintf("%s/tmp", cwd)
	tmpDir := client.Host().Directory(tmpPath, dagger.HostDirectoryOpts{
		Gitignore: false,
	})

	var g errgroup.Group

	for _, s := range builds {
		g.Go(take(func() error {
			if err := downloadFFI(ctx, client, s); err != nil {
				return err
			}

			container := client.Container(dagger.ContainerOpts{
				Platform: dagger.Platform("linux/amd64"),
			})

			return s.Build(ctx, client, container, dir, tmpDir, sdks.BuildOpts{
				Tag:       tag,
				EngineTag: engineTag,
				Push:      push,
			})
		}))
	}

	defer func() {
		if err := os.RemoveAll("tmp"); err != nil {
			fmt.Printf("error removing tmp directory: %s\n", err)
		}
	}()

	return g.Wait()
}

func take(fn func() error) func() error {
	return func() error {
		// insert into semaphore channel to maintain
		// a max concurrency
		sema <- struct{}{}
		defer func() { <-sema }()

		return fn()
	}
}

func args(args string, a ...any) []string {
	if len(a) == 0 {
		return strings.Split(args, " ")
	}
	return strings.Split(fmt.Sprintf(args, a...), " ")
}

const engineTagPrefix = "flipt-engine-ffi-"

// getLatestEngineTag returns the highest published ffi engine version (e.g. v0.21.4).
// GITHUB_TOKEN is sent when set, since unauthenticated requests are rate limited.
func getLatestEngineTag(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://api.github.com/repos/flipt-io/flipt-client-sdks/releases?per_page=100", nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to list releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("failed to list releases: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var releases []struct {
		TagName    string `json:"tag_name"`
		Draft      bool   `json:"draft"`
		Prerelease bool   `json:"prerelease"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return "", fmt.Errorf("failed to decode releases: %w", err)
	}

	var latest string
	for _, r := range releases {
		if r.Draft || r.Prerelease || !strings.HasPrefix(r.TagName, engineTagPrefix) {
			continue
		}

		version := strings.TrimPrefix(r.TagName, engineTagPrefix)
		if semver.IsValid(version) && (latest == "" || semver.Compare(version, latest) > 0) {
			latest = version
		}
	}

	return latest, nil
}

func downloadFFI(ctx context.Context, client *dagger.Client, sdk sdks.SDK) error {
	var err error
	if engineTag == "" {
		engineTag, err = getLatestEngineTag(ctx)
		if err != nil {
			return err
		}
	}

	if engineTag == "" {
		return fmt.Errorf("engine tag is not found")
	}

	packages := sdk.SupportedPlatforms()

	// Get absolute path for tmp directory to work with Dagger 0.18.17+
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}
	tmpDir := fmt.Sprintf("%s/tmp", cwd)

	if err := os.RemoveAll(tmpDir); err != nil {
		return fmt.Errorf("failed to remove tmp directory: %w", err)
	}

	for _, pkg := range packages {
		ext := platform.TarGz
		if pkg.Ext != "" {
			ext = pkg.Ext
		}

		var (
			out = strings.ToLower(strings.ReplaceAll(pkg.ID, "-", "_"))
			url = fmt.Sprintf("https://github.com/flipt-io/flipt-client-sdks/releases/download/flipt-engine-ffi-%s/flipt-engine-ffi-%s.%s", engineTag, pkg.ID, ext)
		)

		container := client.Container().From("debian:bookworm-slim").
			WithExec(args("apt-get update")).
			WithExec(args("apt-get install -y wget p7zip-full")).
			WithExec(args("mkdir -p /tmp/dl")).
			WithExec(args("wget --spider %s", url)).
			WithExec(args("wget %s -O /tmp/dl/%s.%s", url, pkg.ID, ext)).
			WithExec(args("mkdir -p /tmp/%s", out)).
			WithExec(args("mkdir -p /out/%s", out))

		if ext == "tar.gz" {
			container = container.WithExec(args("tar xzf /tmp/dl/%s.%s -C /tmp/%s", pkg.ID, ext, out))
		} else {
			container = container.WithExec(args("7z x /tmp/dl/%s.%s -o/tmp/%s", pkg.ID, ext, out))
		}

		// copy the files into the output directory
		// ex: /tmp/darwin_aarch64/target/aarch64-apple-darwin/release/libfliptengine.a -> /out/darwin_aarch64/libfliptengine.a
		cmd := []string{"sh", "-c", fmt.Sprintf("cp -r /tmp/%s/target/%s/release/* /out/%s", out, pkg.Target, out)}
		if _, err := container.
			WithExec(cmd).
			Directory("/out").
			Export(ctx, tmpDir); err != nil {
			return err
		}
	}

	return nil
}
