package mission

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GrayCodeAI/reflect/internal/storage"
)

func TestLineageLoadAfterCleanup(t *testing.T) {
	m := New("explain lineage", Config{MaxWorkers: 2})
	dir, err := m.createDir()
	if err != nil {
		t.Fatal(err)
	}
	m.Dir = dir
	m.Features = []Feature{{ID: "f1", Description: "do x", Status: FeatureCompleted}}
	if err := m.persistState(); err != nil {
		t.Fatal(err)
	}
	if err := m.Cleanup(); err != nil {
		t.Fatal(err)
	}
	durable := filepath.Join(storage.StateDir(), "missions", m.ID)
	loaded, err := LoadMission(durable)
	if err != nil {
		t.Fatalf("load durable mission: %v", err)
	}
	got := loaded.LineageMarkdown()
	if !strings.Contains(got, "explain lineage") {
		t.Fatalf("lineage missing prompt: %s", got)
	}
	if _, err := os.Stat(filepath.Join(durable, "mission.json")); err != nil {
		t.Fatalf("mission.json not durable: %v", err)
	}
}
