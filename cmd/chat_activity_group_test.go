package cmd

import (
	"fmt"
	"strings"
	"testing"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/GrayCodeAI/reflect/internal/engine"
	sessionfeature "github.com/GrayCodeAI/reflect/internal/features/session"
	"github.com/GrayCodeAI/reflect/internal/session"
	"github.com/GrayCodeAI/reflect/internal/types"
)

func TestActivityGroupHidesRoutineCommandsAndOutput(t *testing.T) {
	m := newTestChatModel()
	m.sessionID = ""
	for i, name := range []string{"Read", "Read", "Grep", "Bash"} {
		id := fmt.Sprint(i)
		next, _ := m.handleToolUse(toolUseMsg{name: name, id: id, summary: "raw command or path"})
		*m = next.(chatModel)
		next, _ = m.handleToolResult(toolResultMsg{name: name, id: id, content: strings.Repeat("raw output\n", 200)})
		*m = next.(chatModel)
		incremental := m.assembleViewportContent(100)
		fresh := &chatModel{messages: m.messages}
		if full := fresh.assembleViewportContent(100); incremental != full {
			t.Fatal("growing activity group differs from a fresh render")
		}
	}
	got := stripANSI(m.assembleViewportContent(100))
	if !strings.Contains(got, "Read 2 files · 1 search · Ran 1 command") || strings.Count(got, "\n") != 2 {
		t.Fatalf("expected one activity row, got %q", got)
	}
	if strings.Contains(got, "raw") || len(m.messages) != 8 || !strings.Contains(m.messages[1].content, "raw output") {
		t.Fatal("presentation either leaked routine details or destroyed the transcript")
	}
	m.toolResultExpanded = map[int]bool{0: true}
	m.invalidateViewportCache()
	got = stripANSI(m.assembleViewportContent(100))
	if !strings.Contains(got, "raw command or path") || !strings.Contains(got, "raw output") || !strings.Contains(got, "Enter: collapse") {
		t.Fatal("activity details cannot be inspected")
	}
}

func TestResumedActivityRetainsDetailsAndIncompleteState(t *testing.T) {
	saved := &session.Session{Messages: []session.Message{
		{Role: "assistant", ToolUse: []types.ToolCall{{ID: "done", Name: "Bash", Arguments: map[string]interface{}{"command": "go test ./..."}}}},
		{Role: "user", ToolResults: []types.ToolResult{{ToolUseID: "done", Content: "compiler error", IsError: true}}},
		{Role: "assistant", ToolUse: []types.ToolCall{{ID: "unfinished", Name: "Read"}}},
	}}
	m := newTestChatModel()
	for _, message := range sessionfeature.Hydrate(saved).Display {
		m.messages = append(m.messages, displayFromHydrated(message))
	}
	got := stripANSI(m.assembleViewportContent(120))
	if !strings.Contains(got, "1 failed") || !strings.Contains(got, "1 incomplete") || strings.Contains(got, "running") {
		t.Fatalf("resumed activity claimed an unobserved outcome: %q", got)
	}
	m.toolResultExpanded = map[int]bool{0: true}
	m.invalidateViewportCache()
	got = stripANSI(m.assembleViewportContent(120))
	if !strings.Contains(got, "go test ./...") || !strings.Contains(got, "compiler error") {
		t.Fatal("restoration lost the command or its output")
	}
}

func TestActivityEventsAreIdempotentAndScopedToTurn(t *testing.T) {
	m := newTestChatModel()
	m.sessionID = ""
	start := toolUseMsg{name: "Read", id: "call", summary: "file.go"}
	finish := toolResultMsg{name: "Read", id: "call", content: "contents"}
	for range 2 {
		next, _ := m.handleToolUse(start)
		*m = next.(chatModel)
		next, _ = m.handleToolResult(finish)
		*m = next.(chatModel)
	}
	if len(m.messages) != 2 || m.messages[0].toolState != engine.ToolStateCompleted {
		t.Fatalf("replayed events duplicated or reopened a call: %#v", m.messages)
	}
	m.messages = append(m.messages, displayMsg{role: "user", content: "again"})
	next, _ := m.handleToolUse(start)
	*m = next.(chatModel)
	if len(m.messages) != 4 || m.messages[3].toolState != engine.ToolStateValidating {
		t.Fatal("a provider reusing an ID on another turn lost a call")
	}
}

func TestActivityGroupingPreservesInteractiveBoundaries(t *testing.T) {
	messages := []displayMsg{
		{role: "tool_use", content: "Bash", toolID: "one", toolState: engine.ToolStateCompleted},
		{role: "permission", content: "Run tests?"},
		{role: "tool_result", toolID: "one", toolName: "Bash", content: "passed"},
		{role: "warning", content: "A warning"},
		{role: "assistant", content: "The fix is ready."},
	}
	got := stripANSI(renderMessagesRange(messages, 0, len(messages), 100, nil))
	for _, want := range []string{"Run tests?", "A warning", "The fix is ready."} {
		if !strings.Contains(got, want) {
			t.Fatalf("activity swallowed %q: %s", want, got)
		}
	}
	if strings.Count(got, "details") != 2 || strings.Contains(got, "passed") {
		t.Fatal("activity blocks crossed an interactive boundary")
	}
}

