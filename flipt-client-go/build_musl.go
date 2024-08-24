//go:build musl
// +build musl

package evaluation

/*
#cgo CFLAGS: -I./ext
#cgo linux,arm64 LDFLAGS: -L${SRCDIR}/ext/linux_arm64_musl -lfliptengine  -Wl,-rpath,${SRCDIR}/ext/linux_arm64_musl
#cgo linux,amd64 LDFLAGS: -L${SRCDIR}/ext/linux_x86_64_musl -lfliptengine -Wl,-rpath,${SRCDIR}/ext/linux_x86_64_musl
*/
import "C"
