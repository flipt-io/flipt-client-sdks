//go:build !musl
// +build !musl

package evaluation

/*
#cgo CFLAGS: -I./ext
#cgo darwin,arm64 LDFLAGS: -L${SRCDIR}/ext/darwin_arm64 -lfliptengine  -Wl,-rpath,${SRCDIR}/ext/darwin_arm64
#cgo darwin,amd64 LDFLAGS: -L${SRCDIR}/ext/darwin_x86_64 -lfliptengine -Wl,-rpath,${SRCDIR}/ext/darwin_x86_64
#cgo linux,arm64 LDFLAGS: -L${SRCDIR}/ext/linux_arm64 -lfliptengine  -Wl,-rpath,${SRCDIR}/ext/linux_arm64
#cgo linux,amd64 LDFLAGS: -L${SRCDIR}/ext/linux_x86_64 -lfliptengine -Wl,-rpath,${SRCDIR}/ext/linux_x86_64
*/
import "C"
