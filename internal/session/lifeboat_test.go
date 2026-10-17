package session

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GrayCodeAI/reflect/internal/types"
)

func TestLifeboatRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("REFLECT_STATE_DIR", tmp)

	id := "test-lifeboat"
	state := NewLifeboatState(id)
	state.ActiveTask = "fix billing crash"
	state.RecentDecisions = []string{"use postgres", "cache 5m"}
	state.OpenQuestions = []string{"migrate or drop column?"}
	state.WorkingFiles = []string{"/repo/billing.go"}
	state.MessageCount = 42

	path, err := SaveLifeboat(id, state)
	if err != nil {
		t.Fatalf("SaveLifeboat: %v", err)
	}
	if !filepath.IsAbs(path) {
		t.Fatalf("path not absolute: %s", path)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file missing: %v", err)
	}

	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode=%o want 600", info.Mode().Perm())
	}

	got, err := LoadLifeboat(id)
	if err != nil {
		t.Fatalf("LoadLifeboat: %v", err)
	}
	if got.SessionID != id || got.ActiveTask != state.ActiveTask {
		t.Fatalf("mismatch: %+v", got)
	}
	if len(got.RecentDecisions) != 2 || got.RecentDecisions[1] != "cache 5m" {
		t.Fatalf("decisions: %+v", got.RecentDecisions)
	}
	if len(got.OpenQuestions) != 1 {
		t.Fatalf("questions: %+v", got.OpenQuestions)
	}
	if got.MessageCount != 42 {
		t.Fatalf("count=%d want 42", got.MessageCount)
	}
}

func TestLifeboatMissing(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("REFLECT_STATE_DIR", tmp)
	got, err := LoadLifeboat("never-saved")
	if err != ErrNotFound {
		t.Fatalf("want ErrNotFound got %v", err)
	}
	if got.SessionID != "" || got.ActiveTask != "" || got.Version != 0 || len(got.RecentDecisions) != 0 {
		t.Fatalf("expected zero state, got %+v", got)
	}
}

func TestLifeboatValidation(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("REFLECT_STATE_DIR", tmp)
	if _, err := SaveLifeboat("../escape", NewLifeboatState("x")); err == nil {
		t.Fatal("expected invalid id error")
	}
	if _, err := LifeboatPath("good-id"); err != nil {
		t.Fatalf("valid id rejected: %v", err)
	}
	has, err := HasLifeboat("none")
	if err != nil || has {
		t.Fatalf("HasLifeboat: %v %v", has, err)
	}
	ok := RemoveLifeboat("never-existed")
	if ok != nil {
		t.Fatalf("RemoveLifeboat missing: %v", ok)
	}
}

func TestLifeboatBuildHeuristics(t *testing.T) {
	msgs := []types.RelayMessage{
		{Role: "user", Content: "please fix billing"},
		{Role: "assistant", Content: "Decision: use postgres\nWill use cache 5m\nOpen question: migrate or drop column?"},
		{Role: "assistant", Content: "Reading touched /repo/billing.go and /repo/x.md"},
	}
	state := BuildLifeboat("sid-1", msgs)
	if state.SessionID != "sid-1" {
		t.Fatalf("session id mismatch")
	}
	if state.MessageCount != len(msgs) {
		t.Fatalf("count mismatch")
	}
	if state.ActiveTask != "please fix billing" {
		t.Fatalf("active task=%q", state.ActiveTask)
	}
	if len(state.RecentDecisions) < 1 {
		t.Fatalf("decisions=%v", state.RecentDecisions)
	}
	if len(state.OpenQuestions) != 1 {
		t.Fatalf("questions=%v", state.OpenQuestions)
	}
	// working file extraction is best-effort; we just require no panic.
	_ = state.WorkingFiles
}

func TestLifeboatFutureVersionRejected(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("REFLECT_STATE_DIR", tmp)
	id := "ver-future"
	path, _ := LifeboatPath(id)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	data := `{"version":999,"session_id":"` + id + `","active_task":"x"}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := LoadLifeboat(id)
	if err == nil {
		t.Fatal("expected version error")
	}
}
