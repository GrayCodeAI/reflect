package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

type factoryTicket struct {
	ID                  string `json:"id"`
	Title               string `json:"title"`
	Status              string `json:"status"`
	Assignee            string `json:"assignee,omitempty"`
	Checkpoints         int    `json:"checkpoints"`
	VerifiedCheckpoints int    `json:"verified_checkpoints"`
	Memories            int    `json:"memories"`
	Messages            int    `json:"messages"`
	LastMessage         string `json:"last_message,omitempty"`
	ReadyForReview      bool   `json:"ready_for_review"`
}

type factoryStatus struct {
	Repository          string          `json:"repository"`
	Tickets             []factoryTicket `json:"tickets"`
	Checkpoints         int             `json:"checkpoints"`
	VerifiedCheckpoints int             `json:"verified_checkpoints"`
	Memories            int             `json:"memories"`
	Messages            int             `json:"messages"`
}

type factoryCheckpoint struct {
	ID           string   `json:"id"`
	TaskID       string   `json:"task_id"`
	Commit       string   `json:"commit"`
	Verification []string `json:"verification"`
}

type factoryMessage struct {
	TicketID string `json:"ticket_id"`
	Body     string `json:"body"`
}

type factoryCheck struct {
	Name   string `json:"name"`
	Ready  bool   `json:"ready"`
	Detail string `json:"detail"`
}

var (
	factoryRepo        string
	factoryResolveBin  string
	factoryRecordBin   string
	factoryRecallBin   string
	factoryRallyBin    string
	factoryRivetBin    string
	factoryReinBin     string
	factoryRecallStore string
	factoryRallyData   string
	factoryRunRequest  string
	factoryRunVerify   string
	factoryRunPR       bool
)

var factoryCmd = &cobra.Command{
	Use:   "factory",
	Short: "Inspect the local ticket-to-review software factory",
}

func init() {
	factoryCmd.PersistentFlags().StringVar(&factoryRepo, "repo", ".", "Git repository containing Resolve tickets")
	factoryCmd.PersistentFlags().StringVar(&factoryResolveBin, "resolve-bin", "", "Resolve CLI path")
	factoryCmd.PersistentFlags().StringVar(&factoryRecordBin, "record-bin", "", "Record CLI path")
	factoryCmd.PersistentFlags().StringVar(&factoryRecallBin, "recall-bin", "", "Recall CLI path")
	factoryCmd.PersistentFlags().StringVar(&factoryRallyBin, "rally-bin", "", "Rally CLI path")
	factoryCmd.PersistentFlags().StringVar(&factoryRivetBin, "rivet-bin", "", "Rivet CLI path")
	factoryCmd.PersistentFlags().StringVar(&factoryReinBin, "rein-bin", "", "Rein CLI path")
	factoryCmd.PersistentFlags().StringVar(&factoryRecallStore, "recall-store", "", "Recall data directory")
	factoryCmd.PersistentFlags().StringVar(&factoryRallyData, "rally-data", "", "Rally data directory")
	factoryCmd.AddCommand(&cobra.Command{Use: "status", Short: "Show tickets and linked evidence as JSON", RunE: runFactoryStatus})
	factoryCmd.AddCommand(&cobra.Command{Use: "summary", Short: "Show a one-line status for Ridge", RunE: runFactorySummary})
	factoryCmd.AddCommand(&cobra.Command{Use: "doctor", Short: "Check local factory tools and worktree readiness", RunE: runFactoryDoctor})
	run := &cobra.Command{Use: "run [ticket ID]", Short: "Run a ticket through Reflect, Rivet, Record, Recall, Resolve, and Rally", Args: cobra.MaximumNArgs(1), RunE: runFactoryWork}
	run.Flags().StringVar(&factoryRunRequest, "request", "", "plain request; creates a Resolve ticket")
	run.Flags().StringVar(&factoryRunVerify, "verify", "", "verification command (default: inferred from project manifest)")
	run.Flags().BoolVar(&factoryRunPR, "pr", false, "push the verified branch and open a draft GitHub PR")
	factoryCmd.AddCommand(run)
	rootCmd.AddCommand(factoryCmd)
}

func factoryVerification(root string) string {
	switch {
	case factoryFileExists(filepath.Join(root, "deployment", "vaultly.json")) && factoryFileExists(filepath.Join(root, "pnpm-lock.yaml")):
		return "pnpm check"
	case factoryFileExists(filepath.Join(root, "scripts", "check-assets.py")):
		return "python3 scripts/check-assets.py"
	case factoryFileExists(filepath.Join(root, "go.mod")):
		return "go test ./..."
	case factoryFileExists(filepath.Join(root, "Cargo.toml")):
		return "cargo test"
	case factoryFileExists(filepath.Join(root, "pnpm-lock.yaml")):
		return "pnpm test"
	case factoryFileExists(filepath.Join(root, "package-lock.json")):
		return "npm test"
	default:
		return ""
	}
}

func factoryFileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func runFactoryWork(cmd *cobra.Command, args []string) error {
	if (len(args) == 0) == (strings.TrimSpace(factoryRunRequest) == "") {
		return errors.New("provide either one ticket ID or --request")
	}
	root, err := factoryRepoRoot()
	if err != nil {
		return err
	}
	verify := strings.TrimSpace(factoryRunVerify)
	if verify == "" {
		verify = factoryVerification(root)
	}
	if verify == "" {
		return errors.New("no project verification found; pass --verify COMMAND")
	}
	rivetBin, err := workFindBinary(factoryRivetBin, "rivet")
	if err != nil {
		return err
	}
	oldRepo, oldResolve, oldRecord, oldRecall := workRepo, workResolveBin, workRecordBin, workRecallBin
	oldRally, oldRivet, oldRecallStore, oldRallyData := workRallyBin, workRivetBin, workRecallStore, workRallyData
	oldRequest, oldVerify, oldVerifyBlock, oldPR := workRequest, workVerify, workVerifyBlock, workOpenPR
	defer func() {
		workRepo, workResolveBin, workRecordBin, workRecallBin = oldRepo, oldResolve, oldRecord, oldRecall
		workRallyBin, workRivetBin, workRecallStore, workRallyData = oldRally, oldRivet, oldRecallStore, oldRallyData
		workRequest, workVerify, workVerifyBlock, workOpenPR = oldRequest, oldVerify, oldVerifyBlock, oldPR
	}()
	workRepo, workResolveBin, workRecordBin, workRecallBin = root, factoryResolveBin, factoryRecordBin, factoryRecallBin
	workRallyBin, workRivetBin, workRecallStore, workRallyData = factoryRallyBin, rivetBin, factoryRecallStore, factoryRallyData
	workRequest, workVerify, workVerifyBlock, workOpenPR = factoryRunRequest, verify, "", factoryRunPR
	return runWork(cmd, args)
}

func factoryRepoRoot() (string, error) {
	path, err := filepath.Abs(factoryRepo)
	if err != nil {
		return "", err
	}
	root, err := workGit(path, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("find Git repository: %w", err)
	}
	return root, nil
}

func factoryBins() (map[string]string, error) {
	requested := map[string]string{
		"resolve": factoryResolveBin,
		"record":  factoryRecordBin,
		"recall":  factoryRecallBin,
		"rally":   factoryRallyBin,
	}
	bins := make(map[string]string, len(requested))
	for name, explicit := range requested {
		path, err := workFindBinary(explicit, name)
		if err != nil {
			return nil, err
		}
		bins[name] = path
	}
	return bins, nil
}

func collectFactoryStatus(ctx context.Context) (factoryStatus, error) {
	root, err := factoryRepoRoot()
	if err != nil {
		return factoryStatus{}, err
	}
	return collectFactoryStatusAt(ctx, root)
}

func collectFactoryStatusAt(ctx context.Context, root string) (factoryStatus, error) {
	bins, err := factoryBins()
	if err != nil {
		return factoryStatus{}, err
	}
	var tickets []struct {
		ID       string `json:"id"`
		Title    string `json:"title"`
		Status   string `json:"status"`
		Assignee string `json:"assignee"`
	}
	if err := workJSON(ctx, bins["resolve"], []string{"list", "--repo", root}, &tickets); err != nil {
		return factoryStatus{}, fmt.Errorf("resolve tickets: %w", err)
	}
	var checkpoints []factoryCheckpoint
	if err := workJSON(ctx, bins["record"], []string{"list", "--repo", root}, &checkpoints); err != nil {
		return factoryStatus{}, fmt.Errorf("record checkpoints: %w", err)
	}
	scope := filepath.Base(root)
	recallArgs := []string{"list", "--scope", scope}
	if factoryRecallStore != "" {
		recallArgs = append(recallArgs, "--store", factoryRecallStore)
	}
	var memories []workMemory
	if err := workJSON(ctx, bins["recall"], recallArgs, &memories); err != nil {
		return factoryStatus{}, fmt.Errorf("recall outcomes: %w", err)
	}
	rallyArgs := []string{"list", "--channel", scope}
	if factoryRallyData != "" {
		rallyArgs = append(rallyArgs, "--data", factoryRallyData)
	}
	var messages []factoryMessage
	if err := workJSON(ctx, bins["rally"], rallyArgs, &messages); err != nil {
		return factoryStatus{}, fmt.Errorf("rally messages: %w", err)
	}
	status := factoryStatus{Repository: root, Tickets: make([]factoryTicket, 0, len(tickets)), Checkpoints: len(checkpoints), Memories: len(memories), Messages: len(messages)}
	checkpointByTask := make(map[string][]factoryCheckpoint)
	for _, checkpoint := range checkpoints {
		checkpointByTask[checkpoint.TaskID] = append(checkpointByTask[checkpoint.TaskID], checkpoint)
		if checkpoint.Commit != "" && len(checkpoint.Verification) > 0 {
			status.VerifiedCheckpoints++
		}
	}
	messagesByTicket := make(map[string]int)
	lastMessageByTicket := make(map[string]string)
	for _, message := range messages {
		messagesByTicket[message.TicketID]++
		lastMessageByTicket[message.TicketID] = message.Body
	}
	for _, ticket := range tickets {
		item := factoryTicket{ID: ticket.ID, Title: ticket.Title, Status: ticket.Status, Assignee: ticket.Assignee, Messages: messagesByTicket[ticket.ID], LastMessage: lastMessageByTicket[ticket.ID]}
		linked := checkpointByTask[ticket.ID]
		item.Checkpoints = len(linked)
		for _, checkpoint := range linked {
			if checkpoint.Commit != "" && len(checkpoint.Verification) > 0 {
				item.VerifiedCheckpoints++
			}
		}
		for _, memory := range memories {
			for _, source := range memory.Sources {
				for _, checkpoint := range linked {
					if source == "checkpoint:"+checkpoint.ID || (checkpoint.Commit != "" && source == "commit:"+checkpoint.Commit) {
						item.Memories++
						goto nextMemory
					}
				}
			}
		nextMemory:
		}
		item.ReadyForReview = item.Status == "in_review" && item.VerifiedCheckpoints > 0
		status.Tickets = append(status.Tickets, item)
	}
	return status, nil
}

