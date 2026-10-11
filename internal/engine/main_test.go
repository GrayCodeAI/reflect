package engine

import (
	"os"
	"testing"

	"github.com/GrayCodeAI/decoder/internal/catalogtest"
	"github.com/GrayCodeAI/decoder/internal/testutil"
)

func TestMain(m *testing.M) {
	cleanupStorage, err := testutil.InstallHermeticStorage()
	if err != nil {
		os.Exit(1)
	}
	cleanup := catalogtest.InstallGlobal()
	code := m.Run()
	cleanup()
	cleanupStorage()
	os.Exit(code)
}
