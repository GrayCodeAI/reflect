package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestWorkConnectsResolveReflectAndRecord(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test fixture uses POSIX shell")
	}
	repo := t.TempDir()
	for _, args := range [][]string{{"init"}, {"config", "user.email", "test@example.com"}, {"config", "user.name", "Test"}} {
		if _, err := workGit(repo, args...); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("base\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "README.md"}, {"commit", "-m", "base"}} {
		if _, err := workGit(repo, args...); err != nil {
			t.Fatal(err)
		}
	}
	base, _ := workGit(repo, "rev-parse", "HEAD")
	t.Cleanup(func() {
		listing, _ := workGit(repo, "worktree", "list", "--porcelain")
		for _, line := range strings.Split(listing, "\n") {
			if strings.HasPrefix(line, "worktree ") {
				path := strings.TrimPrefix(line, "worktree ")
				if path != repo {
					_, _ = workGit(repo, "worktree", "remove", "--force", path)
				}
			}
		}
	})
	binDir := t.TempDir()
	resolveLog := filepath.Join(binDir, "resolve.log")
	resolve := filepath.Join(binDir, "resolve")
	resolveScript := fmt.Sprintf("#!/bin/sh\necho \"$1\" >> %q\nif [ \"$1\" = show ]; then printf '{\"id\":\"abc123\",\"title\":\"Fix onboarding\",\"body\":\"Make setup clear\",\"status\":\"open\"}\\n'; else printf '{}\\n'; fi\n", resolveLog)
	if err := os.WriteFile(resolve, []byte(resolveScript), 0700); err != nil {
		t.Fatal(err)
	}
	record := filepath.Join(binDir, "record")
	recordScript := "#!/bin/sh\nshift\nwhile [ $# -gt 0 ]; do if [ \"$1\" = --repo ]; then shift; repo=$1; break; fi; shift; done\ncommit=$(git -C \"$repo\" rev-parse HEAD)\nprintf '{\"id\":\"checkpoint123\",\"commit\":\"%s\"}\\n' \"$commit\"\n"
	if err := os.WriteFile(record, []byte(recordScript), 0700); err != nil {
		t.Fatal(err)
	}
	recall := filepath.Join(binDir, "recall")
	if err := os.WriteFile(recall, []byte("#!/bin/sh\nif [ \"$1\" = list ]; then echo '[]'; else echo '{\"id\":\"memory123\"}'; fi\n"), 0700); err != nil {
		t.Fatal(err)
	}
	rally := filepath.Join(binDir, "rally")
	if err := os.WriteFile(rally, []byte("#!/bin/sh\necho '{}'\n"), 0700); err != nil {
		t.Fatal(err)
	}
	oldRepo, oldVerify, oldAuto, oldModel := workRepo, workVerify, workAuto, workModel
	oldResolve, oldRecord, oldAgent := workResolveBin, workRecordBin, workAgentJSON
	oldRecall, oldRally, oldRequest := workRecallBin, workRallyBin, workRequest
	oldRecallStore, oldRallyData := workRecallStore, workRallyData
	oldOpenPR, oldPRBase := workOpenPR, workPRBase
	oldRivetBin, oldVerifyBlock := workRivetBin, workVerifyBlock
	t.Cleanup(func() {
		workRepo, workVerify, workAuto, workModel = oldRepo, oldVerify, oldAuto, oldModel
		workResolveBin, workRecordBin, workAgentJSON = oldResolve, oldRecord, oldAgent
		workRecallBin, workRallyBin, workRequest = oldRecall, oldRally, oldRequest
		workRecallStore, workRallyData = oldRecallStore, oldRallyData
		workOpenPR, workPRBase = oldOpenPR, oldPRBase
		workRivetBin, workVerifyBlock = oldRivetBin, oldVerifyBlock
	})
	workRepo, workVerify, workAuto, workModel = repo, "test -f change.txt", "semi", ""
	workResolveBin, workRecordBin = resolve, record
	workRecallBin, workRallyBin, workRequest = recall, rally, ""
	workRecallStore, workRallyData = "", ""
	workOpenPR, workPRBase = false, ""
	workRivetBin, workVerifyBlock = "", ""
	workAgentJSON = func(_ context.Context, _ string, args []string, dst any) error {
		var worktree string
		for i := range args {
			if args[i] == "--cwd" && i+1 < len(args) {
				worktree = args[i+1]
			}
		}
		if worktree == "" {
			return fmt.Errorf("missing worktree")
		}
		if err := os.WriteFile(filepath.Join(worktree, "change.txt"), []byte("fixed\n"), 0600); err != nil {
			return err
		}
		*dst.(*ExecResult) = ExecResult{SessionID: "exec-test", ExitCode: 0}
		return nil
	}
	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := runWork(cmd, []string{"abc123"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Checkpoint: checkpoint123") {
		t.Fatalf("missing checkpoint in output: %s", out.String())
	}
	logs, err := os.ReadFile(resolveLog)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(logs)); got != "show\nclaim\ncomment\nstatus" {
		t.Fatalf("Resolve calls: %q", got)
	}
	listing, err := workGit(repo, "worktree", "list", "--porcelain")
	if err != nil {
		t.Fatal(err)
	}
	var worktree string
	for _, line := range strings.Split(listing, "\n") {
		if strings.HasPrefix(line, "worktree ") {
			path := strings.TrimPrefix(line, "worktree ")
			if path != repo {
				worktree = path
			}
		}
	}
	if worktree == "" {
		t.Fatal("worktree was removed")
	}
	commit, err := workGit(worktree, "rev-parse", "HEAD")
	if err != nil || commit == base {
		t.Fatalf("ticket change was not committed: %q, %v", commit, err)
	}
	if clean, err := workGit(worktree, "status", "--porcelain"); err != nil || clean != "" {
		t.Fatalf("worktree is not clean: %q, %v", clean, err)
	}
	remote := filepath.Join(t.TempDir(), "remote.git")
	if _, err := workGit(repo, "init", "--bare", remote); err != nil {
		t.Fatal(err)
	}
	if _, err := workGit(repo, "remote", "add", "origin", remote); err != nil {
		t.Fatal(err)
	}
	gh := filepath.Join(binDir, "gh")
	ghLog := filepath.Join(binDir, "gh.log")
	if err := os.WriteFile(gh, []byte(fmt.Sprintf("#!/bin/sh\necho \"$*\" > %q\necho https://github.com/example/project/pull/1\n", ghLog)), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	branch, err := workGit(worktree, "branch", "--show-current")
	if err != nil {
		t.Fatal(err)
	}
	url, err := workPublishPR(context.Background(), worktree, branch, "main", workTicket{ID: "abc123", Title: "Fix onboarding"}, workCheckpoint{ID: "checkpoint123", Commit: commit})
	if err != nil || url != "https://github.com/example/project/pull/1" {
		t.Fatalf("publish draft PR: %q, %v", url, err)
	}
	ghArgs, err := os.ReadFile(ghLog)
	if err != nil || !strings.Contains(string(ghArgs), "pr create --draft --base main") {
		t.Fatalf("gh args: %q, %v", ghArgs, err)
	}
	if pushed, err := workGit(repo, "ls-remote", "origin", "refs/heads/"+branch); err != nil || !strings.HasPrefix(pushed, commit) {
		t.Fatalf("verified commit not pushed: %q, %v", pushed, err)
	}
	workVerify = "false"
	if err := runWork(cmd, []string{"abc123"}); err == nil || !strings.Contains(err.Error(), "verification failed") {
		t.Fatalf("expected independent verification to block commit: %v", err)
	}
	logs, err = os.ReadFile(resolveLog)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(logs), "comment\n") != 1 || strings.Count(string(logs), "status\n") != 1 {
		t.Fatalf("failed verification advanced Resolve: %q", logs)
	}
}

