package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

type workTicket struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	Status string `json:"status"`
}

type workCheckpoint struct {
	ID     string `json:"id"`
	Commit string `json:"commit"`
}

var (
	workRepo       string
	workVerify     string
	workAuto       string
	workModel      string
	workResolveBin string
	workRecordBin  string
	workAgentJSON  = workJSON
)

var workCmd = &cobra.Command{
	Use:   "work <ticket-id>",
	Short: "Implement a Resolve ticket in an isolated worktree",
	Long: `Load a Resolve ticket, run Reflect in a new Git worktree, independently
	commit the changes, verify the exact commit, and create a Record checkpoint.

The worktree and branch remain available for review. The verification command
is supplied by the operator and runs after the agent exits. No PR is pushed or
opened automatically. Resolve and Record must be installed as CLIs.`,
	Args: cobra.ExactArgs(1),
	RunE: runWork,
}

func init() {
	workCmd.Flags().StringVar(&workRepo, "repo", ".", "Git repository containing the Resolve ticket")
	workCmd.Flags().StringVar(&workVerify, "verify", "", "verification command to run in the worktree (required)")
	workCmd.Flags().StringVar(&workAuto, "auto", "semi", "Reflect exec autonomy level (semi or full)")
	workCmd.Flags().StringVar(&workModel, "model", "", "model override for Reflect exec")
	workCmd.Flags().StringVar(&workResolveBin, "resolve-bin", "", "path to the Resolve CLI")
	workCmd.Flags().StringVar(&workRecordBin, "record-bin", "", "path to the Record CLI")
	rootCmd.AddCommand(workCmd)
}

func runWork(cmd *cobra.Command, args []string) error {
	if strings.TrimSpace(workVerify) == "" {
		return errors.New("work requires --verify with a command that checks the change")
	}
	if workAuto != "semi" && workAuto != "full" {
		return errors.New("work --auto must be semi or full")
	}
	repo, err := filepath.Abs(workRepo)
	if err != nil {
		return err
	}
	repo, err = workGit(repo, "rev-parse", "--show-toplevel")
	if err != nil {
		return fmt.Errorf("find Git repository: %w", err)
	}
	status, err := workGit(repo, "status", "--porcelain")
	if err != nil {
		return err
	}
	if status != "" {
		return errors.New("repository has uncommitted changes; commit or stash them before starting a ticket")
	}
	resolveBin, err := workFindBinary(workResolveBin, "resolve")
	if err != nil {
		return err
	}
	recordBin, err := workFindBinary(workRecordBin, "record")
	if err != nil {
		return err
	}
	var ticket workTicket
	if err := workJSON(cmd.Context(), resolveBin, []string{"show", "--repo", repo, args[0]}, &ticket); err != nil {
		return fmt.Errorf("load Resolve ticket: %w", err)
	}
	if ticket.ID != args[0] {
		return fmt.Errorf("Resolve returned ticket %q for %q", ticket.ID, args[0])
	}
	if ticket.Status == "done" {
		return errors.New("ticket is already done")
	}
	base, err := workGit(repo, "rev-parse", "HEAD")
	if err != nil {
		return err
	}
	suffix := randomHex(3)
	branch := "reflect/" + ticket.ID + "-" + suffix
	worktree := filepath.Join(filepath.Dir(repo), ".reflect-worktrees", filepath.Base(repo), ticket.ID+"-"+suffix)
	if err := os.MkdirAll(filepath.Dir(worktree), 0700); err != nil {
		return err
	}
	if _, err := workGit(repo, "worktree", "add", "-b", branch, worktree, base); err != nil {
		return fmt.Errorf("create ticket worktree: %w", err)
	}
	cmd.Printf("Ticket: %s — %s\nBranch: %s\nWorktree: %s\n", ticket.ID, ticket.Title, branch, worktree)
	if err := workCall(cmd.Context(), resolveBin, "claim", "--repo", repo, ticket.ID); err != nil {
		return fmt.Errorf("claim ticket (worktree retained at %s): %w", worktree, err)
	}
	self, err := os.Executable()
	if err != nil {
		return err
	}
	prompt := workPrompt(ticket, workVerify)
	agentArgs := []string{"exec", "--cwd", worktree, "--auto", workAuto, "--json"}
	if workModel != "" {
		agentArgs = append(agentArgs, "--model", workModel)
	}
	agentArgs = append(agentArgs, prompt)
	var result ExecResult
	if err := workAgentJSON(cmd.Context(), self, agentArgs, &result); err != nil {
		return fmt.Errorf("Reflect run failed (worktree retained at %s): %w", worktree, err)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("Reflect run exited %d (worktree retained at %s)", result.ExitCode, worktree)
	}
	// Stage and commit the agent's changes before verification so the test
	// result refers to a stable, exact commit.
	if _, err := workGit(worktree, "add", "-A"); err != nil {
		return err
	}
	changed, err := workGit(worktree, "diff", "--cached", "--name-only")
	if err != nil {
		return err
	}
	head, err := workGit(worktree, "rev-parse", "HEAD")
	if err != nil {
		return err
	}
	if changed == "" && head == base {
		return fmt.Errorf("Reflect made no code change (worktree retained at %s)", worktree)
	}
	if changed != "" {
		message := fmt.Sprintf("resolve %s: %s", ticket.ID, ticket.Title)
		if _, err := workGit(worktree, "commit", "-m", message); err != nil {
			return fmt.Errorf("commit agent change (worktree retained at %s): %w", worktree, err)
		}
	}
	commit, err := workGit(worktree, "rev-parse", "HEAD")
	if err != nil {
		return err
	}
	cmd.Printf("Verifying commit %s: %s\n", commit, workVerify)
	verify := workVerificationCommand(cmd.Context(), workVerify)
	verify.Dir, verify.Stdout, verify.Stderr = worktree, cmd.OutOrStdout(), cmd.ErrOrStderr()
	if err := verify.Run(); err != nil {
		return fmt.Errorf("verification failed for %s (worktree retained at %s): %w", commit, worktree, err)
	}
	clean, err := workGit(worktree, "status", "--porcelain")
	if err != nil {
		return err
	}
	if clean != "" {
		return fmt.Errorf("verification changed the worktree; commit %s cannot be marked verified (worktree retained at %s)", commit, worktree)
	}
	var checkpoint workCheckpoint
	verification := workVerify + ": passed"
	if err := workJSON(cmd.Context(), recordBin, []string{"create", "--repo", worktree, "--session", result.SessionID, "--task", ticket.ID, "--summary", ticket.Title, "--verification", verification}, &checkpoint); err != nil {
		return fmt.Errorf("create Record checkpoint for %s: %w", commit, err)
	}
	if checkpoint.Commit != commit || checkpoint.ID == "" {
		return fmt.Errorf("Record returned an invalid checkpoint for %s", commit)
	}
	comment := fmt.Sprintf("Reflect run %s verified commit %s with `%s`; Record checkpoint %s. Branch: %s", result.SessionID, commit, workVerify, checkpoint.ID, branch)
	if err := workCall(cmd.Context(), resolveBin, "comment", "--repo", repo, "--text", comment, ticket.ID); err != nil {
		return fmt.Errorf("checkpoint %s created, but Resolve comment failed: %w", checkpoint.ID, err)
	}
	if err := workCall(cmd.Context(), resolveBin, "status", "--repo", repo, "--set", "in_review", ticket.ID); err != nil {
		return fmt.Errorf("checkpoint %s created, but Resolve status update failed: %w", checkpoint.ID, err)
	}
	cmd.Printf("Ready for review\nTicket: %s\nSession: %s\nCommit: %s\nCheckpoint: %s\nWorktree: %s\n", ticket.ID, result.SessionID, commit, checkpoint.ID, worktree)
	return nil
}

