package cmd

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/viewport"
)

func TestCompactToolResultKeepsFullOutputExpandable(t *testing.T) {
	raw := "first\nsecond\nhidden-a\nhidden-b\nlast"
	got := stripANSI(collapseToolResult(raw, 0, nil, false))
	if !strings.Contains(got, "first\nsecond") || strings.Contains(got, "hidden-a") || !strings.Contains(got, "Enter to expand") {
		t.Fatalf("verbose or unusable preview: %q", got)
	}
	if expanded := collapseToolResult(raw, 0, map[int]bool{0: true}, false); expanded != raw {
		t.Fatalf("expanded output lost content: %q", expanded)
	}
	failed := stripANSI(collapseToolResult(raw, 0, nil, true))
	if !strings.Contains(failed, "last") {
		t.Fatalf("failure preview hid the diagnostic tail: %q", failed)
	}
}

func TestCollapsedToolResultHitTestingUsesVisibleRows(t *testing.T) {
	m := &chatModel{width: 80, viewport: viewport.New(viewport.WithWidth(80), viewport.WithHeight(1))}
	m.messages = []displayMsg{
		{role: "tool_result", content: strings.Repeat("first result\n", 50)},
		{role: "tool_result", content: strings.Repeat("second result\n", 10)},
	}
	prefix := renderDisplayMessage(m.messages[0], 0, m.messages, 80, nil)
	m.viewport.SetContent(renderMessagesRange(m.messages, 0, len(m.messages), 80, nil))
	m.viewport.SetYOffset(strings.Count(prefix, "\n"))
	if got := m.toolResultIndexAtViewportCenter(80); got != 1 {
		t.Fatalf("Enter targets message %d, want visible second result", got)
	}
}

func TestCodeCopyAfterCollapsedToolUsesVisibleRows(t *testing.T) {
	m := &chatModel{width: 80, viewport: viewport.New(viewport.WithWidth(80), viewport.WithHeight(1))}
	m.messages = []displayMsg{
		{role: "tool_result", content: strings.Repeat("long result\n", 50)},
		{role: "assistant", content: "Example:\n\n```go\nfmt.Println(\"hello\")\n```"},
	}
	prefix := renderDisplayMessage(m.messages[0], 0, m.messages, 80, nil)
	m.viewport.SetContent(renderMessagesRange(m.messages, 0, len(m.messages), 80, nil))
	m.viewport.SetYOffset(strings.Count(prefix, "\n") + 2)
	if code, ok := m.codeBlockAtViewportCenter(); !ok || !strings.Contains(code, "fmt.Println") {
		t.Fatalf("code copy missed visible block: %q, %v", code, ok)
	}
}