// Run with GRAYCODE_E2E_BIN_DIR pointing to real Resolve, Record, Recall and
// Rally binaries. The model call is stubbed; all local product CLIs are real.
func TestWorkPlainRequestWithRealProductCLIs(t *testing.T) {
	testWorkRealProductCLIs(t, false)
}

func TestWorkRivetIntegration(t *testing.T) {
	testWorkRealProductCLIs(t, true)
}

func testWorkRealProductCLIs(t *testing.T, withRivet bool) {
	binDir := os.Getenv("GRAYCODE_E2E_BIN_DIR")
	if binDir == "" {
		t.Skip("set GRAYCODE_E2E_BIN_DIR to run the local product integration")
	}
	rivet := filepath.Join(binDir, "rivet")
	if withRivet {
		if _, err := os.Stat(rivet); err != nil {
			t.Skip("build the Rivet CLI in GRAYCODE_E2E_BIN_DIR to run this integration")
		}
	}
	repo := t.TempDir()
	for _, args := range [][]string{{"init"}, {"config", "user.email", "test@example.com"}, {"config", "user.name", "Test"}} {
		if _, err := workGit(repo, args...); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("base\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if withRivet {
		source := t.TempDir()
		script, check := filepath.Join(source, "run.sh"), filepath.Join(source, "verify.sh")
		if err := os.WriteFile(script, []byte("#!/bin/sh\nset -eu\ntest -f \"$RIVET_PROJECT_ROOT/change.txt\"\n"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(check, []byte("#!/bin/sh\nset -eu\nsh \"$RIVET_SCRIPT\"\n"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := workCall(context.Background(), rivet, "add", "--project", repo, "--description", "Check requested change", "--script", script, "--check", check, "change-check"); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{{"add", "-A"}, {"commit", "-m", "base"}} {
		if _, err := workGit(repo, args...); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		listing, _ := workGit(repo, "worktree", "list", "--porcelain")
		for _, line := range strings.Split(listing, "\n") {
			if strings.HasPrefix(line, "worktree ") {
				path := strings.TrimPrefix(line, "worktree ")
				if path != repo {
					_, _ = workGit(repo, "worktree", "remove", "--force", path)
				}
			}
		}
	})
	oldRepo, oldVerify, oldAuto, oldModel := workRepo, workVerify, workAuto, workModel
	oldResolve, oldRecord, oldRecall, oldRally := workResolveBin, workRecordBin, workRecallBin, workRallyBin
	oldRecallStore, oldRallyData, oldRequest := workRecallStore, workRallyData, workRequest
	oldOpenPR, oldPRBase, oldAgent := workOpenPR, workPRBase, workAgentJSON
	oldRivetBin, oldVerifyBlock := workRivetBin, workVerifyBlock
	t.Cleanup(func() {
		workRepo, workVerify, workAuto, workModel = oldRepo, oldVerify, oldAuto, oldModel
		workResolveBin, workRecordBin, workRecallBin, workRallyBin = oldResolve, oldRecord, oldRecall, oldRally
		workRecallStore, workRallyData, workRequest = oldRecallStore, oldRallyData, oldRequest
		workOpenPR, workPRBase, workAgentJSON = oldOpenPR, oldPRBase, oldAgent
		workRivetBin, workVerifyBlock = oldRivetBin, oldVerifyBlock
	})
	workRepo, workVerify, workAuto, workModel = repo, "test -f change.txt", "semi", ""
	workRivetBin, workVerifyBlock = "", ""
	if withRivet {
		workVerify = ""
		workRivetBin, workVerifyBlock = rivet, "change-check"
	}
	workResolveBin, workRecordBin = filepath.Join(binDir, "resolve"), filepath.Join(binDir, "record")
	workRecallBin, workRallyBin = filepath.Join(binDir, "recall"), filepath.Join(binDir, "rally")
	workRecallStore, workRallyData = t.TempDir(), t.TempDir()
	workRequest, workOpenPR, workPRBase = "Build a small verified change\nCreate change.txt", false, ""
	workAgentJSON = func(_ context.Context, _ string, args []string, dst any) error {
		for i := range args {
			if args[i] == "--cwd" && i+1 < len(args) {
				if err := os.WriteFile(filepath.Join(args[i+1], "change.txt"), []byte("built\n"), 0600); err != nil {
					return err
				}
				*dst.(*ExecResult) = ExecResult{SessionID: "exec-e2e", ExitCode: 0}
				return nil
			}
		}
		return fmt.Errorf("missing worktree argument")
	}
	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := runWork(cmd, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Ready for review") {
		t.Fatal(out.String())
	}
	if withRivet {
		if !strings.Contains(out.String(), "Verification log: ") {
			t.Fatalf("Rivet raw log missing from output: %s", out.String())
		}
		logLine := strings.SplitN(strings.SplitN(out.String(), "Verification log: ", 2)[1], "\n", 2)[0]
		if _, err := os.Stat(logLine); err != nil {
			t.Fatalf("Rivet log unavailable: %s: %v", logLine, err)
		}
		checkpointID := strings.SplitN(strings.SplitN(out.String(), "Checkpoint: ", 2)[1], "\n", 2)[0]
		var checkpoint struct {
			Verification []string `json:"verification"`
		}
		if err := workJSON(context.Background(), workRecordBin, []string{"show", "--repo", repo, checkpointID}, &checkpoint); err != nil {
			t.Fatal(err)
		}
		if len(checkpoint.Verification) != 1 || !strings.Contains(checkpoint.Verification[0], logLine) {
			t.Fatalf("Record checkpoint missing Rivet evidence: %+v", checkpoint.Verification)
		}
		plainLog, err := workRivetVerify(context.Background(), rivet, repo, t.TempDir(), "test -f README.md", "", &bytes.Buffer{}, &bytes.Buffer{})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(plainLog); err != nil {
			t.Fatal(err)
		}
		if clean, err := workGit(repo, "status", "--porcelain"); err != nil || clean != "" {
			t.Fatalf("Rivet exec changed source repository: %q, %v", clean, err)
		}
		if _, err := workRivetVerify(context.Background(), rivet, repo, t.TempDir(), "exit 7", "", &bytes.Buffer{}, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "exited 7") {
			t.Fatalf("Rivet failed command was accepted: %v", err)
		}
	}
	var tickets []workTicket
	if err := workJSON(context.Background(), workResolveBin, []string{"list", "--repo", repo}, &tickets); err != nil {
		t.Fatal(err)
	}
	if len(tickets) != 1 || tickets[0].Status != "in_review" {
		t.Fatalf("Resolve items: %+v", tickets)
	}
	var memories []workMemory
	if err := workJSON(context.Background(), workRecallBin, []string{"list", "--store", workRecallStore, "--scope", filepath.Base(repo)}, &memories); err != nil {
		t.Fatal(err)
	}
	if len(memories) != 1 || !strings.HasPrefix(memories[0].Sources[0], "checkpoint:") {
		t.Fatalf("Recall records: %+v", memories)
	}
	if !strings.Contains(out.String(), "Checkpoint: ") || !strings.Contains(out.String(), "Memory: ") {
		t.Fatal(out.String())
	}
	files, err := filepath.Glob(filepath.Join(workRallyData, "messages", "*.json"))
	if err != nil || len(files) != 2 {
		t.Fatalf("Rally messages: %v, %v", files, err)
	}
}
