package storage

import (
	"path/filepath"
	"testing"
)

// SetTestDirs configures isolated Decoder config/state/cache directories for tests.
func SetTestDirs(t *testing.T, root string) {
	t.Helper()
	t.Setenv(envConfigDir, filepath.Join(root, "config"))
	t.Setenv(envMuxConfigDir, filepath.Join(root, "mux"))
	t.Setenv(envStateDir, filepath.Join(root, "state"))
	t.Setenv(envCacheDir, filepath.Join(root, "cache"))
}
