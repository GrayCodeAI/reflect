package plugin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GrayCodeAI/reflect/internal/trust"
)

func TestInstallWorkspacePackagePreservesAndLoadsComponents(t *testing.T) {
	t.Setenv("REFLECT_STATE_DIR", t.TempDir())
	t.Setenv("REFLECT_CONFIG_DIR", t.TempDir())
	t.Setenv("REFLECT_MANAGED_PLUGINS", "")
	if err := Install(filepath.Join("..", "..", "examples", "plugins", "graycode-workspace")); err != nil {
		t.Fatal(err)
	}
	project := filepath.Join(t.TempDir(), "project with spaces")
	if err := os.MkdirAll(project, 0o700); err != nil {
		t.Fatal(err)
	}
	servers := InstalledMCPServers(project)
	if len(servers) != 5 {
		t.Fatalf("servers=%+v", servers)
	}
	if servers[0].Command != "resolve" || servers[0].Args[len(servers[0].Args)-1] != project {
		t.Fatalf("repository argument was not preserved literally: %+v", servers[0])
	}
	if servers[1].Command != "recall" || servers[1].Args[len(servers[1].Args)-1] != "project with spaces" {
		t.Fatalf("scope expansion failed: %+v", servers[1])
	}
	skill, err := NewSkillRegistry().Get(context.Background(), project, "graycode-workspace")
	if err != nil || skill.Content == "" {
		t.Fatalf("installed skill is unavailable: %+v %v", skill, err)
	}
	// Removal must affect both MCP and skill discovery on the next session.
	if err := Uninstall("graycode-workspace"); err != nil {
		t.Fatal(err)
	}
	if len(InstalledMCPServers(project)) != 0 {
		t.Fatal("uninstalled MCP servers survived")
	}
	if _, err := NewSkillRegistry().Get(context.Background(), project, "graycode-workspace"); err == nil {
		t.Fatal("uninstalled skill survived")
	}
}

func TestInstallFailurePreservesPreviousPackage(t *testing.T) {
	t.Setenv("REFLECT_STATE_DIR", t.TempDir())
	src := t.TempDir()
	manifest := filepath.Join(src, "plugin.json")
	if err := os.WriteFile(manifest, []byte(`{"name":"safe-package","version":"1.0.0","components":{"mcp":[{"name":"safe","command":"safe-mcp"}]}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Install(src); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifest, []byte(`{"name":"safe-package","version":"2.0.0","components":{"skills_dir":"../escape"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Install(src); err == nil {
		t.Fatal("component traversal accepted")
	}
	installed, err := LoadManifest(filepath.Join(pluginsDir(), "safe-package"))
	if err != nil || installed.Version != "1.0.0" {
		t.Fatalf("failed update replaced working package: %+v %v", installed, err)
	}
}

func TestComponentScopePriorityAndUnsupportedEnv(t *testing.T) {
	t.Setenv("REFLECT_STATE_DIR", t.TempDir())
	t.Setenv("REFLECT_CONFIG_DIR", t.TempDir())
	managed := t.TempDir()
	t.Setenv("REFLECT_MANAGED_PLUGINS", managed)
	for _, root := range []string{managed, pluginsDir()} {
		dir := filepath.Join(root, "demo")
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		command := "managed-mcp"
		if root == pluginsDir() {
			command = "user-mcp"
		}
		data := `{"name":"demo","version":"1.0.0","components":{"mcp":[{"name":"demo","command":"` + command + `"},{"name":"unsupported","command":"echo","env":{"PATH":"/other"}}]}}`
		if err := os.WriteFile(filepath.Join(dir, "plugin.json"), []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	servers := InstalledMCPServers(t.TempDir())
	if len(servers) != 1 || servers[0].Command != "managed-mcp" {
		t.Fatalf("scope shadowing/env handling: %+v", servers)
	}
}

func TestComponentPathsCannotEscapePackage(t *testing.T) {
	for _, field := range []string{"skills_dir", "hooks_dir", "mcp_file"} {
		t.Run(field, func(t *testing.T) {
			dir := t.TempDir()
			data := `{"name":"demo","version":"1.0.0","components":{"` + field + `":"../outside"}}`
			if err := os.WriteFile(filepath.Join(dir, "plugin.json"), []byte(data), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := DiscoverComponents(dir); err == nil || !strings.Contains(err.Error(), "escapes") {
				t.Fatalf("accepted escape: %v", err)
			}
		})
	}
}

func TestComponentProjectScopeRequiresTrust(t *testing.T) {
	t.Setenv("REFLECT_STATE_DIR", t.TempDir())
	t.Setenv("REFLECT_CONFIG_DIR", t.TempDir())
	t.Setenv("REFLECT_MANAGED_PLUGINS", "")
	t.Setenv("REFLECT_Y0_FOLDER_TRUST", "1")
	project := t.TempDir()
	dir := filepath.Join(project, ".reflect", "plugins", "local")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "plugin.json"), []byte(`{"name":"local","version":"1.0.0","components":{"mcp":[{"name":"local","command":"local-mcp"}]}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if len(InstalledMCPServers(project)) != 0 {
		t.Fatal("untrusted project MCP server discovered")
	}
	store, err := trust.Open("")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Trust(project, "test"); err != nil {
		t.Fatal(err)
	}
	if len(InstalledMCPServers(project)) != 1 {
		t.Fatal("trusted project MCP server missing")
	}
}

func TestInstallPreservesComponentFilesAndExecutableHelpers(t *testing.T) {
	t.Setenv("REFLECT_STATE_DIR", t.TempDir())
	src := t.TempDir()
	for path, content := range map[string]string{
		"plugin.json": `{"name":"file-package","version":"1.0.0"}`,
		"mcp.json":    `{"mcpServers":{"memory":{"command":"recall","args":["mcp","serve","--scope","project"]}}}`,
		"helper.sh":   "#!/bin/sh\nexit 0\n",
	} {
		if err := os.WriteFile(filepath.Join(src, path), []byte(content), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := Install(src); err != nil {
		t.Fatal(err)
	}
	installed := filepath.Join(pluginsDir(), "file-package")
	components, err := DiscoverComponents(installed)
	if err != nil || len(components.MCPServers) != 1 || components.MCPServers[0].Name != "memory" {
		t.Fatalf("component file lost: %+v %v", components, err)
	}
	info, err := os.Stat(filepath.Join(installed, "helper.sh"))
	if err != nil || info.Mode()&0o111 == 0 {
		t.Fatal("helper executable mode lost", err)
	}
	if err := os.Symlink(filepath.Join(src, "helper.sh"), filepath.Join(src, "escape")); err != nil {
		t.Skip("symlinks unavailable")
	}
	if err := Install(src); err == nil {
		t.Fatal("symlink-containing package installed")
	}
	if _, err := LoadManifest(installed); err != nil {
		t.Fatal("failed update damaged installed package", err)
	}
}
