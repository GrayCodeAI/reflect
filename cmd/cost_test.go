package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	analytics "github.com/GrayCodeAI/reflect/internal/observability"
)

// withCostStateDir points REFLECT_STATE_DIR at a fresh temp dir so cost
// command tests never read (or write) the developer's real ledger.
func withCostStateDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("REFLECT_STATE_DIR", dir)
	return dir
}

// withCostFlags snapshots and restores the cost command flag variables.
func withCostFlags(t *testing.T) {
	t.Helper()
	session, since := costSession, costSince
	analyzeJSON, summaryJSON := costAnalyzeJSON, costSummaryJSON
	t.Cleanup(func() {
		costSession, costSince = session, since
		costAnalyzeJSON, costSummaryJSON = analyzeJSON, summaryJSON
	})
}

// costFixture is one cost.jsonl line per entry, with a stable shape.
type costFixture struct {
	SessionID string
	Model     string
	TaskType  string
	CostUSD   float64
	Age       time.Duration
	Kept      bool
}

func writeCostFixture(t *testing.T, dir string, entries []costFixture) {
	t.Helper()
	var b strings.Builder
	for _, e := range entries {
		b.WriteString(`{"session_id":"` + e.SessionID +
			`","model":"` + e.Model +
			`","task_type":"` + e.TaskType +
			`","input_tokens":100,"output_tokens":10,"cost_usd":` +
			jsonNumber(e.CostUSD) +
			`,"duration":1000000000,"kept":` + jsonBool(e.Kept) +
			`,"timestamp":"` + time.Now().Add(-e.Age).UTC().Format(time.RFC3339) + `"}` + "\n")
	}
	path := filepath.Join(dir, "cost.jsonl")
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		t.Fatalf("write cost fixture: %v", err)
	}
}

func jsonNumber(f float64) string {
	raw, _ := json.Marshal(f)
	return string(raw)
}

func jsonBool(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func TestCostAnalyze_JSON_Empty(t *testing.T) {
	withCostStateDir(t)
	withCostFlags(t)
	costAnalyzeJSON = true

	var buf bytes.Buffer
	costAnalyzeCmd.SetOut(&buf)
	costAnalyzeCmd.SetErr(&buf)

	if err := costAnalyzeCmd.RunE(costAnalyzeCmd, nil); err != nil {
		t.Fatalf("RunE returned error: %v", err)
	}

	var report analytics.OptimizationReport
	if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatalf("decoding JSON output %q: %v", buf.String(), err)
	}
	if report.TotalSpend != 0 {
		t.Errorf("expected zero total_spend, got %f", report.TotalSpend)
	}
	if report.ByModel == nil || len(report.ByModel) != 0 {
		t.Errorf("expected empty by_model map, got %v", report.ByModel)
	}
}

func TestCostSummary_JSON_Empty(t *testing.T) {
	withCostStateDir(t)
	withCostFlags(t)
	costSummaryJSON = true

	var buf bytes.Buffer
	costSummaryCmd.SetOut(&buf)
	costSummaryCmd.SetErr(&buf)

	if err := costSummaryCmd.RunE(costSummaryCmd, nil); err != nil {
		t.Fatalf("RunE returned error: %v", err)
	}

	var report analytics.OptimizationReport
	if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatalf("decoding JSON output %q: %v", buf.String(), err)
	}
	if report.YieldRate != 0 {
		t.Errorf("expected zero yield_rate, got %f", report.YieldRate)
	}
}

func TestCostAnalyze_Text_Empty(t *testing.T) {
	withCostStateDir(t)
	withCostFlags(t)
	costAnalyzeJSON = false

	var buf bytes.Buffer
	costAnalyzeCmd.SetOut(&buf)
	costAnalyzeCmd.SetErr(&buf)

	if err := costAnalyzeCmd.RunE(costAnalyzeCmd, nil); err != nil {
		t.Fatalf("RunE returned error: %v", err)
	}

	out := buf.String()
	if strings.HasPrefix(strings.TrimSpace(out), "{") {
		t.Error("text mode should not start with '{'")
	}
	if !strings.Contains(out, "No cost data recorded yet") {
		t.Errorf("expected empty-state notice, got: %q", out)
	}
}

func TestCostAnalyze_ReportsLedgerTotals(t *testing.T) {
	dir := withCostStateDir(t)
	withCostFlags(t)
	costAnalyzeJSON = false
	writeCostFixture(t, dir, []costFixture{
		{SessionID: "ses-1", Model: "gpt-4o", TaskType: "code", CostUSD: 0.01, Age: 2 * time.Hour, Kept: true},
		{SessionID: "ses-2", Model: "claude-3-5", TaskType: "chat", CostUSD: 0.02, Age: 2 * time.Hour, Kept: true},
		{SessionID: "ses-1", Model: "old-model", TaskType: "code", CostUSD: 9.0, Age: 72 * time.Hour, Kept: true},
	})

	var buf bytes.Buffer
	costAnalyzeCmd.SetOut(&buf)
	costAnalyzeCmd.SetErr(&buf)
	if err := costAnalyzeCmd.RunE(costAnalyzeCmd, nil); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"Cost analysis (all)",
		"Total spend:      $9.0300",
		"gpt-4o",
		"claude-3-5",
		"old-model",
		"Spend by Model",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("analyze output missing %q:\n%s", want, out)
		}
	}
}

