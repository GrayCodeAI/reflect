package cmd

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// writeUsageFixture writes raw lines into the temp state dir's usage ledger.
func writeUsageFixture(t *testing.T, lines ...string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("REFLECT_STATE_DIR", dir)
	path := filepath.Join(dir, "usage", "usage.jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("mkdir usage dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatalf("write usage fixture: %v", err)
	}
}

func TestHelperSpendSummaryAttributedLedger(t *testing.T) {
	now := time.Now().UnixMilli()
	writeUsageFixture(
		t,
		`{"schema_version":1,"kind":"generation","fact":{"created_at_ms":`+strconv.FormatInt(now, 10)+`,"model":"a/m","input_tokens":50,"output_tokens":5,"total_cost":0.02,"subagent_id":"explore"}}`,
		`{"schema_version":1,"kind":"generation","fact":{"created_at_ms":`+strconv.FormatInt(now, 10)+`,"model":"a/m","input_tokens":5,"output_tokens":1,"total_cost":0.03,"subagent_id":"worker:feat-1","mission_id":"m-42"}}`,
	)

	got := helperSpendSummary()
	if got == "" {
		t.Fatal("helperSpendSummary empty, want breakdown for attributed ledger")
	}
	for _, want := range []string{
		"Helpers (ledger, last 24h)",
		"mission m-42/worker:feat-1 1 gens $0.0300",
		"subagent explore 1 gens $0.0200",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("helper summary missing %q: %q", want, got)
		}
	}
}

func TestHelperSpendSummaryUnattributedLedger(t *testing.T) {
	now := time.Now().UnixMilli()
	writeUsageFixture(
		t,
		`{"schema_version":1,"kind":"generation","fact":{"created_at_ms":`+strconv.FormatInt(now, 10)+`,"model":"a/m","input_tokens":50,"output_tokens":5,"total_cost":0.02}}`,
	)
	if got := helperSpendSummary(); got != "" {
		t.Errorf("helperSpendSummary = %q, want empty without attribution", got)
	}
}

func TestHelperSpendSummaryMissingLedger(t *testing.T) {
	t.Setenv("REFLECT_STATE_DIR", t.TempDir())
	if got := helperSpendSummary(); got != "" {
		t.Errorf("helperSpendSummary = %q, want empty for missing ledger", got)
	}
}
