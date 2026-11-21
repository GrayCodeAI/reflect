package session

import (
	"testing"

	store "github.com/GrayCodeAI/reflect/internal/session"
	"github.com/GrayCodeAI/reflect/internal/types"
)

func TestHydrateKeepsRuntimeAndVisibleDisplayProjections(t *testing.T) {
	saved := &store.Session{Messages: []store.Message{
		{Role: "system", Content: "system"},
		{Role: "user", Content: "hello"},
		{Role: "tool", Content: "tool output"},
		{Role: "assistant", Content: "hi"},
	}}
	got := Hydrate(saved)
	if len(got.Runtime) != 4 || len(got.Display) != 2 || got.Display[0].Role != "user" || got.Display[1].Content != "hi" {
		t.Fatalf("Hydrate = %#v", got)
	}
}

func TestHydrateRetainsConcurrentToolIdentityAndObservedFailure(t *testing.T) {
	saved := &store.Session{Messages: []store.Message{
		{Role: "user", Content: "check files"},
		{Role: "assistant", ToolUse: []types.ToolCall{{ID: "slow", Name: "Read"}, {ID: "fast", Name: "Read"}}},
		{Role: "user", ToolResults: []types.ToolResult{{ToolUseID: "fast", Content: "fast output"}}},
		{Role: "user", ToolResults: []types.ToolResult{{ToolUseID: "slow", Content: "missing file", IsError: true}}},
		{Role: "assistant", Content: "One file is missing."},
	}}
	got := Hydrate(saved)
	if len(got.Runtime) != 5 || len(got.Display) != 6 {
		t.Fatalf("unexpected projections: %#v", got)
	}
	if got.Display[1].Call.ID != "slow" || !got.Display[1].Result.IsError || got.Display[2].Result.IsError {
		t.Fatal("out-of-order results lost their call identity or failure state")
	}
	if got.Display[3].Call.ID != "fast" || got.Display[3].Content != "fast output" || got.Runtime[3].ToolResults[0].Content != "missing file" {
		t.Fatal("presentation changed the durable result or the model transcript")
	}
}

func TestHydrateScopesReusedToolIDsToTheirTurn(t *testing.T) {
	saved := &store.Session{Messages: []store.Message{
		{Role: "assistant", ToolUse: []types.ToolCall{{ID: "call", Name: "Read"}}},
		{Role: "user", ToolResults: []types.ToolResult{{ToolUseID: "call", Content: "first", IsError: true}}},
		{Role: "user", Content: "again"},
		{Role: "assistant", ToolUse: []types.ToolCall{{ID: "call", Name: "Read"}}},
	}}
	got := Hydrate(saved)
	if got.Display[0].Result == nil || !got.Display[0].Result.IsError || got.Display[3].Result != nil {
		t.Fatal("a reused ID turned an incomplete call into a previous turn's result")
	}
}

func TestHydrateNil(t *testing.T) {
	got := Hydrate(nil)
	if got.Runtime != nil || got.Display != nil {
		t.Fatalf("Hydrate(nil) = %#v", got)
	}
}