func TestActivityDetailsToggleTargetsVisibleGroup(t *testing.T) {
	m := newTestChatModel()
	m.width = 80
	m.viewport = viewport.New(viewport.WithWidth(80), viewport.WithHeight(1))
	m.messages = []displayMsg{
		{role: "tool_use", content: "Read", toolState: engine.ToolStateCompleted},
		{role: "tool_result", content: strings.Repeat("hidden\n", 50)},
		{role: "assistant", content: "Next check."},
		{role: "tool_use", content: "Bash", toolState: engine.ToolStateCompleted},
		{role: "tool_result", content: "second output"},
	}
	m.viewport.SetContent(m.assembleViewportContent(80))
	prefix := renderMessagesRange(m.messages, 0, 3, 80, nil)
	m.viewport.SetYOffset(strings.Count(prefix, "\n"))
	m.uiFocus = focusScrollback
	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	*m = next.(chatModel)
	if !m.toolResultExpanded[3] || m.toolResultExpanded[0] {
		t.Fatal("Enter expanded the wrong group")
	}
	if got := stripANSI(m.assembleViewportContent(80)); !strings.Contains(got, "second output") || strings.Contains(got, "hidden") {
		t.Fatalf("incorrect details shown: %q", got)
	}
}

func TestActivityFailuresStayVisibleAtNarrowWidths(t *testing.T) {
	messages := []displayMsg{
		{role: "tool_use", content: "Bash", toolID: "call", toolState: engine.ToolStateFailed},
		{role: "tool_result", toolID: "call", content: "start\n\x1b[2Jcompile failed\u202e", toolState: engine.ToolStateFailed},
	}
	for _, width := range []int{16, 40, 80} {
		got := stripANSI(renderMessagesRange(messages, 0, 2, width, nil))
		if !strings.Contains(got, "1 failed") || strings.ContainsAny(got, "\x1b\u202e") {
			t.Fatalf("failure hidden or unsafe at width %d: %q", width, got)
		}
		for _, line := range strings.Split(got, "\n") {
			if visibleWidth(line) > width {
				t.Fatalf("overflow at width %d: %q", width, line)
			}
		}
	}
}

func TestTrimKeepsActivityMetadata(t *testing.T) {
	m := newTestChatModel()
	for i := range messageTrimThreshold + 2 {
		m.messages = append(m.messages, displayMsg{role: "system", content: fmt.Sprint(i)})
	}
	m.messages = append(m.messages, displayMsg{role: "tool_use", content: "Read", toolID: "kept", summary: "file.go", toolState: engine.ToolStateFailed})
	m.trimOldMessages()
	last := m.messages[len(m.messages)-1]
	if last.toolID != "kept" || last.toolState != engine.ToolStateFailed || last.summary != "file.go" {
		t.Fatalf("trim discarded activity metadata: %#v", last)
	}
}

func TestShortActivityCanBeExpandedWithoutScrolling(t *testing.T) {
	m := newTestChatModel()
	m.messages = []displayMsg{{role: "tool_result", content: "inspectable output"}}
	m.viewport.SetContent(m.assembleViewportContent(m.width))
	m.uiFocus = focusScrollback
	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	*m = next.(chatModel)
	if !m.toolResultExpanded[0] || !strings.Contains(stripANSI(m.assembleViewportContent(m.width)), "inspectable output") {
		t.Fatal("a group above the center of a short transcript cannot be inspected")
	}
}

func TestRetryRetainsActivityIdentityAndOutcome(t *testing.T) {
	m := newTestChatModel()
	m.messages = []displayMsg{
		{role: "user", content: "check file"},
		{role: "thinking", content: "private reasoning"},
		{role: "tool_use", content: "Read", summary: "file.go", toolID: "call", toolState: engine.ToolStateFailed},
		{role: "tool_result", toolID: "call", content: "missing file", toolState: engine.ToolStateFailed},
	}
	_ = m.assembleViewportContent(m.width)
	next, _ := m.handleStreamRetry(streamRetryMsg{content: "Retrying"})
	*m = next.(chatModel)
	if m.messages[1].toolID != "call" || m.messages[1].toolState != engine.ToolStateFailed || m.messages[1].summary != "file.go" {
		t.Fatal("retry lost activity metadata")
	}
	if got := stripANSI(m.assembleViewportContent(m.width)); !strings.Contains(got, "1 failed") || strings.Contains(got, "private reasoning") {
		t.Fatalf("retry projection is stale: %q", got)
	}
}

type activityTestWAL struct{ messages []session.Message }

func (w *activityTestWAL) Append(message session.Message) error {
	w.messages = append(w.messages, message)
	return nil
}
func (w *activityTestWAL) Remove() error { return nil }
func (w *activityTestWAL) Close() error  { return nil }

func TestActivityPersistsIdentifiedFailureExactlyOnce(t *testing.T) {
	m := newTestChatModel()
	wal := &activityTestWAL{}
	m.wal = wal
	result := toolResultMsg{id: "call", name: "Bash", content: "partial diagnostics\ncheck failed", state: engine.ToolStateFailed}
	for range 2 {
		next, _ := m.handleToolResult(result)
		*m = next.(chatModel)
	}
	if len(wal.messages) != 1 || len(wal.messages[0].ToolResults) != 1 {
		t.Fatalf("expected one durable observation, got %#v", wal.messages)
	}
	if wal.messages[0].Role != "user" || wal.messages[0].Content != "" {
		t.Fatal("durable result must use the runtime message contract without duplicating its body")
	}
	output := wal.messages[0].ToolResults[0]
	if output.ToolUseID != "call" || !output.IsError || output.Content != result.content {
		t.Fatalf("durability lost identity, failure, or full output: %#v", output)
	}
}
