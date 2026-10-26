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
	oldRepo, oldVerify, oldAuto, oldModel := workRepo, workVerify, workAuto, workModel
	oldResolve, oldRecord, oldAgent := workResolveBin, workRecordBin, workAgentJSON
	t.Cleanup(func() {
		workRepo, workVerify, workAuto, workModel = oldRepo, oldVerify, oldAuto, oldModel
		workResolveBin, workRecordBin, workAgentJSON = oldResolve, oldRecord, oldAgent
	})
	workRepo, workVerify, workAuto, workModel = repo, "test -f change.txt", "semi", ""
	workResolveBin, workRecordBin = resolve, record
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
