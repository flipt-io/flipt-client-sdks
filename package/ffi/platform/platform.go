package platform

type Ext string

const (
	TarGz Ext = "tar.gz"
	Zip   Ext = "zip"
)

type Libc string

const (
	Glibc Libc = "glibc"
	Musl  Libc = "musl"
)

type Platform struct {
	ID     string
	Target string
	Ext    Ext
	Libc   Libc
}

func (p Platform) String() string {
	return p.ID
}

var (
	LinuxArm64      = Platform{ID: "Linux-arm64", Target: "aarch64-unknown-linux-gnu", Libc: Glibc}
	LinuxArm64Musl  = Platform{ID: "Linux-arm64-musl", Target: "aarch64-unknown-linux-musl", Libc: Musl}
	LinuxX86_64     = Platform{ID: "Linux-x86_64", Target: "x86_64-unknown-linux-gnu", Libc: Glibc}
	LinuxX86_64Musl = Platform{ID: "Linux-x86_64-musl", Target: "x86_64-unknown-linux-musl", Libc: Musl}
	DarwinArm64     = Platform{ID: "Darwin-arm64", Target: "aarch64-apple-darwin"}
	DarwinX86_64    = Platform{ID: "Darwin-x86_64", Target: "x86_64-apple-darwin"}
	WindowsX86_64   = Platform{ID: "Windows-x86_64", Target: "x86_64-pc-windows-msvc", Ext: Zip}
	IOSArm64        = Platform{ID: "iOS-arm64", Target: "aarch64-apple-ios"}
	IOSSimArm64     = Platform{ID: "iOS-arm64-sim", Target: "aarch64-apple-ios-sim"}
)
