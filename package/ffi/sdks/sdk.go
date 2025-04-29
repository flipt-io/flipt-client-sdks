package sdks

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"dagger.io/dagger"
	"go.flipt.io/flipt/client-sdks/package/ffi/platform"
)

var (
	dynamicInclude = []string{"**/*.so", "**/*.dylib", "**/*.dll"}
	staticInclude  = []string{"**/*.a"}
)

type BuildOpts struct {
	Push      bool
	Tag       string
	EngineTag string
}

type SDK interface {
	SupportedPlatforms() []platform.Platform
	Build(ctx context.Context, client *dagger.Client, container *dagger.Container, hostDirectory *dagger.Directory, opts BuildOpts) error
}

type BaseSDK struct{}

func (s *BaseSDK) SupportedPlatforms() []platform.Platform {
	return []platform.Platform{
		platform.LinuxX86_64,
		platform.LinuxArm64,
		platform.DarwinX86_64,
		platform.DarwinArm64,
		platform.WindowsX86_64,
	}
}

func isDirEmptyOrNotExist(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}

func args(args string, a ...any) []string {
	if len(a) == 0 {
		return strings.Split(args, " ")
	}
	return strings.Split(fmt.Sprintf(args, a...), " ")
}