func runFactoryStatus(cmd *cobra.Command, _ []string) error {
	status, err := collectFactoryStatus(cmd.Context())
	if err != nil {
		return err
	}
	return json.NewEncoder(cmd.OutOrStdout()).Encode(status)
}

func factorySummary(status factoryStatus) string {
	counts := map[string]int{}
	stopped := 0
	for _, ticket := range status.Tickets {
		counts[ticket.Status]++
		if strings.HasPrefix(ticket.LastMessage, "Work stopped during ") {
			stopped++
		}
	}
	return fmt.Sprintf("Factory: %d tickets (open %d, working %d, review %d, done %d, stopped %d); %d verified checkpoints; %d memories; %d messages", len(status.Tickets), counts["open"], counts["in_progress"], counts["in_review"], counts["done"], stopped, status.VerifiedCheckpoints, status.Memories, status.Messages)
}

func runFactorySummary(cmd *cobra.Command, _ []string) error {
	status, err := collectFactoryStatus(cmd.Context())
	if err != nil {
		return err
	}
	cmd.Println(factorySummary(status))
	return nil
}

func runFactoryDoctor(cmd *cobra.Command, _ []string) error {
	checks := make([]factoryCheck, 0, 9)
	root, err := factoryRepoRoot()
	if err != nil {
		checks = append(checks, factoryCheck{Name: "repository", Detail: err.Error()})
	} else {
		checks = append(checks, factoryCheck{Name: "repository", Ready: true, Detail: root})
		changed, err := workGit(root, "status", "--porcelain")
		checks = append(checks, factoryCheck{Name: "clean worktree", Ready: err == nil && changed == "", Detail: changed})
	}
	for _, entry := range []struct{ name, explicit string }{{"resolve", factoryResolveBin}, {"record", factoryRecordBin}, {"recall", factoryRecallBin}, {"rally", factoryRallyBin}, {"rivet", factoryRivetBin}, {"rein", factoryReinBin}} {
		path, err := workFindBinary(entry.explicit, entry.name)
		if err == nil {
			info, statErr := os.Stat(path)
			if statErr != nil || info.IsDir() {
				err = fmt.Errorf("%s is not an executable file", path)
			}
		}
		if err != nil {
			checks = append(checks, factoryCheck{Name: entry.name, Detail: err.Error()})
		} else {
			checks = append(checks, factoryCheck{Name: entry.name, Ready: true, Detail: path})
		}
	}
	if path, err := exec.LookPath("ridge"); err == nil {
		checks = append(checks, factoryCheck{Name: "ridge", Ready: true, Detail: path})
	} else {
		checks = append(checks, factoryCheck{Name: "ridge", Detail: "install Ridge for the terminal dashboard"})
	}
	failed := false
	for _, check := range checks {
		state := "ok"
		if !check.Ready {
			state, failed = "missing", true
		}
		cmd.Printf("%-16s %-8s %s\n", check.Name, state, strings.TrimSpace(check.Detail))
	}
	if failed {
		return errors.New("factory is not ready; fix the checks above")
	}
	return nil
}
