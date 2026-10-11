package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStoragePolicyHelpersDoNotCreateProjectDecoder(t *testing.T) {
	project := t.TempDir()
	t.Setenv("DECODER_STATE_DIR", filepath.Join(t.TempDir(), "state"))
	t.Setenv("DECODER_CACHE_DIR", filepath.Join(t.TempDir(), "cache"))
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })
	if err := os.Chdir(project); err != nil {
		t.Fatal(err)
	}

	planPath := resolvePlanPath("demo")
	if strings.Contains(planPath, filepath.Join(project, ".decoder")) {
		t.Fatalf("resolvePlanPath leaked project .decoder: %q", planPath)
	}
	saveInputHistory([]string{"hello"})
	recordTipShown("slash-help")

	if _, err := os.Stat(filepath.Join(project, ".decoder")); !os.IsNotExist(err) {
		t.Fatalf("normal storage helpers created project .decoder, stat err=%v", err)
	}
}
