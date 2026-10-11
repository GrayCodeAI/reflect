package cmd

import (
	"strings"

	"github.com/GrayCodeAI/decoder"
)

// versionLine is the single user-facing version format shared by
// `decoder --version` and `decoder version`.
func versionLine() string {
	ver := DisplayVersion()
	if ver != "" && !strings.HasPrefix(ver, "v") && !strings.HasPrefix(ver, "V") {
		ver = "v" + ver
	}
	line := auditTint("decoder", textPrimary) + " " + auditTint(ver, decoderColor)
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
	return decoder.Version
}
