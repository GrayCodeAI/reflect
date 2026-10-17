package hosttools

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEmpty(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("REFLECT_CONFIG_DIR", dir)
	// ensure dir exists
	os.MkdirAll(filepath.Join(dir, "host_tools"), 0o755)
	m, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(m) != 0 {
		t.Fatalf("expected 0, got %d", len(m))
	}
}

func TestLoadAndRun(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("REFLECT_CONFIG_DIR", dir)
	ht := filepath.Join(dir, "host_tools")
	os.MkdirAll(ht, 0o755)
	manifest := `{"name":"echo-tool","description":"echoes input","command":"sh","args":["-c","echo {input}"]}`
	os.WriteFile(filepath.Join(ht, "echo.json"), []byte(manifest), 0o644)
	m, err := Load()
	if err != nil || len(m) != 1 {
		t.Fatalf("load: %v len=%d", err, len(m))
	}
	out, err := Run(t.Context(), m[0], "hello")
	if err != nil {
		t.Fatal(err)
	}
	if out != "hello\n" {
		t.Fatalf("got %q", out)
	}
}
