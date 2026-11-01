package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/GrayCodeAI/reflect/internal/fsutil"
)

// DiscoveredComponents is the result of scanning a multi-component plugin package.
type DiscoveredComponents struct {
	Root       string
	Skills     []string // absolute paths to SKILL.md parent dirs
	HookFiles  []string // absolute paths under hooks/
	MCPServers []MCPServerSpec
	HasTools   bool // plugin.json tools or tools/ present
}

// DiscoverComponents scans a plugin directory for tools, hooks, skills, and MCP.
// Layout (convention):
//
//	plugin/
//	  plugin.json          # required
//	  tools/               # optional tool binaries/scripts
//	  hooks/               # optional hook scripts
//	  skills/<name>/SKILL.md
//	  mcp.json             # optional MCP server definitions
func DiscoverComponents(pluginDir string) (DiscoveredComponents, error) {
	out := DiscoveredComponents{Root: pluginDir}

	m, err := ParseManifestV2(pluginDir)
	if err != nil {
		// Still try convention scan without manifest
		m = &ManifestV2{}
	} else {
		out.HasTools = len(m.Tools) > 0
	}

	skillsDir := "skills"
	hooksDir := "hooks"
	mcpFile := "mcp.json"
	if m.Components != nil {
		if m.Components.SkillsDir != "" {
			skillsDir = m.Components.SkillsDir
		}
		if m.Components.HooksDir != "" {
			hooksDir = m.Components.HooksDir
		}
		if m.Components.MCPFile != "" {
			mcpFile = m.Components.MCPFile
		}
		out.MCPServers = append(out.MCPServers, m.Components.MCP...)
	}
	for _, rel := range []string{skillsDir, hooksDir, mcpFile} {
		if err := withinPlugin(pluginDir, rel); err != nil {
			return out, err
		}
	}

	// tools/
	if st, err := os.Stat(filepath.Join(pluginDir, "tools")); err == nil && st.IsDir() {
		out.HasTools = true
	}

	// skills/
	skillsRoot := filepath.Join(pluginDir, skillsDir)
	if m.Components != nil && len(m.Components.Skills) > 0 {
		for _, name := range m.Components.Skills {
			if err := withinPlugin(pluginDir, filepath.Join(skillsDir, name, "SKILL.md")); err != nil {
				return out, err
			}
			p := filepath.Join(skillsRoot, name)
			if skillMD := filepath.Join(p, "SKILL.md"); fileExists(skillMD) {
				out.Skills = append(out.Skills, p)
			}
		}
	} else {
		entries, err := os.ReadDir(skillsRoot)
		if err == nil {
			for _, e := range entries {
				if !e.IsDir() {
					continue
				}
				p := filepath.Join(skillsRoot, e.Name())
				if err := withinPlugin(pluginDir, filepath.Join(skillsDir, e.Name(), "SKILL.md")); err != nil {
					return out, err
				}
				if fileExists(filepath.Join(p, "SKILL.md")) {
					out.Skills = append(out.Skills, p)
				}
			}
		}
	}

	// hooks/
	hooksRoot := filepath.Join(pluginDir, hooksDir)
	_ = filepath.WalkDir(hooksRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil || d == nil || d.IsDir() {
			return nil
		}
		name := d.Name()
		if strings.HasSuffix(name, ".sh") || strings.HasSuffix(name, ".md") || strings.HasSuffix(name, ".json") {
			out.HookFiles = append(out.HookFiles, path)
		}
		return nil
	})

	// mcp.json
	mcpPath := filepath.Join(pluginDir, mcpFile)
	if data, err := fsutil.ReadPinnedFile(mcpPath); err == nil {
		var file struct {
			Servers []MCPServerSpec `json:"servers"`
			// also accept map form { "servers": { "name": {...} } }
			ServerMap map[string]MCPServerSpec `json:"mcpServers"`
		}
		if json.Unmarshal(data, &file) == nil {
			out.MCPServers = append(out.MCPServers, file.Servers...)
			names := make([]string, 0, len(file.ServerMap))
			for name := range file.ServerMap {
				names = append(names, name)
			}
			sort.Strings(names)
			for _, name := range names {
				spec := file.ServerMap[name]
				if spec.Name == "" {
					spec.Name = name
				}
				out.MCPServers = append(out.MCPServers, spec)
			}
		}
	}

	return out, nil
}

func withinPlugin(root, rel string) error {
	if filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(filepath.Clean(rel), ".."+string(filepath.Separator)) {
		return fmt.Errorf("plugin component escapes package: %s", rel)
	}
	resolved, err := filepath.EvalSymlinks(filepath.Join(root, rel))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	base, err := filepath.EvalSymlinks(root)
	if err != nil {
		return err
	}
	relative, err := filepath.Rel(base, resolved)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("plugin component symlink escapes package: %s", rel)
	}
	return nil
}

// ComponentSummary returns a short human-readable description of components.
func (d DiscoveredComponents) ComponentSummary() string {
	var parts []string
	if d.HasTools {
		parts = append(parts, "tools")
	}
	if len(d.HookFiles) > 0 {
		parts = append(parts, "hooks")
	}
	if len(d.Skills) > 0 {
		parts = append(parts, "skills")
	}
	if len(d.MCPServers) > 0 {
		parts = append(parts, "mcp")
	}
	if len(parts) == 0 {
		return "empty"
	}
	return strings.Join(parts, "+")
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}
