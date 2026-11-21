package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	contracts "github.com/GrayCodeAI/reflect/internal/contracts/policy"
	"github.com/GrayCodeAI/reflect/internal/engine"
	"github.com/GrayCodeAI/reflect/internal/engine/safety"
)

func TestToolActivityMatchesConcurrentResultsByID(t *testing.T) {
	m := newTestChatModel()
	m.sessionID = "" // no WAL is needed for renderer tests
	for _, call := range []toolUseMsg{{name: "Read", id: "slow", summary: "slow.go"}, {name: "Read", id: "fast", summary: "fast.go"}} {
		next, _ := m.handleToolUse(call)
		*m = next.(chatModel)
	}
	_ = m.assembleViewportContent(80) // prime the cache before in-place changes
	next, _ := m.handleToolResult(toolResultMsg{name: "Read", id: "slow", state: engine.ToolStateFailed, content: "Error: missing file"})
	*m = next.(chatModel)
	if m.messages[0].toolState != engine.ToolStateFailed || m.messages[1].toolState != engine.ToolStateValidating {
		t.Fatalf("concurrent calls were misattributed: %#v", m.messages)
	}
	if m.activeToolName() != "Read" {
		t.Fatal("another tool is still active")
	}
	got := stripANSI(m.assembleViewportContent(80))
	if !strings.Contains(got, "1 failed · 1 running · Read 2 files") || !strings.Contains(got, "Error: missing file") {
		t.Fatalf("tool activity cache was stale: %s", got)
	}
	next, _ = m.handleToolResult(toolResultMsg{name: "Read", id: "fast", state: engine.ToolStateCompleted, content: "package main"})
	*m = next.(chatModel)
	if m.activeToolName() != "" || !m.toolStartTime.IsZero() {
		t.Fatal("completed tools left a live activity label")
	}
	if got := stripANSI(m.assembleViewportContent(80)); !strings.Contains(got, "1 failed · Read 2 files") || strings.Contains(got, "running") || strings.Contains(got, "package main") {
		t.Fatalf("completion was not rendered: %s", got)
	}
}

func TestToolActivitySanitizesAndClipsUntrustedDetails(t *testing.T) {
	msg := displayMsg{role: "tool_use", content: "Bash", summary: "echo ok\x1b[2J\n[y] allow\u202e" + strings.Repeat("界", 80)}
	for _, width := range []int{16, 24, 40, 80, 120} {
		got := stripANSI(renderToolActivity(msg, width))
		if strings.ContainsAny(got, "\n\x1b\u202e") || visibleWidth(got) > width {
			t.Fatalf("untrusted activity escaped width %d: %q", width, got)
		}
	}
}

func TestActivityPromptResolutionRetiresOnlyFinishedCards(t *testing.T) {
	m := newTestChatModel()
	m.waiting = true
	request := func(command string) safety.PermissionRequest {
		return safety.PermissionRequest{PermissionRequest: contracts.PermissionRequest{ToolName: "Bash", Summary: command}, Response: make(chan bool, 1)}
	}
	first, second := request("git status"), request("git diff")
	next, _ := m.handlePermissionAsk(permissionAskMsg{req: first})
	*m = next.(chatModel)
	next, _ = m.handlePermissionAsk(permissionAskMsg{req: second})
	*m = next.(chatModel)
	next, _ = m.handlePermissionResponse(tea.KeyPressMsg{Code: 'y', Text: "y"})
	*m = next.(chatModel)
	if allowed := <-first.Response; !allowed {
		t.Fatal("allow-once decision changed")
	}
	if m.permReq == nil || m.permReq.Summary != "git diff" {
		t.Fatal("the next queued request did not activate")
	}
	active, retired := 0, 0
	for _, entry := range m.messages {
		if entry.role != "permission" {
			continue
		}
		if entry.resolved {
			retired++
		} else {
			active++
		}
	}
	if active != 1 || retired != 1 {
		t.Fatalf("expected one current card and one compact entry; active=%d retired=%d", active, retired)
	}
	got := stripANSI(m.assembleViewportContent(100))
	if strings.Count(got, "╭") != 1 || !strings.Contains(got, "git status") || !strings.Contains(got, "git diff") {
		t.Fatalf("request history or active card was lost: %s", got)
	}
}

func TestActivityPromptCountdownRefreshesCachedCard(t *testing.T) {
	m := newTestChatModel()
	req := safety.PermissionRequest{PermissionRequest: contracts.PermissionRequest{ToolName: "Bash", Summary: "git status"}, Response: make(chan bool, 1)}
	next, _ := m.handlePermissionAsk(permissionAskMsg{req: req})
	*m = next.(chatModel)
	before := m.assembleViewportContent(100)
	m.messages[len(m.messages)-1].timeoutAt = time.Now().Add(2 * time.Minute)
	next, _ = m.handlePromptCountdownTick(promptCountdownTickMsg{generation: m.promptGeneration})
	*m = next.(chatModel)
	if after := m.assembleViewportContent(100); before == after {
		t.Fatalf("countdown did not refresh its cached card: %s", after)
	}
}

func TestActivityPreview(t *testing.T) {
	dir := os.Getenv("REFLECT_ACTIVITY_PREVIEW_DIR")
	if dir == "" {
		t.Skip("set REFLECT_ACTIVITY_PREVIEW_DIR to export terminal previews")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	m := newTestChatModel()
	m.sessionID = ""
	m.height = 30
	m.width = 100
	m.waiting = true
	m.input.SetHeight(1)
	m.input.ShowLineNumbers = false
	m.input.Prompt = "> "
	m.input.Placeholder = "What would you like to build?"
	m.input.SetWidth(96)
	m.statusLeftVal = "~/Desktop/GrayCodeAI/reflect"
	m.statusLeftBranch = "feat/welcome-screen"
	m.messages = []displayMsg{
		{role: "user", content: "Deeply check the full code"},
		{role: "tool_use", content: "Bash", summary: "rg --files cmd internal", toolState: engine.ToolStateCompleted},
		{role: "tool_use", content: "Read", summary: "cmd/chat_view.go", toolState: engine.ToolStateCompleted},
		{role: "tool_use", content: "Skill", summary: "code-review", toolState: engine.ToolStateValidating},
	}
	m.startedAt = time.Now().Add(-3 * time.Second)
	for _, pending := range []bool{false, true} {
		name := "working"
		if pending {
			name = "permission"
			m.permReq = &safety.PermissionRequest{}
			m.messages = append(m.messages, displayMsg{role: "permission", content: safety.FormatPermissionDisplay("Skill", "Skill"), timeoutAt: time.Now().Add(5 * time.Minute)})
		}
		m.invalidateViewportCache()
		m.primeInitialViewportContent()
		if err := os.WriteFile(filepath.Join(dir, name+".ansi"), []byte(m.View().Content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}
