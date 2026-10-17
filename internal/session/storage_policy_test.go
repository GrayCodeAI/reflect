package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSessionPathDoesNotCreateProjectReflect(t *testing.T) {
	project := t.TempDir()
	state := filepath.Join(t.TempDir(), "state")
	t.Setenv("REFLECT_STATE_DIR", state)

	path := SessionPath(project, "abc123")
	if strings.Contains(path, filepath.Join(project, ".reflect")) {
		t.Fatalf("SessionPath leaked project .reflect: %q", path)
	}
	if !strings.HasPrefix(path, state) {
		t.Fatalf("SessionPath = %q, want under %q", path, state)
	}
	if _, err := os.Stat(filepath.Join(project, ".reflect")); !os.IsNotExist(err) {
		t.Fatalf("SessionPath created project .reflect, stat err=%v", err)
	}
}
