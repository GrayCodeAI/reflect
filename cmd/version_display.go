package cmd

import (
	"strings"

	reflectmod "github.com/GrayCodeAI/reflect"
)

// versionLine is the single user-facing version format shared by
// `reflect --version` and `reflect version`.
func versionLine() string {
	ver := DisplayVersion()
	if ver != "" && !strings.HasPrefix(ver, "v") && !strings.HasPrefix(ver, "V") {
		ver = "v" + ver
	}
	line := auditTint("reflect", textPrimary) + " " + auditTint(ver, reflectColor)
	if d := strings.TrimSpace(buildDate); d != "" && d != "unknown" {
		line += auditTint(" (built "+d+")", textMuted)
	}
	return line
}

// DisplayVersion returns the build-injected version or the embedded VERSION.
// It never depends on the current directory or files next to the binary.
func DisplayVersion() string {
	if v := strings.TrimSpace(version); v != "" && v != "dev" {
		return v
	}
	return reflectmod.Version
}
