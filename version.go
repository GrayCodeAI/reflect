// Package decoder exposes the canonical project version.
package decoder

import (
	_ "embed"
	"strings"
)

//go:embed VERSION
var versionFile string

// Version is embedded from the repository's sole version source, VERSION.
var Version = strings.TrimSpace(versionFile)
