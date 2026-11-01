package testaudit

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLocalRelayOverlayPreservesParentWorkspace(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bash development helper is covered on Linux and macOS")
	}
	root := t.TempDir()
	repo, relay := filepath.Join(root, "reflect"), filepath.Join(root, "relay with spaces")
	scriptDir := filepath.Join(repo, "scripts")
	for _, dir := range []string{scriptDir, relay} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	script, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "generate-workspace.sh"))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		filepath.Join(scriptDir, "generate-workspace.sh"): string(script),
		filepath.Join(repo, "go.mod"):                     "module example.com/fixture\n\ngo 1.27.2\n",
		filepath.Join(repo, "go.sum"):                     "",
		filepath.Join(relay, "go.mod"):                    "module github.com/GrayCodeAI/relay\n\ngo 1.27.2\n",
		filepath.Join(root, "go.work"):                    "parent workspace sentinel\n",
		filepath.Join(root, "go.work.sum"):                "parent checksums sentinel\n",
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	run := func(args ...string) error {
		cmd := exec.CommandContext(t.Context(), "bash", append([]string{filepath.Join(scriptDir, "generate-workspace.sh")}, args...)...)
		cmd.Dir = repo
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Logf("overlay helper: %s", output)
		}
		return err
	}
	if err := run(); err == nil {
		t.Fatal("optional overlay must require an explicit Relay checkout")
	}
	if err := run(relay); err != nil {
		t.Fatal(err)
	}
	overlayPath := filepath.Join(repo, "go.local.mod")
	overlay, err := os.ReadFile(overlayPath)
	if err != nil || !strings.Contains(string(overlay), relay) {
		t.Fatalf("local replacement missing: %s, %v", overlay, err)
	}
	if err := run(relay); err == nil {
		t.Fatal("existing local overlay was overwritten")
	}
	if after, _ := os.ReadFile(overlayPath); string(after) != string(overlay) {
		t.Fatal("existing overlay changed")
	}
	for _, name := range []string{"go.work", "go.work.sum"} {
		path := filepath.Join(root, name)
		if got, _ := os.ReadFile(path); string(got) != files[path] {
			t.Fatalf("parent %s was changed", name)
		}
	}
}
