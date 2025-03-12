package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"maps"
	"os"
	"os/exec"
	"strings"

	"dagger.io/dagger"
	"go.flipt.io/flipt/client-sdks/package/ffi/platform"
	"go.flipt.io/flipt/client-sdks/package/ffi/sdks"
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
	var builds = make(map[string]sdks.SDK, len(sdksToFn))

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
		Exclude: []string{".github/", "package/", "test/", ".git/"},
	})

	var g errgroup.Group

	for _, s := range builds {
		s := s
		g.Go(take(func() error {
			if err := downloadFFI(ctx, client, s); err != nil {
				return err
			}

			container := client.Container(dagger.ContainerOpts{
				Platform: dagger.Platform("linux/amd64"),
			})

			return s.Build(ctx, client, container, dir, sdks.BuildOpts{
				Tag:  tag,
				Push: push,
			})
		}))
	}

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

func getLatestEngineTag() (string, error) {
	// check if jq is installed
	if _, err := exec.LookPath("jq"); err != nil {
		return "", fmt.Errorf("jq is not installed")
	}

	// check if curl is installed
	if _, err := exec.LookPath("curl"); err != nil {
		return "", fmt.Errorf("curl is not installed")
	}

	// use github api to get the latest release of the ffi engine
	cmd := exec.Command("sh", "-c", `
		curl -s "https://api.github.com/repos/flipt-io/flipt-client-sdks/releases" |
		jq -r '.[] | select(.tag_name | startswith("flipt-engine-ffi-")) | .tag_name' |
		sort -Vr |
		head -n 1 |
		sed "s/^flipt-engine-ffi-//"
	`)

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get latest engine tag: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

func downloadFFI(ctx context.Context, client *dagger.Client, sdk sdks.SDK) error {
	var err error
	if engineTag == "" {
		engineTag, err = getLatestEngineTag()
		if err != nil {
			return err
		}
	}

	if engineTag == "" {
		return fmt.Errorf("engine tag is not found")
	}

	packages := sdk.SupportedPlatforms()

	if err := os.RemoveAll("tmp"); err != nil {
		return fmt.Errorf("failed to remove tmp directory: %w", err)
	}

	for _, pkg := range packages {
		pkg := pkg

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
			Export(ctx, "tmp"); err != nil {
			return err
		}
	}

	return nil
}
