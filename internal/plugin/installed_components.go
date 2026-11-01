package plugin

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// InstalledComponents discovers only installed plugins in the existing,
// trust-gated scope order. A higher-priority plugin shadows the whole package.
func InstalledComponents(projectRoot string) []DiscoveredComponents {
	var out []DiscoveredComponents
	seen := map[string]bool{}
	for _, scope := range DiscoverScopeDirs(projectRoot) {
		entries, err := os.ReadDir(scope.Path)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			dir := filepath.Join(scope.Path, entry.Name())
			manifest, err := ParseManifestV2(dir)
			if err != nil || len(ValidateManifestIdentity(manifest.Name, manifest.Version)) > 0 || seen[manifest.Name] {
				continue
			}
			seen[manifest.Name] = true
			components, err := DiscoverComponents(dir)
			if err != nil {
				slog.Warn("plugin components unavailable", "plugin", manifest.Name, "error", err)
				continue
			}
			out = append(out, components)
		}
	}
	return out
}

// InstalledMCPServers returns plugin contributions. Expansion is limited to
// explicit path tokens and never evaluates shell syntax or environment values.
// Env overrides are not supported by the host's MCP loader and fail closed.
func InstalledMCPServers(projectRoot string) []MCPServerSpec {
	root, err := filepath.Abs(projectRoot)
	if err != nil {
		return nil
	}
	var out []MCPServerSpec
	for _, components := range InstalledComponents(root) {
		replace := strings.NewReplacer("${project_root}", root, "${project_name}", filepath.Base(root), "${plugin_root}", components.Root)
		for _, spec := range components.MCPServers {
			if len(spec.Env) > 0 {
				slog.Warn("plugin MCP env overrides are unsupported; skipping server", "server", spec.Name)
				continue
			}
			spec.Command = replace.Replace(spec.Command)
			args := make([]string, len(spec.Args))
			for i, arg := range spec.Args {
				args[i] = replace.Replace(arg)
			}
			spec.Args = args
			out = append(out, spec)
		}
	}
	return out
}
