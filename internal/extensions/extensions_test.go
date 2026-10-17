package extensions

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEmpty(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("REFLECT_CONFIG_DIR", dir)
	os.MkdirAll(filepath.Join(dir, "extensions"), 0o755)
	exts, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(exts) != 0 {
		t.Fatalf("expected 0, got %d", len(exts))
	}
}

func TestLoadAndRun(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("REFLECT_CONFIG_DIR", dir)
	extDir := filepath.Join(dir, "extensions", "myext")
	os.MkdirAll(extDir, 0o755)
	os.WriteFile(filepath.Join(extDir, "hello.py"), []byte("import sys; print('hi ' + ' '.join(sys.argv[1:]))"), 0o644)
	manifest := `{"name":"myext","description":"test","commands":[{"name":"hello","description":"says hi","script":"hello.py"}]}`
	os.WriteFile(filepath.Join(extDir, "manifest.json"), []byte(manifest), 0o644)
	exts, err := Load()
	if err != nil || len(exts) != 1 {
		t.Fatalf("load: %v len=%d", err, len(exts))
	}
	out, err := RunCommand(t.Context(), exts[0], exts[0].Commands[0], []string{"world"})
	if err != nil {
		t.Fatal(err)
	}
	if out != "hi world\n" {
		t.Fatalf("got %q", out)
	}
}
