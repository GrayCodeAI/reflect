package lifecycle

import (
	"strings"
	"sync"
	"testing"
)

func TestLimitTracker_RecordToolCall(t *testing.T) {
	lt := NewLimitTracker(DefaultLimits())
	lt.RecordToolCall("Read")
	lt.RecordToolCall("Bash")
	lt.RecordToolCall("Write")

	exceeded, _ := lt.IsExceeded()
	if exceeded {
		t.Fatal("should not be exceeded after 3 calls")
	}

	summary := lt.Summary()
	if !strings.Contains(summary, "Tools: 3/200") {
		t.Errorf("expected Tools: 3/200 in summary, got %q", summary)
	}
	if !strings.Contains(summary, "Bash: 1/100") {
		t.Errorf("expected Bash: 1/100 in summary, got %q", summary)
	}
	if !strings.Contains(summary, "Files: 1/50") {
		t.Errorf("expected Files: 1/50 in summary, got %q", summary)
	}
}

func TestLimitTracker_IsExceeded_ToolCalls(t *testing.T) {
	limits := SafetyLimits{MaxToolCalls: 3}
	lt := NewLimitTracker(limits)
	lt.RecordToolCall("Read")
	lt.RecordToolCall("Read")

	exceeded, _ := lt.IsExceeded()
	if exceeded {
		t.Fatal("should not be exceeded at 2/3")
	}

	lt.RecordToolCall("Read")
	exceeded, reason := lt.IsExceeded()
	if !exceeded {
		t.Fatal("should be exceeded at 3/3")
	}
	if !strings.Contains(reason, "tool call limit") {
		t.Errorf("unexpected reason: %q", reason)
	}
}

func TestLimitTracker_IsExceeded_Cost(t *testing.T) {
	limits := SafetyLimits{MaxCostUSD: 1.0}
	lt := NewLimitTracker(limits)
	lt.RecordCost(0.5)

	exceeded, _ := lt.IsExceeded()
	if exceeded {
		t.Fatal("should not be exceeded at $0.50/$1.00")
	}

	lt.RecordCost(0.6)
	exceeded, reason := lt.IsExceeded()
	if !exceeded {
		t.Fatal("should be exceeded at $1.10/$1.00")
	}
	if !strings.Contains(reason, "cost limit") {
		t.Errorf("unexpected reason: %q", reason)
	}
}

func TestLimitTracker_IsExceeded_OutputTokens(t *testing.T) {
	limits := SafetyLimits{MaxOutputTokens: 100}
	lt := NewLimitTracker(limits)
	lt.RecordTokens(50)

	exceeded, _ := lt.IsExceeded()
	if exceeded {
		t.Fatal("should not be exceeded at 50/100")
	}

	lt.RecordTokens(60)
	exceeded, reason := lt.IsExceeded()
	if !exceeded {
		t.Fatal("should be exceeded at 110/100")
	}
	if !strings.Contains(reason, "output token limit") {
		t.Errorf("unexpected reason: %q", reason)
	}
}

func TestDefaultLimits(t *testing.T) {
	d := DefaultLimits()
	if d.MaxToolCalls != 200 {
		t.Errorf("expected MaxToolCalls=200, got %d", d.MaxToolCalls)
	}
	if d.MaxFileWrites != 50 {
		t.Errorf("expected MaxFileWrites=50, got %d", d.MaxFileWrites)
	}
}

func TestVibeLimits(t *testing.T) {
	v := VibeLimits()
	if v.MaxToolCalls <= DefaultLimits().MaxToolCalls {
		t.Error("vibe limits should be more permissive than defaults")
	}
}

func TestResearchLimits(t *testing.T) {
	r := ResearchLimits()
	if r.MaxToolCalls >= DefaultLimits().MaxToolCalls {
		t.Error("research limits should be stricter than defaults")
	}
}

func TestLimitTracker_Summary_NoLimits(t *testing.T) {
	lt := NewLimitTracker(SafetyLimits{})
	summary := lt.Summary()
	if summary != "no limits configured" {
		t.Errorf("expected 'no limits configured', got %q", summary)
	}
}