func workPrompt(t workTicket, verification string) string {
	return fmt.Sprintf("Implement the following Resolve ticket in this worktree. Follow project instructions. Make the smallest complete change. Do not commit or create a pull request; the runner will verify and commit the result.\n\nTicket ID: %s\nTitle: %s\nDescription:\n%s\n\nThe runner will execute this verification command after you finish: %s", t.ID, t.Title, t.Body, verification)
}

func workFindBinary(explicit, name string) (string, error) {
	if explicit != "" {
		return filepath.Abs(explicit)
	}
	if path, err := exec.LookPath(name); err == nil {
		return path, nil
	}
	self, err := os.Executable()
	if err == nil {
		path := filepath.Join(filepath.Dir(self), "..", name, name)
		if stat, err := os.Stat(path); err == nil && !stat.IsDir() {
			return path, nil
		}
	}
	return "", fmt.Errorf("%s CLI not found; build ../%s/cmd/%s and place it on PATH, or pass --%s-bin", name, name, name, name)
}

func workJSON(ctx context.Context, binary string, args []string, dst any) error {
	c := exec.CommandContext(ctx, binary, args...)
	var out, stderr bytes.Buffer
	c.Stdout, c.Stderr = &out, &stderr
	if err := c.Run(); err != nil {
		return fmt.Errorf("%s %s: %w: %s", binary, args[0], err, strings.TrimSpace(stderr.String()))
	}
	if err := json.Unmarshal(out.Bytes(), dst); err != nil {
		return fmt.Errorf("decode %s output: %w", binary, err)
	}
	return nil
}

func workCall(ctx context.Context, binary string, args ...string) error {
	c := exec.CommandContext(ctx, binary, args...)
	var stderr bytes.Buffer
	c.Stdout, c.Stderr = &bytes.Buffer{}, &stderr
	if err := c.Run(); err != nil {
		return fmt.Errorf("%s %s: %w: %s", binary, args[0], err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func workGit(dir string, args ...string) (string, error) {
	c := exec.Command("git", args...)
	c.Dir = dir
	out, err := c.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", args[0], err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

func workVerificationCommand(ctx context.Context, command string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.CommandContext(ctx, "cmd", "/C", command)
	}
	return exec.CommandContext(ctx, "sh", "-c", command)
}
