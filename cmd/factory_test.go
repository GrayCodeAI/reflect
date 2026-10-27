package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFactoryStatusWithRealProductCLIs(t *testing.T) {
	binDir := os.Getenv("GRAYCODE_E2E_BIN_DIR")
	if binDir == "" {
		t.Skip("set GRAYCODE_E2E_BIN_DIR to test the local factory CLIs")
	}
	repo := t.TempDir()
	for _, args := range [][]string{{"init"}, {"config", "user.email", "test@example.com"}, {"config", "user.name", "Test"}} {
		if _, err := workGit(repo, args...); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("factory\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "README.md"}, {"commit", "-m", "base"}} {
		if _, err := workGit(repo, args...); err != nil {
			t.Fatal(err)
		}
	}
	commit, err := workGit(repo, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	resolve := filepath.Join(binDir, "resolve")
	record := filepath.Join(binDir, "record")
	recall := filepath.Join(binDir, "recall")
	rally := filepath.Join(binDir, "rally")
	for _, path := range []string{resolve, record, recall, rally} {
		if _, err := os.Stat(path); err != nil {
			t.Skipf("build %s to run integration: %v", path, err)
		}
	}
	var ticket workTicket
	if err := workJSON(context.Background(), resolve, []string{"new", "--repo", repo, "--title", "Factory ticket"}, &ticket); err != nil {
		t.Fatal(err)
	}
	if err := workCall(context.Background(), resolve, "status", "--repo", repo, "--set", "in_review", ticket.ID); err != nil {
		t.Fatal(err)
	}
	var checkpoint factoryCheckpoint
	if err := workJSON(context.Background(), record, []string{"create", "--repo", repo, "--task", ticket.ID, "--summary", "Verified change", "--verification", "go test ./...: passed"}, &checkpoint); err != nil {
		t.Fatal(err)
	}
	recallStore, rallyData := t.TempDir(), t.TempDir()
	if err := workCall(context.Background(), recall, "remember", "--store", recallStore, "--scope", filepath.Base(repo), "--kind", "outcome", "--source", "commit:"+commit, "--text", "Verified factory ticket"); err != nil {
		t.Fatal(err)
	}
	if err := workCall(context.Background(), rally, "note", "--data", rallyData, "--channel", filepath.Base(repo), "--ticket", ticket.ID, "--text", "Ready for review"); err != nil {
		t.Fatal(err)
	}
	oldRepo, oldResolve, oldRecord := factoryRepo, factoryResolveBin, factoryRecordBin
	oldRecall, oldRally := factoryRecallBin, factoryRallyBin
	oldRecallStore, oldRallyData := factoryRecallStore, factoryRallyData
	t.Cleanup(func() {
		factoryRepo, factoryResolveBin, factoryRecordBin = oldRepo, oldResolve, oldRecord
		factoryRecallBin, factoryRallyBin = oldRecall, oldRally
		factoryRecallStore, factoryRallyData = oldRecallStore, oldRallyData
	})
	factoryRepo, factoryResolveBin, factoryRecordBin = repo, resolve, record
	factoryRecallBin, factoryRallyBin = recall, rally
	factoryRecallStore, factoryRallyData = recallStore, rallyData
	status, err := collectFactoryStatus(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(status.Tickets) != 1 || !status.Tickets[0].ReadyForReview {
		t.Fatalf("factory tickets = %+v", status.Tickets)
	}
	if status.Tickets[0].Checkpoints != 1 || status.Tickets[0].Memories != 1 || status.Tickets[0].Messages != 1 {
		t.Fatalf("factory links = %+v", status.Tickets[0])
	}
	if status.Tickets[0].LastMessage != "Ready for review" {
		t.Fatalf("latest Rally message = %q", status.Tickets[0].LastMessage)
	}
	if !strings.Contains(factorySummary(status), "1 verified checkpoints") {
		t.Fatal(factorySummary(status))
	}
}

func TestFactoryVerification(t *testing.T) {
	repo := t.TempDir()
	if got := factoryVerification(repo); got != "" {
		t.Fatalf("unknown project gate = %q", got)
	}
	if err := os.WriteFile(filepath.Join(repo, "go.mod"), []byte("module example.com/test\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := factoryVerification(repo); got != "go test ./..." {
		t.Fatalf("Go project gate = %q", got)
	}
}
