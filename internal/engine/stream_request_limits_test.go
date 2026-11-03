package engine

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/GrayCodeAI/reflect/internal/engine/lifecycle"
	"github.com/GrayCodeAI/reflect/internal/engine/safety"
	"github.com/GrayCodeAI/reflect/internal/tool"
	"github.com/GrayCodeAI/reflect/internal/types"
)

func requestLimitReply(t *testing.T, s *Session, prompt string) string {
	t.Helper()
	s.AddUser(prompt)
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	events, err := s.Stream(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	for event := range events {
		if event.Type == "error" {
			t.Errorf("request failed: %s", event.Content)
		}
		if event.Type == "content" {
			out.WriteString(event.Content)
		}
	}
	return out.String()
}

func TestNewRequestDoesNotInheritBashCommandCount(t *testing.T) {
	client := &conversationalStreamClient{resilienceBoundaryClient: &resilienceBoundaryClient{streams: [][]types.RelayStreamEvent{
		{{Type: "content", Content: "The repository has cmd and internal directories."}, {Type: "done", StopReason: "end_turn"}},
	}}}
	s := NewSessionWithClient(client, "test", "test-model", "", nil, false)
	for range 100 {
		s.LifecycleSvc().Limits().RecordToolCall("Bash")
	}
	if out := requestLimitReply(t, s, "How is this repository structured?"); !strings.Contains(out, "cmd and internal") || len(client.options) != 1 {
		t.Fatalf("previous requests blocked repository question: requests=%d reply=%q", len(client.options), out)
	}
}

func TestRequestToolLimitStopsCurrentRunAndAllowsFollowUp(t *testing.T) {
	client := &conversationalStreamClient{resilienceBoundaryClient: &resilienceBoundaryClient{streams: [][]types.RelayStreamEvent{
		{{Type: "tool_call", ToolCall: &types.ToolCall{ID: "read", Name: "Read", Arguments: map[string]interface{}{"id": 1}}}, {Type: "done", StopReason: "tool_use"}},
		{{Type: "content", Content: "Here is the repository structure."}, {Type: "done", StopReason: "end_turn"}},
	}}}
	s := NewSessionWithClient(client, "test", "test-model", "", tool.NewRegistry(&countedReadTool{}), false)
	s.LifecycleSvc().limits = lifecycle.NewLimitTracker(lifecycle.SafetyLimits{MaxToolCalls: 1})
	s.PermSvc().SetAutonomy(safety.AutonomyYOLO)
	first := requestLimitReply(t, s, "Inspect a source file")
	if !strings.Contains(first, "tool call limit reached (1/1)") || !strings.Contains(first, "follow-up") || len(client.options) != 1 {
		t.Fatalf("current request failed to enforce or explain its limit: requests=%d reply=%q", len(client.options), first)
	}
	second := requestLimitReply(t, s, "Explain the repository structure")
	if !strings.Contains(second, "Here is the repository structure.") || len(client.options) != 2 {
		t.Fatalf("follow-up remained blocked: requests=%d reply=%q", len(client.options), second)
	}
}
