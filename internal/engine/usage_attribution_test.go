package engine

import (
	"testing"
	"time"

	"github.com/GrayCodeAI/reflect/internal/tool"
	"github.com/GrayCodeAI/reflect/internal/types"
	"github.com/GrayCodeAI/reflect/internal/usage"
)

func TestSetUsageAttributionNormalizesWhitespace(t *testing.T) {
	sess := NewSessionWithClient(NewMockClientForTest(), "acme", "acme/m", "", nil, true)
	sess.SetUsageAttribution(UsageAttribution{SubagentID: "  explore\t", MissionID: " m-42 "})
	attrib := sess.usageAttribution()
	if attrib.SubagentID != "explore" || attrib.MissionID != "m-42" {
		t.Fatalf("usageAttribution = %+v, want trimmed identity", attrib)
	}
}

func TestSubSessionInheritsMissionAttribution(t *testing.T) {
	t.Setenv("REFLECT_STATE_DIR", t.TempDir())
	parent := NewSessionWithClient(NewMockClientForTest(), "acme", "acme/m", "", tool.NewRegistry(), true)
	parent.SetUsageAttribution(UsageAttribution{SubagentID: "worker:feat-1", MissionID: "m-42"})

	sub := parent.SubSession("acme/m", "", nil)
	attrib := sub.usageAttribution()
	if attrib.MissionID != "m-42" {
		t.Errorf("child MissionID = %q, want inherited m-42", attrib.MissionID)
	}
	if attrib.SubagentID != "" {
		t.Errorf("child SubagentID = %q, want cleared (set by the spawn site)", attrib.SubagentID)
	}
}

func TestRecordStreamUsageStampsAttribution(t *testing.T) {
	t.Setenv("REFLECT_STATE_DIR", t.TempDir())
	sess := NewSessionWithClient(NewMockClientForTest(), "acme", "acme/attr-model", "", nil, true)
	sess.SetUsageAttribution(UsageAttribution{SubagentID: "explore", MissionID: "m-42"})
	ch := make(chan StreamEvent, 1)

	sess.recordStreamUsage(ch, &types.RelayUsage{PromptTokens: 10, CompletionTokens: 5}, "acme", "acme/attr-model", "tool", time.Now())
	<-ch

	records, err := usage.Read()
	if err != nil {
		t.Fatalf("usage.Read: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("ledger has %d records, want 1", len(records))
	}
	r := records[0]
	if r.Model != "acme/attr-model" {
		t.Errorf("record model = %q, want acme/attr-model", r.Model)
	}
	if r.TaskType != "tool" {
		t.Errorf("record TaskType = %q, want tool", r.TaskType)
	}
	if r.SubagentID != "explore" || r.MissionID != "m-42" {
		t.Errorf("record attribution = %q/%q, want explore/m-42", r.SubagentID, r.MissionID)
	}
}

func TestRecordStreamUsageMainSessionOmitsAttribution(t *testing.T) {
	t.Setenv("REFLECT_STATE_DIR", t.TempDir())
	sess := NewSessionWithClient(NewMockClientForTest(), "acme", "acme/main-model", "", nil, true)
	ch := make(chan StreamEvent, 1)

	sess.recordStreamUsage(ch, &types.RelayUsage{PromptTokens: 10, CompletionTokens: 5}, "acme", "acme/main-model", "code", time.Now())
	<-ch

	records, err := usage.Read()
	if err != nil {
		t.Fatalf("usage.Read: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("ledger has %d records, want 1", len(records))
	}
	r := records[0]
	if r.SubagentID != "" || r.MissionID != "" {
		t.Errorf("main-session record gained attribution: %+v", r)
	}
	if r.TaskType != "code" {
		t.Errorf("record TaskType = %q, want code", r.TaskType)
	}
}
