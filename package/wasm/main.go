package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"maps"
	"os"
	"strings"

	"dagger.io/dagger"
	"go.flipt.io/flipt/client-sdks/package/wasm/sdks"
	"golang.org/x/sync/errgroup"
)

var (
	selected string
	push     bool
	tag      string
	sdksToFn = map[string]sdks.SDK{
		"node":    &sdks.NodeSDK{},
		"browser": &sdks.BrowserSDK{},
		"go":      &sdks.GoSDK{},
	}
	sema = make(chan struct{}, 5)
)

func init() {
	flag.StringVar(&selected, "sdks", "", "comma separated list of which sdks(s) to run builds for")
	flag.BoolVar(&push, "push", false, "push built artifacts to registry")
	flag.StringVar(&tag, "tag", "", "tag to use for release")
}

func main() {
	flag.Parse()

	if err := run(); err != nil {
		log.Fatal(err)
	}
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
			container := client.Container(dagger.ContainerOpts{
				Platform: dagger.Platform("linux/amd64"),
			})

			return s.Build(ctx, client, container, dir, sdks.BuildOpts{
				Push: push,
				Tag:  tag,
			})
		}))
	}

	return g.Wait()
}
