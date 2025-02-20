package platform

type Ext string

const (
	TarGz Ext = "tar.gz"
	Zip   Ext = "zip"
)

type Platform struct {
	ID     string
	Target string
	Ext    Ext
}

func (p Platform) String() string {
	return p.ID
}

var (
	LinuxArm64    = Platform{ID: "Linux-aarch64", Target: "aarch64-unknown-linux-musl"}
	LinuxX86_64   = Platform{ID: "Linux-x86_64", Target: "x86_64-unknown-linux-musl"}
	DarwinArm64   = Platform{ID: "Darwin-aarch64", Target: "aarch64-apple-darwin"}
	DarwinX86_64  = Platform{ID: "Darwin-x86_64", Target: "x86_64-apple-darwin"}
	WindowsX86_64 = Platform{ID: "Windows-x86_64", Target: "x86_64-pc-windows-msvc", Ext: Zip}
	IOSArm64      = Platform{ID: "iOS-aarch64", Target: "aarch64-apple-ios"}
	IOSSimArm64   = Platform{ID: "iOS-aarch64-sim", Target: "aarch64-apple-ios-sim"}
)
