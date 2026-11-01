package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/GrayCodeAI/reflect/internal/engine/safety"
	"github.com/GrayCodeAI/reflect/internal/storage"
	"github.com/GrayCodeAI/reflect/internal/tool"
	"github.com/GrayCodeAI/reflect/internal/types"
)

type batchLifecycleReadTool struct{ orderedReadTool }

func (batchLifecycleReadTool) Execute(_ context.Context, input json.RawMessage) (string, error) {
	var args struct {
		Fail bool `json:"fail"`
	}
	_ = json.Unmarshal(input, &args)
	if args.Fail {
		return "", fmt.Errorf("read failed")
	}
	return "private-answer", nil
}

func TestBatchLifecycleFinishesEveryCallOnceAndRedactsResults(t *testing.T) {
	storage.SetTestDirs(t, t.TempDir())
	sess := NewSession("test", "test", "system", tool.NewRegistry(batchLifecycleReadTool{}))
	sess.PermSvc().SetAutonomy(safety.AutonomyYOLO)
	service := sess.Tools()
	service.deps.redactOutput = func(value string) string { return strings.ReplaceAll(value, "private-answer", "[redacted]") }
	verified := map[string]int{}
	service.deps.recordVerification = func(call types.ToolCall, _ string, _ bool) { verified[call.ID]++ }
	postProcessed := 0
	service.Pipeline().Register(tool.StagePostExecute, tool.InterceptFn(func(_ context.Context, _ tool.ToolRequest, result *tool.ToolResult, next func() error) error {
		postProcessed++
		result.Output += " [postprocessed]"
		return next()
	}))
	calls := []types.ToolCall{{ID: "ok", Name: "Read"}, {ID: "bad", Name: "Read", Arguments: map[string]interface{}{"fail": true}}}
	ch := make(chan StreamEvent, 16)
	results := service.ExecuteAll(context.Background(), calls, ch, 0, "inspect")
	if len(results) != 2 || results[0].isErr || !results[1].isErr || postProcessed != 2 {
		t.Fatalf("batch skipped result processing: %#v; postprocessed=%d", results, postProcessed)
	}
	terminal := map[string]int{}
	for len(ch) > 0 {
		event := <-ch
		if event.Type != "tool_result" {
			continue
		}
		terminal[event.ToolID]++
		if strings.Contains(event.Content, "private-answer") || !strings.Contains(event.Content, "[postprocessed]") {
			t.Fatalf("unsafe or unprocessed event: %#v", event)
		}
		want := ToolStateCompleted
		if event.ToolID == "bad" {
			want = ToolStateFailed
		}
		if event.ToolState != want {
			t.Fatalf("%s: state %s, want %s", event.ToolID, event.ToolState, want)
		}
	}
	for _, call := range calls {
		if terminal[call.ID] != 1 || verified[call.ID] != 1 {
			t.Fatalf("%s: %d terminal events, %d completion hooks", call.ID, terminal[call.ID], verified[call.ID])
		}
	}
	if results[0].output != "[redacted] [postprocessed]" {
		t.Fatalf("model result not normalized: %q", results[0].output)
	}
}

func TestBatchLifecycleDeniedCallEmitsOnePairedFailure(t *testing.T) {
	storage.SetTestDirs(t, t.TempDir())
	sess := NewSession("test", "test", "system", tool.NewRegistry(orderedReadTool{}))
	sess.PermSvc().SetDryRun(true)
	ch := make(chan StreamEvent, 8)
	results := sess.Tools().ExecuteAll(context.Background(), []types.ToolCall{{ID: "denied", Name: "Read"}}, ch, 0, "")
	if len(results) != 1 || !results[0].isErr {
		t.Fatal("denied call ran")
	}
	count := 0
	for len(ch) > 0 {
		event := <-ch
		if event.Type == "tool_result" {
			count++
			if event.ToolID != "denied" || event.ToolState != ToolStateFailed || event.ToolReason != ToolReasonPermission {
				t.Fatalf("unpaired failure: %#v", event)
			}
		}
	}
	if count != 1 {
		t.Fatalf("denied call emitted %d terminal events", count)
	}
}

func TestBatchLifecycleAgentStreamReportsActualCompletion(t *testing.T) {
	storage.SetTestDirs(t, t.TempDir())
	client := &resilienceBoundaryClient{streams: [][]types.RelayStreamEvent{
		{{Type: "tool_call", ToolCall: &types.ToolCall{ID: "read-7", Name: "Read", Arguments: map[string]interface{}{"id": 7}}}, {Type: "done", StopReason: "tool_use"}},
		{{Type: "content", Content: "Read complete."}, {Type: "done", StopReason: "end_turn"}},
	}}
	sess := NewSessionWithClient(client, "test", "test-model", "test system", tool.NewRegistry(orderedReadTool{}), false)
	sess.PermSvc().SetAutonomy(safety.AutonomyYOLO)
	sess.LifecycleSvc().Limits().SetMaxTurns(3)
	sess.AddUser("Inspect the file.")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	stream, err := sess.Stream(ctx)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for event := range stream {
		if event.Type == "tool_result" {
			count++
			if event.ToolID != "read-7" || event.ToolState != ToolStateCompleted || event.Content != "result-7" {
				t.Fatalf("incorrect streamed result: %#v", event)
			}
		}
	}
	if count != 1 {
		t.Fatalf("agent stream produced %d completion events", count)
	}
}

func TestBatchLifecycleCompletionAdvancesProposalAndParallelSpecStages(t *testing.T) {
	storage.SetTestDirs(t, t.TempDir())
	sess := NewSession("test", "test", "system", tool.NewRegistry())
	sess.PermSvc().SetSpecStage(safety.SpecStageProposal)
	sess.PermSvc().SetSpecSlug("test-spec")
	for _, step := range []struct {
		name  string
		stage safety.SpecStage
	}{
		{"Proposal", safety.SpecStageProposal}, {"Specify", safety.SpecStageSpecify}, {"Design", safety.SpecStageDesign},
	} {
		sess.Tools().CompleteResult(context.Background(), toolExecResult{tc: types.ToolCall{Name: step.name}, state: ToolStateCompleted, reason: ToolReasonCompleted}, nil)
		if got := sess.PermSvc().SpecStage(); got != step.stage {
			t.Fatalf("after %s: stage=%v, want %v", step.name, got, step.stage)
		}
	}
}
