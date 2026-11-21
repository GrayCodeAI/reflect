//go:build !cgo

package tool

// codeMatchAvailable reports whether tree-sitter grammars are linked in.
// CodeMatch tests skip in builds without cgo instead of failing.
const codeMatchAvailable = false