func TestLimitTracker_BashAndFileTracking(t *testing.T) {
	limits := SafetyLimits{MaxBashCommands: 2, MaxFileWrites: 2}
	lt := NewLimitTracker(limits)

	lt.RecordToolCall("Bash")
	lt.RecordToolCall("Edit")
	exceeded, _ := lt.IsExceeded()
	if exceeded {
		t.Fatal("should not be exceeded yet")
	}

	lt.RecordToolCall("Bash")
	exceeded, reason := lt.IsExceeded()
	if !exceeded {
		t.Fatal("bash limit should be exceeded")
	}
	if !strings.Contains(reason, "bash command limit") {
		t.Errorf("unexpected reason: %q", reason)
	}
}

func TestLimitTrackerBeginRequestResetsOperationsAndPreservesLimits(t *testing.T) {
	for _, tc := range []struct {
		name   string
		limits SafetyLimits
		record func(*LimitTracker)
	}{
		{name: "tools", limits: SafetyLimits{MaxToolCalls: 1}, record: func(lt *LimitTracker) { lt.RecordToolCall("Read") }},
		{name: "bash", limits: SafetyLimits{MaxBashCommands: 1}, record: func(lt *LimitTracker) { lt.RecordToolCall("Bash") }},
		{name: "writes", limits: SafetyLimits{MaxFileWrites: 1}, record: func(lt *LimitTracker) { lt.RecordToolCall("Write") }},
		{name: "turns", limits: SafetyLimits{MaxTurns: 1}, record: func(lt *LimitTracker) { lt.RecordTurn() }},
		{name: "output", limits: SafetyLimits{MaxOutputTokens: 1}, record: func(lt *LimitTracker) { lt.RecordTokens(1) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			lt := NewLimitTracker(tc.limits)
			tc.record(lt)
			if exceeded, _ := lt.IsExceeded(); !exceeded {
				t.Fatal("fixture did not exhaust limit")
			}
			lt.BeginRequest()
			if exceeded, reason := lt.IsExceeded(); exceeded {
				t.Fatalf("previous request blocked follow-up: %s", reason)
			}
			tc.record(lt)
			if exceeded, _ := lt.IsExceeded(); !exceeded {
				t.Fatal("configured limit was lost after request reset")
			}
		})
	}
}

func TestLimitTrackerBeginRequestPreservesSessionSpending(t *testing.T) {
	lt := NewLimitTracker(SafetyLimits{MaxBudgetUSD: 1})
	lt.RecordCost(0.6)
	lt.BeginRequest()
	lt.RecordCost(0.5)
	if exceeded, reason := lt.IsExceeded(); !exceeded || !strings.Contains(reason, "cost limit") {
		t.Fatalf("follow-up bypassed session spending limit: exceeded=%v reason=%q", exceeded, reason)
	}
	lt.BeginRequest()
	if exceeded, _ := lt.IsExceeded(); !exceeded {
		t.Fatal("new request cleared an exhausted spending limit")
	}
}

func TestLimitTracker_ConcurrentAccessors(t *testing.T) {
	lt := NewLimitTracker(DefaultLimits())

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 500; j++ {
				lt.SetMaxTurns(j % 100)
				lt.MaxTurns()
				lt.SetMaxBudgetUSD(float64(j % 50))
				lt.MaxBudgetUSD()
				lt.SetCostUSD(float64(j))
				_, _ = lt.IsExceeded()
			}
		}()
	}
	wg.Wait()
}

func TestLimitTracker_CostLimitFallsBackToBudget(t *testing.T) {
	// MaxCostUSD unset -> MaxBudgetUSD is the effective cap.
	lt := NewLimitTracker(SafetyLimits{MaxBudgetUSD: 10})
	lt.SetCostUSD(9.99)
	if exceeded, _ := lt.IsExceeded(); exceeded {
		t.Fatal("should not be exceeded below budget")
	}
	lt.SetCostUSD(10)
	if exceeded, reason := lt.IsExceeded(); !exceeded {
		t.Fatal("budget should be enforced via MaxCostUSD fallback")
	} else if !strings.Contains(reason, "cost limit") {
		t.Errorf("unexpected reason: %q", reason)
	}
}

func TestLimitTracker_SetMaxBudgetUSD_SyncsCostLimit(t *testing.T) {
	lt := NewLimitTracker(SafetyLimits{})
	lt.SetMaxBudgetUSD(25)
	lt.SetCostUSD(24.99)
	if exceeded, _ := lt.IsExceeded(); exceeded {
		t.Fatal("should not be exceeded below budget")
	}
	lt.SetCostUSD(25)
	if exceeded, _ := lt.IsExceeded(); !exceeded {
		t.Fatal("SetMaxBudgetUSD should drive IsExceeded via MaxCostUSD fallback")
	}
}
