package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestWorkFixRoundsFlagDefault(t *testing.T) {
	flag := workCmd.Flags().Lookup("fix-rounds")
	if flag == nil {
		t.Fatal("flag --fix-rounds is not registered on work")
	}
	if flag.DefValue != "0" {
		t.Errorf("--fix-rounds default = %q, want \"0\" (current single-pass behavior)", flag.DefValue)
	}
}

func TestWorkAgentArgs(t *testing.T) {
	tests := []struct {
		name  string
		model string
		want  []string
	}{
		{
			name:  "without model",
			model: "",
			want:  []string{"exec", "--cwd", "/wt", "--auto", "semi", "--json", "do it"},
		},
		{
			name:  "with model",
			model: "m1",
			want:  []string{"exec", "--cwd", "/wt", "--auto", "semi", "--json", "--model", "m1", "do it"},
		},
	}
	for _, tt := range tests {
		got := workAgentArgs("/wt", "semi", tt.model, "do it")
		if fmt.Sprint(got) != fmt.Sprint(tt.want) {
			t.Errorf("%s: workAgentArgs = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestWorkFixPrompt(t *testing.T) {
	prompt := workFixPrompt(workTicket{ID: "t1", Title: "Fix onboarding", Body: "Make setup clear"}, "go test ./...", "assert failed at line 3")
	for _, want := range []string{"t1", "Fix onboarding", "Make setup clear", "go test ./...", "assert failed at line 3"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("fix prompt missing %q:\n%s", want, prompt)
		}
	}
}

func TestWorkRunVerification(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test fixture uses POSIX shell")
	}
	dir := t.TempDir()
	var out bytes.Buffer

	output, err := workRunVerification(context.Background(), dir, "echo hello-verify", &out, &out)
	if err != nil {
		t.Fatalf("passing command: %v", err)
	}
	if !strings.Contains(output, "hello-verify") {
		t.Errorf("captured output = %q, want the command output", output)
	}

	output, err = workRunVerification(context.Background(), dir, "echo bad-verify >&2; exit 3", &out, &out)
	if err == nil {
		t.Fatal("failing command should return an error")
	}
	if !strings.Contains(output, "bad-verify") {
		t.Errorf("captured output = %q, want stderr captured too", output)
	}
}

func TestWorkVerifyFixRounds(t *testing.T) {
	tests := []struct {
		name          string
		fixRounds     int
		verifyFails   int // number of initial failing verification passes
		wantErr       bool
		wantRounds    int
		wantFixCalls  int
		wantOutput    []string
		wantNoOutput  []string
		wantFinalHead string
	}{
		{
			name:         "first pass",
			fixRounds:    2,
			verifyFails:  0,
			wantRounds:   0,
			wantFixCalls: 0,
			wantNoOutput: []string{"Fix round"},
		},
		{
			name:          "fails once then passes",
			fixRounds:     2,
			verifyFails:   1,
			wantRounds:    1,
			wantFixCalls:  1,
			wantOutput:    []string{"starting fix round 1/2", "Fix round 1/2: verification passed"},
			wantFinalHead: "head1",
		},
		{
			name:         "exhausts rounds",
			fixRounds:    2,
			verifyFails:  3,
			wantErr:      true,
			wantRounds:   2,
			wantFixCalls: 2,
			wantOutput:   []string{"starting fix round 1/2", "starting fix round 2/2", "Fix round 2/2: verification still failing"},
		},
		{
			name:         "zero rounds keeps single pass",
			fixRounds:    0,
			verifyFails:  1,
			wantErr:      true,
			wantFixCalls: 0,
			wantNoOutput: []string{"fix round"},
		},
	}
	for _, tt := range tests {
		verifyCalls := 0
		verify := func() (string, error) {
			verifyCalls++
			if verifyCalls <= tt.verifyFails {
				return "boom output", errors.New("exit status 1")
			}
			return "ok", nil
		}
		fixCalls := 0
		var gotOutput string
		fix := func(round int, output string) (string, error) {
			fixCalls++
			gotOutput = output
			return fmt.Sprintf("head%d", fixCalls), nil
		}

		var out bytes.Buffer
		cmd := &cobra.Command{}
		cmd.SetOut(&out)
		cmd.SetErr(&out)
		spec := workFixSpec{
			Worktree:  "/wt",
			Command:   "go test ./...",
			Ticket:    workTicket{ID: "t1", Title: "Fix it", Body: "Body"},
			FixRounds: tt.fixRounds,
			Commit:    "base0",
		}

		commit, rounds, err := workVerifyFixRounds(cmd, spec, verify, fix)
		if tt.wantErr {
			if err == nil {
				t.Fatalf("%s: expected error, got commit=%s rounds=%d", tt.name, commit, rounds)
			}
			if !strings.Contains(err.Error(), "verification failed") || !strings.Contains(err.Error(), "/wt") {
				t.Errorf("%s: error = %v, want verification failure naming the worktree", tt.name, err)
			}
		} else if err != nil {
			t.Fatalf("%s: unexpected error: %v", tt.name, err)
		}
		if rounds != tt.wantRounds {
			t.Errorf("%s: rounds = %d, want %d", tt.name, rounds, tt.wantRounds)
		}
		if fixCalls != tt.wantFixCalls {
			t.Errorf("%s: fix calls = %d, want %d", tt.name, fixCalls, tt.wantFixCalls)
		}
		if !tt.wantErr {
			wantCommit := spec.Commit
			if tt.wantFinalHead != "" {
				wantCommit = tt.wantFinalHead
			}
			if commit != wantCommit {
				t.Errorf("%s: commit = %q, want %q", tt.name, commit, wantCommit)
			}
		}
		if fixCalls > 0 && gotOutput != "boom output" {
			t.Errorf("%s: fix prompt output = %q, want the failed verification output", tt.name, gotOutput)
		}
		for _, want := range tt.wantOutput {
			if !strings.Contains(out.String(), want) {
				t.Errorf("%s: output missing %q:\n%s", tt.name, want, out.String())
			}
		}
		for _, banned := range tt.wantNoOutput {
			if strings.Contains(strings.ToLower(out.String()), strings.ToLower(banned)) {
				t.Errorf("%s: output should not contain %q:\n%s", tt.name, banned, out.String())
			}
		}
	}
}

func TestWorkVerifyFixRounds_FixErrorPropagates(t *testing.T) {
	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	spec := workFixSpec{Worktree: "/wt", FixRounds: 3, Commit: "base0"}
	_, _, err := workVerifyFixRounds(cmd, spec,
		func() (string, error) { return "boom", errors.New("exit status 1") },
		func(int, string) (string, error) { return "", errors.New("fix exploded") })
	if err == nil || !strings.Contains(err.Error(), "fix exploded") {
		t.Fatalf("expected fix error to propagate, got %v", err)
	}
}

func TestWorkFixSession(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test fixture uses POSIX shell and git")
	}
	tests := []struct {
		name        string
		makeChanges bool
		wantOutput  string
	}{
		{name: "commits fix changes", makeChanges: true, wantOutput: "Fix round 1/2: committed"},
		{name: "no changes stays put", makeChanges: false, wantOutput: "Fix round 1/2: no changes from fix session"},
	}
	for _, tt := range tests {
		repo := t.TempDir()
		for _, args := range [][]string{{"init"}, {"config", "user.email", "test@example.com"}, {"config", "user.name", "Test"}} {
			if _, err := workGit(repo, args...); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("base\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		for _, args := range [][]string{{"add", "README.md"}, {"commit", "-m", "base"}} {
			if _, err := workGit(repo, args...); err != nil {
				t.Fatal(err)
			}
		}
		wt := filepath.Join(t.TempDir(), "wt")
		if _, err := workGit(repo, "worktree", "add", "-b", "fixrounds", wt, "HEAD"); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _, _ = workGit(repo, "worktree", "remove", "--force", wt) })
		base, err := workGit(wt, "rev-parse", "HEAD")
		if err != nil {
			t.Fatal(err)
		}

		oldAgent := workAgentJSON
		t.Cleanup(func() { workAgentJSON = oldAgent })
		workAgentJSON = func(_ context.Context, _ string, args []string, dst any) error {
			var worktree string
			for i := range args {
				if args[i] == "--cwd" && i+1 < len(args) {
					worktree = args[i+1]
				}
			}
			if worktree == "" {
				return errors.New("missing worktree")
			}
			if !strings.Contains(strings.Join(args, " "), "fix the code") {
				return errors.New("fix session prompt missing context")
			}
			if tt.makeChanges {
				if err := os.WriteFile(filepath.Join(worktree, "fix.txt"), []byte("fixed\n"), 0o600); err != nil {
					return err
				}
			}
			*dst.(*ExecResult) = ExecResult{SessionID: "fix-session", ExitCode: 0}
			return nil
		}

		var out bytes.Buffer
		cmd := &cobra.Command{}
		cmd.SetOut(&out)
		cmd.SetErr(&out)
		spec := workFixSpec{Worktree: wt, Ticket: workTicket{ID: "t1", Title: "Fix it"}, FixRounds: 2, Commit: base}
		head, err := workFixSession(context.Background(), cmd, spec, 1, "fix the code so verification passes")
		if err != nil {
			t.Fatalf("%s: %v", tt.name, err)
		}
		if !strings.Contains(out.String(), tt.wantOutput) {
			t.Errorf("%s: output missing %q:\n%s", tt.name, tt.wantOutput, out.String())
		}
		if clean, err := workGit(wt, "status", "--porcelain"); err != nil || clean != "" {
			t.Errorf("%s: worktree not clean: %q, %v", tt.name, clean, err)
		}
		if tt.makeChanges {
			if head == base {
				t.Errorf("%s: fix commit not recorded", tt.name)
			}
		} else if head != base {
			t.Errorf("%s: head = %q, want base commit", tt.name, head)
		}
	}
}
