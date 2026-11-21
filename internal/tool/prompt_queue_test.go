package tool

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GrayCodeAI/reflect/internal/storage"
)

// isolatePromptQueue redirects the state directory to a fresh temp dir and
// drops the in-memory queue cache so each test starts from the on-disk file.
func isolatePromptQueue(t *testing.T) {
	t.Helper()
	storage.SetTestDirs(t, t.TempDir())
	ResetPromptQueueForTest()
	t.Cleanup(ResetPromptQueueForTest)
	if err := os.MkdirAll(storage.StateDir(), 0o755); err != nil {
		t.Fatalf("mkdir state dir: %v", err)
	}
}

// TestPromptQueueEnqueueDequeueRoundtrip exercises every exported mutator in
// one FIFO sequence. Before the locking fix this deadlocked on the first
// call: the exported helpers re-acquired queueMu via LoadPromptQueue.
func TestPromptQueueEnqueueDequeueRoundtrip(t *testing.T) {
	isolatePromptQueue(t)

	if err := EnqueuePrompt("first task", "subject-one"); err != nil {
		t.Fatalf("enqueue first: %v", err)
	}
	if err := EnqueuePrompt("second task", ""); err != nil {
		t.Fatalf("enqueue second: %v", err)
	}

	items := GetPromptQueue()
	if len(items) != 2 {
		t.Fatalf("queue length = %d, want 2", len(items))
	}
	if items[0].Prompt != "first task" || items[0].Subject != "subject-one" {
		t.Errorf("items[0] = %+v, want first task with subject-one", items[0])
	}
	if items[1].Prompt != "second task" {
		t.Errorf("items[1] = %+v, want second task", items[1])
	}

	first, err := DequeuePrompt()
	if err != nil || first == nil {
		t.Fatalf("first dequeue = %v, err = %v", first, err)
	}
	if first.Prompt != "first task" {
		t.Errorf("first dequeue = %q, want first task", first.Prompt)
	}
	second, err := DequeuePrompt()
	if err != nil || second == nil {
		t.Fatalf("second dequeue = %v, err = %v", second, err)
	}
	if second.Prompt != "second task" {
		t.Errorf("second dequeue = %q, want second task", second.Prompt)
	}
	empty, err := DequeuePrompt()
	if err != nil || empty != nil {
		t.Fatalf("dequeue on empty queue = %v, err = %v; want nil, nil", empty, err)
	}
}

// TestPromptQueuePersistsAcrossReload proves items survive the in-memory
// cache being dropped (disk round-trip), not just the process lifetime.
func TestPromptQueuePersistsAcrossReload(t *testing.T) {
	isolatePromptQueue(t)

	if err := EnqueuePrompt("durable task", ""); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if _, err := os.Stat(filepath.Join(storage.StateDir(), "prompt_queue.json")); err != nil {
		t.Fatalf("queue file missing after enqueue: %v", err)
	}

	ResetPromptQueueForTest()
	if state := LoadPromptQueue(); state.Count() != 1 {
		t.Fatalf("after reload, count = %d, want 1", state.Count())
	}
	next, err := DequeuePrompt()
	if err != nil || next == nil || next.Prompt != "durable task" {
		t.Fatalf("dequeue after reload = %v, err = %v; want durable task", next, err)
	}
}

func TestPromptQueueRemoveAndClear(t *testing.T) {
	isolatePromptQueue(t)

	for _, p := range []string{"a", "b", "c"} {
		if err := EnqueuePrompt(p, ""); err != nil {
			t.Fatalf("enqueue %s: %v", p, err)
		}
	}
	if err := RemovePromptFromQueue(1); err != nil {
		t.Fatalf("remove: %v", err)
	}
	items := GetPromptQueue()
	if len(items) != 2 || items[0].Prompt != "a" || items[1].Prompt != "c" {
		t.Fatalf("after remove = %+v, want [a c]", items)
	}
	if err := ClearPromptQueue(); err != nil {
		t.Fatalf("clear: %v", err)
	}
	if got := len(GetPromptQueue()); got != 0 {
		t.Fatalf("after clear, length = %d, want 0", got)
	}
}

func TestPromptQueueRemoveInvalidIndexIsNoop(t *testing.T) {
	isolatePromptQueue(t)

	if err := EnqueuePrompt("keep", ""); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if err := RemovePromptFromQueue(5); err != nil {
		t.Fatalf("out-of-range index should be a no-op, got %v", err)
	}
	if err := RemovePromptFromQueue(-1); err != nil {
		t.Fatalf("negative index should be a no-op, got %v", err)
	}
	if got := len(GetPromptQueue()); got != 1 {
		t.Fatalf("length = %d, want 1", got)
	}
}

// TestPromptQueueMalformedFileStartsEmpty verifies a corrupt state file
// degrades to an empty, usable queue instead of hanging or poisoning the
// cache with a partially decoded state.
func TestPromptQueueMalformedFileStartsEmpty(t *testing.T) {
	isolatePromptQueue(t)

	if err := os.WriteFile(filepath.Join(storage.StateDir(), "prompt_queue.json"), []byte("{not json"), 0o600); err != nil {
		t.Fatalf("write malformed file: %v", err)
	}
	if state := LoadPromptQueue(); state.Count() != 0 {
		t.Fatalf("malformed file: count = %d, want 0", state.Count())
	}
	if err := EnqueuePrompt("after garbage", ""); err != nil {
		t.Fatalf("enqueue after malformed file: %v", err)
	}
	if got := len(GetPromptQueue()); got != 1 {
		t.Fatalf("length after recovery = %d, want 1", got)
	}
}
