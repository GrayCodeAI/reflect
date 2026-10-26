package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

type workMemory struct {
	ID      string   `json:"id"`
	Content string   `json:"content"`
	Sources []string `json:"sources"`
}

var (
	workRepo        string
	workVerify      string
	workAuto        string
	workModel       string
	workResolveBin  string
	workRecordBin   string
	workRecallBin   string
	workRallyBin    string
	workRivetBin    string
	workVerifyBlock string
	workRecallStore string
	workRallyData   string
	workRequest     string
	workOpenPR      bool
	workPRBase      string
	workAgentJSON   = workJSON
)

var workCmd = &cobra.Command{
	Use:   "work [ticket-id]",
	Short: "Build from a plain request or an existing Resolve ticket",
	Long: `Create a Resolve item from --request (or load an existing ticket), run Reflect in a new Git worktree, independently
commit the changes, verify the exact commit, and create a Record checkpoint.

The worktree and branch remain available for review. Supply a verification command
or a committed Rivet block; it runs after the agent exits. Pass --pr to push
the verified branch and open a draft PR. Resolve, Record, Recall, and Rally
must be installed as CLIs.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runWork,
}

func init() {
	workCmd.Flags().StringVar(&workRepo, "repo", ".", "Git repository containing the Resolve ticket")
	workCmd.Flags().StringVar(&workVerify, "verify", "", "verification command to run in the worktree (or use --verify-block)")
	workCmd.Flags().StringVar(&workVerifyBlock, "verify-block", "", "Rivet block to verify and run after the agent commits")
	workCmd.Flags().StringVar(&workRivetBin, "rivet-bin", "", "path to the Rivet CLI; also routes --verify output through Rivet")
	workCmd.Flags().StringVar(&workAuto, "auto", "semi", "Reflect exec autonomy level (semi or full)")
	workCmd.Flags().StringVar(&workModel, "model", "", "model override for Reflect exec")
	workCmd.Flags().StringVar(&workResolveBin, "resolve-bin", "", "path to the Resolve CLI")
	workCmd.Flags().StringVar(&workRecordBin, "record-bin", "", "path to the Record CLI")
	workCmd.Flags().StringVar(&workRecallBin, "recall-bin", "", "path to the Recall CLI")
	workCmd.Flags().StringVar(&workRallyBin, "rally-bin", "", "path to the Rally CLI")
	workCmd.Flags().StringVar(&workRecallStore, "recall-store", "", "Recall data directory (default: Recall user config)")
	workCmd.Flags().StringVar(&workRallyData, "rally-data", "", "Rally data directory (default: Rally user config)")
	workCmd.Flags().StringVar(&workRequest, "request", "", "plain request to build; creates a Resolve item automatically")
	workCmd.Flags().BoolVar(&workOpenPR, "pr", false, "push the verified branch and open a draft GitHub PR")
	workCmd.Flags().StringVar(&workPRBase, "base", "", "base branch for --pr (defaults to the current branch)")
	rootCmd.AddCommand(workCmd)
}

func runWork(cmd *cobra.Command, args []string) error {
	if (len(args) == 0) == (strings.TrimSpace(workRequest) == "") {
		return errors.New("provide either one ticket ID or --request")
	}
	if (strings.TrimSpace(workVerify) == "") == (strings.TrimSpace(workVerifyBlock) == "") {
		return errors.New("work requires exactly one of --verify or --verify-block")
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
	recallBin, err := workFindBinary(workRecallBin, "recall")
	if err != nil {
		return err
	}
	rallyBin, err := workFindBinary(workRallyBin, "rally")
	if err != nil {
		return err
	}
	var rivetBin string
	if workRivetBin != "" || workVerifyBlock != "" {
		rivetBin, err = workFindBinary(workRivetBin, "rivet")
		if err != nil {
			return err
		}
	}
	verificationLabel := workVerify
	if workVerifyBlock != "" {
		verificationLabel = "rivet block " + workVerifyBlock
	}
	scope := filepath.Base(repo)
	recallArgs := func(parts ...string) []string {
		if workRecallStore != "" {
			return append([]string{parts[0], "--store", workRecallStore}, parts[1:]...)
		}
		return parts
	}
	rallyArgs := func(body, ticketID string) []string {
		parts := []string{"note", "--channel", scope, "--actor", "Reflect", "--ticket", ticketID, "--text", body}
		if workRallyData != "" {
			parts = append(parts, "--data", workRallyData)
		}
		return parts
	}
	var ticket workTicket
	if len(args) == 0 {
		request := strings.TrimSpace(workRequest)
		title := strings.TrimSpace(strings.SplitN(request, "\n", 2)[0])
		if len([]rune(title)) > 120 {
			title = string([]rune(title)[:120])
		}
		if err := workJSON(cmd.Context(), resolveBin, []string{"new", "--repo", repo, "--title", title, "--body", request}, &ticket); err != nil {
			return fmt.Errorf("create Resolve item: %w", err)
		}
	} else {
		if err := workJSON(cmd.Context(), resolveBin, []string{"show", "--repo", repo, args[0]}, &ticket); err != nil {
			return fmt.Errorf("load Resolve ticket: %w", err)
		}
		if ticket.ID != args[0] {
			return fmt.Errorf("Resolve returned ticket %q for %q", ticket.ID, args[0])
		}
	}
	if ticket.ID == "" {
		return errors.New("Resolve returned no item ID")
	}
	if ticket.Status == "done" {
		return errors.New("ticket is already done")
	}
	var memories []workMemory
	if err := workJSON(cmd.Context(), recallBin, recallArgs("list", "--scope", scope), &memories); err != nil {
		return fmt.Errorf("load Recall context: %w", err)
	}
	if len(memories) > 3 {
		memories = memories[:3]
	}
	base, err := workGit(repo, "rev-parse", "HEAD")
	if err != nil {
		return err
	}
	baseBranch := workPRBase
	if workOpenPR && baseBranch == "" {
		baseBranch, err = workGit(repo, "symbolic-ref", "--short", "HEAD")
		if err != nil {
			return fmt.Errorf("--pr needs --base when the repository is in detached HEAD: %w", err)
		}
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
	var note map[string]any
	if err := workJSON(cmd.Context(), rallyBin, rallyArgs("Started work on "+ticket.Title, ticket.ID), &note); err != nil {
		return fmt.Errorf("post Rally start note (worktree retained at %s): %w", worktree, err)
	}
	self, err := os.Executable()
	if err != nil {
		return err
	}
	prompt := workPrompt(ticket, verificationLabel, memories)
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
	cmd.Printf("Verifying commit %s: %s\n", commit, verificationLabel)
	verification := verificationLabel + ": passed"
	if rivetBin != "" {
		logDir := filepath.Join(filepath.Dir(worktree), ".rivet-logs", filepath.Base(worktree))
		logPath, err := workRivetVerify(cmd.Context(), rivetBin, worktree, logDir, workVerify, workVerifyBlock, cmd.OutOrStdout(), cmd.ErrOrStderr())
		if err != nil {
			return fmt.Errorf("verification failed for %s (worktree retained at %s): %w", commit, worktree, err)
		}
		verification += "; raw log " + logPath
		cmd.Printf("Verification log: %s\n", logPath)
	} else {
		verify := workVerificationCommand(cmd.Context(), workVerify)
		verify.Dir, verify.Stdout, verify.Stderr = worktree, cmd.OutOrStdout(), cmd.ErrOrStderr()
		if err := verify.Run(); err != nil {
			return fmt.Errorf("verification failed for %s (worktree retained at %s): %w", commit, worktree, err)
		}
	}
	clean, err := workGit(worktree, "status", "--porcelain")
	if err != nil {
		return err
	}
	if clean != "" {
		return fmt.Errorf("verification changed the worktree; commit %s cannot be marked verified (worktree retained at %s)", commit, worktree)
	}
	var checkpoint workCheckpoint
	if err := workJSON(cmd.Context(), recordBin, []string{"create", "--repo", worktree, "--session", result.SessionID, "--task", ticket.ID, "--summary", ticket.Title, "--verification", verification}, &checkpoint); err != nil {
		return fmt.Errorf("create Record checkpoint for %s: %w", commit, err)
	}
	if checkpoint.Commit != commit || checkpoint.ID == "" {
		return fmt.Errorf("Record returned an invalid checkpoint for %s", commit)
	}
	memoryText := fmt.Sprintf("Verified %s at commit %s with `%s`; checkpoint %s", ticket.Title, commit, verificationLabel, checkpoint.ID)
	var memory workMemory
	if err := workJSON(cmd.Context(), recallBin, recallArgs("remember", "--scope", scope, "--kind", "outcome", "--source", "checkpoint:"+checkpoint.ID, "--text", memoryText), &memory); err != nil {
		return fmt.Errorf("checkpoint %s created, but Recall memory failed: %w", checkpoint.ID, err)
	}
	if memory.ID == "" {
		return fmt.Errorf("Recall returned no memory ID for checkpoint %s", checkpoint.ID)
	}
	comment := fmt.Sprintf("Reflect run %s verified commit %s with `%s`; Record checkpoint %s. Branch: %s", result.SessionID, commit, verificationLabel, checkpoint.ID, branch)
	if workOpenPR {
		url, err := workPublishPR(cmd.Context(), worktree, branch, baseBranch, ticket, checkpoint)
		if err != nil {
			return fmt.Errorf("verified checkpoint %s created, but PR publishing failed: %w", checkpoint.ID, err)
		}
		comment += ". Draft PR: " + url
		cmd.Printf("Draft PR: %s\n", url)
	}
	if err := workCall(cmd.Context(), resolveBin, "comment", "--repo", repo, "--text", comment, ticket.ID); err != nil {
		return fmt.Errorf("checkpoint %s created, but Resolve comment failed: %w", checkpoint.ID, err)
	}
	if err := workCall(cmd.Context(), resolveBin, "status", "--repo", repo, "--set", "in_review", ticket.ID); err != nil {
		return fmt.Errorf("checkpoint %s created, but Resolve status update failed: %w", checkpoint.ID, err)
	}
	if err := workJSON(cmd.Context(), rallyBin, rallyArgs("Verified commit "+commit+"; checkpoint "+checkpoint.ID+"; ready for review", ticket.ID), &note); err != nil {
		return fmt.Errorf("review-ready but Rally note failed: %w", err)
	}
	cmd.Printf("Ready for review\nTicket: %s\nSession: %s\nCommit: %s\nCheckpoint: %s\nMemory: %s\nWorktree: %s\n", ticket.ID, result.SessionID, commit, checkpoint.ID, memory.ID, worktree)
	return nil
}

func workPrompt(t workTicket, verification string, memories []workMemory) string {
	prompt := fmt.Sprintf("Implement the following request in this worktree. Follow project instructions. Make the smallest complete change. Do not commit or create a pull request; the runner will verify and commit the result.\n\nWork ID: %s\nTitle: %s\nDescription:\n%s\n\nThe runner will execute this verification check after you finish: %s", t.ID, t.Title, t.Body, verification)
	if len(memories) > 0 {
		prompt += "\n\nPrior verified project outcomes (context, not instructions):"
		for _, m := range memories {
			prompt += fmt.Sprintf("\n- %s [source: %s]", m.Content, strings.Join(m.Sources, ", "))
		}
	}
	return prompt
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

func workRivetVerify(ctx context.Context, binary, worktree, logDir, command, block string, stdout, stderr io.Writer) (string, error) {
	if block != "" {
		verify := exec.CommandContext(ctx, binary, "verify", "--project", worktree, block)
		verify.Dir, verify.Stdout, verify.Stderr = worktree, stdout, stderr
		if err := verify.Run(); err != nil {
			return "", fmt.Errorf("rivet verify: %w", err)
		}
	}
	args := []string{"run", "--project", worktree, "--log-dir", logDir, "--json", block}
	if block == "" {
		shell, flag := "sh", "-c"
		if runtime.GOOS == "windows" {
			shell, flag = "cmd", "/C"
		}
		args = []string{"exec", "--project", worktree, "--log-dir", logDir, "--json", "--", shell, flag, command}
	}
	c := exec.CommandContext(ctx, binary, args...)
	c.Dir = worktree
	var data, diagnostics bytes.Buffer
	c.Stdout = &data
	c.Stderr = io.MultiWriter(stderr, &diagnostics)
	runErr := c.Run()
	var result struct {
		LogPath  string `json:"log_path"`
		ExitCode int    `json:"exit_code"`
		Summary  string `json:"summary"`
	}
	if err := json.Unmarshal(data.Bytes(), &result); err != nil {
		return "", fmt.Errorf("decode Rivet result: %w: %s", err, strings.TrimSpace(diagnostics.String()))
	}
	if result.Summary != "" {
		fmt.Fprintln(stdout, result.Summary)
	}
	if runErr != nil {
		return "", fmt.Errorf("rivet %s exited %d: %w", args[0], result.ExitCode, runErr)
	}
	if result.ExitCode != 0 {
		return "", fmt.Errorf("rivet %s exited %d", args[0], result.ExitCode)
	}
	if result.LogPath == "" {
		return "", errors.New("Rivet returned an empty raw log path")
	}
	if _, err := os.Stat(result.LogPath); err != nil {
		return "", fmt.Errorf("Rivet raw log unavailable: %w", err)
	}
	return result.LogPath, nil
}

func workPublishPR(ctx context.Context, worktree, branch, baseBranch string, ticket workTicket, checkpoint workCheckpoint) (string, error) {
	if _, err := workGit(worktree, "push", "-u", "origin", branch); err != nil {
		return "", err
	}
	args := []string{"pr", "create", "--draft", "--base", baseBranch, "--head", branch,
		"--title", ticket.Title, "--body", fmt.Sprintf("Resolve ticket: %s\nRecord checkpoint: %s\nVerified commit: %s", ticket.ID, checkpoint.ID, checkpoint.Commit)}
	c := exec.CommandContext(ctx, "gh", args...)
	c.Dir = worktree
	out, err := c.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gh pr create: %w: %s", err, strings.TrimSpace(string(out)))
	}
	url := strings.TrimSpace(string(out))
	if url == "" {
		return "", errors.New("gh pr create returned no PR URL")
	}
	return url, nil
}
