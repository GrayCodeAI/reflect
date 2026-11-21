package cmd

import (
	"os"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/GrayCodeAI/reflect/internal/storage"
	"github.com/GrayCodeAI/reflect/internal/tool"
)

// isolateSteeringTestState isolates config/state dirs (including the
// persisted prompt queue) for steering tests.
func isolateSteeringTestState(t *testing.T) {
	t.Helper()
	isolateChatCommandSweepEnv(t)
	if err := os.MkdirAll(storage.StateDir(), 0o755); err != nil {
		t.Fatalf("mkdir state dir: %v", err)
	}
	tool.ResetPromptQueueForTest()
	t.Cleanup(tool.ResetPromptQueueForTest)
}

func TestSubmitWhileWaitingSteersActiveRun(t *testing.T) {
	m := newTestChatModel()
	m.waiting = true
	m.input.SetValue("focus on the failing test")

	result, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = requireChatModel(t, result)

	if len(m.messageQueue) != 0 {
		t.Fatalf("deferred queue should stay empty while steering is live, got %v", m.messageQueue)
	}
	sq := m.session.Persistence().Steering()
	if sq == nil || !sq.HasPending() {
		t.Fatal("steering text should be pending on the session's steering queue")
	}
	drained := sq.Drain()
	if len(drained) != 1 || drained[0].Content != "focus on the failing test" {
		t.Fatalf("steering queue = %+v, want one guidance message", drained)
	}
	if got := m.input.Value(); got != "" {
		t.Fatalf("input should be cleared after steering submit, got %q", got)
	}
	if got := lastSystemMessage(m.messages); !strings.Contains(got, "Steering active run: focus on the failing test") {
		t.Fatalf("expected steering status line, got %q", got)
	}
}

func TestSubmitWhileWaitingFallsBackWithoutSteeringQueue(t *testing.T) {
	m := newTestChatModel()
	m.session.Persistence().SetSteering(nil)
	m.waiting = true
	m.input.SetValue("deferred note")

	result, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = requireChatModel(t, result)

	if len(m.messageQueue) != 1 || m.messageQueue[0] != "deferred note" {
		t.Fatalf("fallback deferred queue = %v, want [deferred note]", m.messageQueue)
	}
	if got := lastSystemMessage(m.messages); !strings.Contains(got, "Queued: deferred note") {
		t.Fatalf("expected queued status line, got %q", got)
	}
}

// TestStreamDoneReclaimsUnconsumedSteering covers the conversational-run
// edge: a turn with no tool batches never drains the steering queue, so the
// done handler must hand the guidance back to the deferred queue instead of
// stranding it for an unrelated future run.
func TestStreamDoneReclaimsUnconsumedSteering(t *testing.T) {
	isolateSteeringTestState(t)

	m := newTestChatModel()
	m.waiting = true
	if !m.steerActiveRun("never drained") {
		t.Fatal("steering enqueue failed")
	}

	result, _ := m.Update(streamDoneMsg{})
	m = requireChatModel(t, result)

	sq := m.session.Persistence().Steering()
	if sq.HasPending() {
		t.Fatal("unconsumed steering should be reclaimed at stream end")
	}
	// The stream-done auto-submit path turns the reclaimed guidance into
	// the next user turn (same behavior the deferred queue always had).
	if !m.waiting {
		t.Fatal("reclaimed steering should auto-start the next turn")
	}
	if got := lastMessageByRole(m.messages, "user"); got != "never drained" {
		t.Fatalf("last user message = %q, want %q", got, "never drained")
	}
}

func TestStreamErrorReclaimsUnconsumedSteering(t *testing.T) {
	m := newTestChatModel()
	m.waiting = true
	if !m.steerActiveRun("aborted run guidance") {
		t.Fatal("steering enqueue failed")
	}

	result, _ := m.Update(streamErrMsg{err: errNoInteractivePromptInput})
	m = requireChatModel(t, result)

	if sq := m.session.Persistence().Steering(); sq.HasPending() {
		t.Fatal("unconsumed steering should be reclaimed on stream error")
	}
	if len(m.messageQueue) != 1 || m.messageQueue[0] != "aborted run guidance" {
		t.Fatalf("reclaimed queue = %v, want [aborted run guidance]", m.messageQueue)
	}
}

// TestPromptQueueAddSteersLiveRun: /prompt-queue add during a live run feeds
// the steering queue directly instead of persisting dead text.
func TestPromptQueueAddSteersLiveRun(t *testing.T) {
	isolateSteeringTestState(t)

	m := newTestChatModel()
	m.waiting = true

	m.handleCommand("/prompt-queue add focus on the auth module")

	sq := m.session.Persistence().Steering()
	if sq == nil || !sq.HasPending() {
		t.Fatal("/prompt-queue add during a live run should steer the run")
	}
	drained := sq.Drain()
	if len(drained) != 1 || drained[0].Content != "focus on the auth module" {
		t.Fatalf("steering queue = %+v, want the queued prompt", drained)
	}
	// Steered live: nothing is written to the persisted queue.
	if items := tool.GetPromptQueue(); len(items) != 0 {
		t.Fatalf("persisted queue = %+v, want empty", items)
	}
	if got := lastSystemMessage(m.messages); !strings.Contains(got, "Steered into the active run") {
		t.Fatalf("expected steering status line, got %q", got)
	}
}

// TestPromptQueueAddPersistsWhenIdle: with no live run, /prompt-queue add
// persists for the next run's steering splice.
func TestPromptQueueAddPersistsWhenIdle(t *testing.T) {
	isolateSteeringTestState(t)

	m := newTestChatModel()

	m.handleCommand("/prompt-queue add queued while idle")

	items := tool.GetPromptQueue()
	if len(items) != 1 || items[0].Prompt != "queued while idle" {
		t.Fatalf("persisted queue = %+v, want one queued prompt", items)
	}
	if sq := m.session.Persistence().Steering(); sq.HasPending() {
		t.Fatal("idle add must not enqueue steering")
	}
	if got := lastSystemMessage(m.messages); !strings.Contains(got, "steers the next run") {
		t.Fatalf("expected next-run status line, got %q", got)
	}
}

// TestPersistedPromptQueueFeedsNextRun: the run-start drain is the persisted
// queue's consumer — FIFO order is preserved and the file is emptied.
func TestPersistedPromptQueueFeedsNextRun(t *testing.T) {
	isolateSteeringTestState(t)

	if err := tool.EnqueuePrompt("first queued task", ""); err != nil {
		t.Fatal(err)
	}
	if err := tool.EnqueuePrompt("second queued task", ""); err != nil {
		t.Fatal(err)
	}

	m := newTestChatModel()
	m.drainPersistedPromptQueue()

	drained := m.session.Persistence().Steering().Drain()
	if len(drained) != 2 || drained[0].Content != "first queued task" || drained[1].Content != "second queued task" {
		t.Fatalf("steering feed = %+v, want FIFO [first queued task, second queued task]", drained)
	}
	if items := tool.GetPromptQueue(); len(items) != 0 {
		t.Fatalf("persisted queue should be drained, got %+v", items)
	}
}
