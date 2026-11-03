package engine

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/GrayCodeAI/reflect/internal/engine/branching"
	"github.com/GrayCodeAI/reflect/internal/engine/control"
	"github.com/GrayCodeAI/reflect/internal/engine/safety"
	"github.com/GrayCodeAI/reflect/internal/tool"
	"github.com/GrayCodeAI/reflect/internal/types"
)

func TestReviewContinuesBeyondOldCumulativeTokenCeiling(t *testing.T) {
	client := &conversationalStreamClient{resilienceBoundaryClient: &resilienceBoundaryClient{}}
	for i := range 3 {
		client.streams = append(client.streams, []types.RelayStreamEvent{
			{Type: "tool_call", ToolCall: &types.ToolCall{ID: fmt.Sprint(i), Name: "Read", Arguments: map[string]interface{}{"id": i}}},
			{Type: "done", StopReason: "tool_use", Usage: &types.RelayUsage{PromptTokens: 200000, CompletionTokens: 10}},
		})
	}
	client.streams = append(client.streams, []types.RelayStreamEvent{{Type: "content", Content: "Review complete."}, {Type: "done", StopReason: "end_turn"}})
	s := NewSessionWithClient(client, "test", "test-model", "", tool.NewRegistry(&countedReadTool{}), false)
	s.PermSvc().SetAutonomy(safety.AutonomyYOLO)
	s.AddUser("Review the code")
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	events, err := s.Stream(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var reply strings.Builder
	for event := range events {
		if event.Type == "error" {
			t.Errorf("review failed: %s", event.Content)
		}
		if event.Type == "content" {
			reply.WriteString(event.Content)
		}
	}
	if !strings.Contains(reply.String(), "Review complete.") || len(client.options) != 4 {
		t.Fatalf("review stopped early after %d requests: %s", len(client.options), reply.String())
	}
}

func TestReviewStopExplainsReasonInCLI(t *testing.T) {
	s := newMockSession(newMockClient())
	detector := branching.NewSnowballDetector(100)
	detector.RecordTurn(101, 1)
	ch := make(chan StreamEvent, 2)
	if s.checkGuardConditions(t.Context(), ch, 0, detector, control.NewLoopDetector(10, control.DoomLoopThreshold)) {
		t.Fatal("explicit token limit not enforced")
	}
	event := <-ch
	if event.Type != "content" || !strings.Contains(event.Content, "Token limit reached") || !strings.Contains(event.Content, "follow-up") || strings.Contains(event.Content, "Recommendation:") || strings.Contains(event.Content, "fresh context") {
		t.Fatalf("unhelpful stop message: %+v", event)
	}
	if event := <-ch; event.Type != "done" {
		t.Fatalf("missing terminal event: %+v", event)
	}
}
