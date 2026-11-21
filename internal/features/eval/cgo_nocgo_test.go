//go:build !cgo

package eval

// cgoAvailable reports whether this test binary was built with cgo. The
// go-fix-race-condition task validates with `go test -race`, which the Go
// toolchain only supports in cgo builds.
const cgoAvailable = false
