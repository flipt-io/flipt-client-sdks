package sdks

import (
	"context"
	"io"
	"os"

	"dagger.io/dagger"
	"go.flipt.io/flipt/client-sdks/package/ffi/platform"
)

var defaultInclude = []string{"**/*.so", "**/*.dylib", "**/*.dll"}

type BuildOpts struct {
	Push bool
	Tag  string
}

type SDK interface {
	SupportedPlatforms() []platform.Platform
	Build(ctx context.Context, client *dagger.Client, hostDirectory *dagger.Directory, opts BuildOpts) error
}

type BaseSDK struct {
	Name string
}

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
