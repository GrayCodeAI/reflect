package chat

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GrayCodeAI/reflect/internal/storage"
)

func TestPlanDocumentReplacesOnlyThisConversation(t *testing.T) {
	storage.SetTestDirs(t, t.TempDir())
	project := t.TempDir()
	first, err := SavePlanDocument(project, "session-one", "# Original plan")
	if err != nil {
		t.Fatal(err)
	}
	other, err := SavePlanDocument(project, "session-two", "# Other plan")
	if err != nil {
		t.Fatal(err)
	}
	updated, err := SavePlanDocument(project, "session-one", "# Revised plan\n\n1. Test it.")
	if err != nil {
		t.Fatal(err)
	}
	if first != updated || first == other {
		t.Fatal("plan paths must be stable and conversation-scoped")
	}
	for path, want := range map[string]string{first: "# Revised plan\n\n1. Test it.\n", other: "# Other plan\n"} {
		got, err := os.ReadFile(path)
		if err != nil || string(got) != want {
			t.Fatalf("read %s = %q, %v", path, got, err)
		}
	}
	entries, err := os.ReadDir(project)
	if err != nil || len(entries) != 0 {
		t.Fatalf("planning changed source tree: %v, %v", entries, err)
	}
	entries, err = os.ReadDir(filepath.Dir(first))
	if err != nil || len(entries) != 1 || entries[0].Name() != "plan.md" {
		t.Fatalf("unexpected plan artifacts: %v, %v", entries, err)
	}
	if _, err := SavePlanDocument(project, "session-one", " \n"); err == nil {
		t.Fatal("empty response overwrote saved plan")
	}
	got, _ := os.ReadFile(first)
	if !strings.HasPrefix(string(got), "# Revised plan") {
		t.Fatal("invalid response replaced the plan")
	}
}

func TestPlanDocumentScopesProjectAndContainsSessionPath(t *testing.T) {
	storage.SetTestDirs(t, t.TempDir())
	project := t.TempDir()
	path, err := SavePlanDocument(project, "../../outside", "# Plan")
	if err != nil {
		t.Fatal(err)
	}
	rel, err := filepath.Rel(storage.PlansDir(project), path)
	if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		t.Fatalf("escaped plan directory: %q, %v", rel, err)
	}
	if path == PlanDocumentPath(t.TempDir(), "../../outside") {
		t.Fatal("different projects share a plan")
	}
	if _, err := SavePlanDocument(project, "", "# Plan"); err == nil {
		t.Fatal("accepted missing session")
	}
}