func TestCostAnalyze_SessionFilter(t *testing.T) {
	dir := withCostStateDir(t)
	withCostFlags(t)
	costSession = "ses-2"
	writeCostFixture(t, dir, []costFixture{
		{SessionID: "ses-1", Model: "gpt-4o", TaskType: "code", CostUSD: 0.01, Age: time.Hour, Kept: true},
		{SessionID: "ses-2", Model: "claude-3-5", TaskType: "chat", CostUSD: 0.02, Age: time.Hour, Kept: true},
	})

	var buf bytes.Buffer
	costAnalyzeCmd.SetOut(&buf)
	costAnalyzeCmd.SetErr(&buf)
	if err := costAnalyzeCmd.RunE(costAnalyzeCmd, nil); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "session ses-2, all") {
		t.Errorf("scope label missing session filter:\n%s", out)
	}
	if !strings.Contains(out, "Total spend:      $0.0200") {
		t.Errorf("session-filtered total wrong:\n%s", out)
	}
	if strings.Contains(out, "gpt-4o") {
		t.Errorf("session filter leaked ses-1 entries:\n%s", out)
	}
}

func TestCostAnalyze_SinceFilter(t *testing.T) {
	dir := withCostStateDir(t)
	withCostFlags(t)
	costSince = "24h"
	writeCostFixture(t, dir, []costFixture{
		{SessionID: "ses-1", Model: "fresh-model", TaskType: "code", CostUSD: 0.01, Age: 2 * time.Hour, Kept: true},
		{SessionID: "ses-1", Model: "old-model", TaskType: "code", CostUSD: 9.0, Age: 72 * time.Hour, Kept: true},
	})

	var buf bytes.Buffer
	costAnalyzeCmd.SetOut(&buf)
	costAnalyzeCmd.SetErr(&buf)
	if err := costAnalyzeCmd.RunE(costAnalyzeCmd, nil); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Cost analysis (24h)") {
		t.Errorf("window label missing:\n%s", out)
	}
	if !strings.Contains(out, "Total spend:      $0.0100") {
		t.Errorf("window-filtered total wrong:\n%s", out)
	}
	if strings.Contains(out, "old-model") {
		t.Errorf("24h window included 3-day-old entry:\n%s", out)
	}
}

func TestCostAnalyze_RejectsInvalidSince(t *testing.T) {
	withCostStateDir(t)
	withCostFlags(t)
	costSince = "1y"

	var buf bytes.Buffer
	costAnalyzeCmd.SetOut(&buf)
	costAnalyzeCmd.SetErr(&buf)
	err := costAnalyzeCmd.RunE(costAnalyzeCmd, nil)
	if err == nil {
		t.Fatal("RunE succeeded with --since 1y, want error")
	}
	if !strings.Contains(err.Error(), "invalid --since") {
		t.Errorf("error = %v, want invalid --since", err)
	}
}

func TestCostSummary_ReportsLedgerTotals(t *testing.T) {
	dir := withCostStateDir(t)
	withCostFlags(t)
	costSummaryJSON = false
	writeCostFixture(t, dir, []costFixture{
		{SessionID: "ses-1", Model: "gpt-4o", TaskType: "code", CostUSD: 0.0123, Age: 2 * time.Hour, Kept: true},
		{SessionID: "ses-1", Model: "claude-3-5", TaskType: "chat", CostUSD: 0.0456, Age: 2 * time.Hour, Kept: true},
	})

	var buf bytes.Buffer
	costSummaryCmd.SetOut(&buf)
	costSummaryCmd.SetErr(&buf)
	if err := costSummaryCmd.RunE(costSummaryCmd, nil); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"Cost summary (all)",
		"Total spend:      $0.0579",
		"Yield rate:       100.0%",
		"Spend by model:",
		// Most expensive first; the model column is 28 chars wide.
		"claude-3-5",
		"1 calls  $0.0456",
		"gpt-4o",
		"1 calls  $0.0123",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("summary output missing %q:\n%s", want, out)
		}
	}
	if strings.Index(out, "claude-3-5") > strings.Index(out, "gpt-4o") {
		t.Errorf("models not ordered by cost descending:\n%s", out)
	}
}

func TestCostAnalyze_JSON_Fixture(t *testing.T) {
	dir := withCostStateDir(t)
	withCostFlags(t)
	costAnalyzeJSON = true
	writeCostFixture(t, dir, []costFixture{
		{SessionID: "ses-1", Model: "gpt-4o", TaskType: "code", CostUSD: 0.01, Age: time.Hour, Kept: true},
		{SessionID: "ses-2", Model: "claude-3-5", TaskType: "chat", CostUSD: 0.02, Age: time.Hour, Kept: true},
	})

	var buf bytes.Buffer
	costAnalyzeCmd.SetOut(&buf)
	costAnalyzeCmd.SetErr(&buf)
	if err := costAnalyzeCmd.RunE(costAnalyzeCmd, nil); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	var report analytics.OptimizationReport
	if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatalf("decoding JSON output %q: %v", buf.String(), err)
	}
	if report.TotalSpend != 0.03 {
		t.Errorf("TotalSpend = %v, want 0.03", report.TotalSpend)
	}
	if report.ByModel["gpt-4o"] == nil || report.ByModel["claude-3-5"] == nil {
		t.Errorf("ByModel missing fixture models: %v", report.ByModel)
	}
}
