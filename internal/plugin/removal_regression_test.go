package plugin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUninstallRejectsTraversalAndEmptyNames(t *testing.T) {
	root := t.TempDir()
	plugins, other := filepath.Join(root, "plugins"), filepath.Join(root, "other")
	for _, path := range []string{plugins, other} {
		if err := os.MkdirAll(path, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	manager := NewDynamicPluginManager([]string{plugins}, nil, nil)
	for _, name := range []string{"../other", "..", ".", "", other, "..\\" + "other"} {
		if err := manager.Uninstall(name); err == nil {
			t.Errorf("dynamic uninstall accepted %q", name)
		}
		if err := Uninstall(name); err == nil {
			t.Errorf("uninstall accepted %q", name)
		}
	}
	if _, err := os.Stat(other); err != nil {
		t.Fatalf("sibling was removed: %v", err)
	}
}
