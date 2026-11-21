package engine

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/GrayCodeAI/reflect/internal/engine/safety"
	"github.com/GrayCodeAI/reflect/internal/tool"
	"github.com/GrayCodeAI/reflect/internal/types"
)

func TestToolActivitySummariesExposeActionFieldsOnly(t *testing.T) {
	for _, tc := range []struct {
		name string
		args map[string]interface{}
		want string
	}{
		{"Read", map[string]interface{}{"file_path": "cmd/main.go"}, "cmd/main.go"},
		{"Bash", map[string]interface{}{"command": "git status"}, "git status"},
		{"Skill", map[string]interface{}{"skill": "code-review"}, "code-review"},
		{"Write", map[string]interface{}{"path": "main.go", "content": "private body"}, "main.go"},
		{"Unknown", map[string]interface{}{"token": "secret", "content": "private body"}, ""},
	} {
		if got := ToolActivitySummary(tc.name, tc.args); got != tc.want {
			t.Errorf("%s: got %q want %q", tc.name, got, tc.want)
		}
	}
}

type cancelledActivityTool struct{ orderedReadTool }

func (cancelledActivityTool) Execute(context.Context, json.RawMessage) (string, error) {
	return "", context.Canceled
}

func TestToolActivityPreservesCancellationThroughPostProcess(t *testing.T) {
	sess := NewSession("test", "test", "system", tool.NewRegistry(cancelledActivityTool{}))
	sess.PermSvc().SetAutonomy(safety.AutonomyYOLO)
	ch := make(chan StreamEvent, 16)
	sess.executeSingleTool(context.Background(), types.ToolCall{ID: "cancelled", Name: "Read"}, ch, 0, "test")
	for len(ch) > 0 {
		event := <-ch
		if event.Type == "tool_result" {
			if event.ToolState != ToolStateCancelled || event.ToolReason != ToolReasonCancelled {
				t.Fatalf("cancelled call became a success: %#v", event)
			}
			return
		}
	}
	t.Fatal("missing cancellation event")
}

func TestToolActivityEventsKeepIDsAndRedactHeadings(t *testing.T) {
	sess := NewSession("test", "test", "system", tool.NewRegistry(orderedReadTool{}))
	sess.PermSvc().SetAutonomy(safety.AutonomyYOLO)
	sess.Tools().deps.redactOutput = func(s string) string { return strings.ReplaceAll(s, "private.go", "[redacted]") }
	ch := make(chan StreamEvent, 16)
	sess.executeSingleTool(context.Background(), types.ToolCall{ID: "read-1", Name: "Read", Arguments: map[string]interface{}{"id": 1, "file_path": "private.go"}}, ch, 0, "test")
	seenUse, seenResult := false, false
	for len(ch) > 0 {
		event := <-ch
		switch event.Type {
		case "tool_use":
			seenUse = true
			if event.ToolID != "read-1" || event.Content != "[redacted]" {
				t.Fatalf("unsafe or unpaired heading: %#v", event)
			}
		case "tool_result":
			seenResult = true
			if event.ToolID != "read-1" || event.ToolState != ToolStateCompleted {
				t.Fatalf("unpaired terminal event: %#v", event)
			}
		}
	}
	if !seenUse || !seenResult {
		t.Fatal("missing lifecycle events")
	}
}
