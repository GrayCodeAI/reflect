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

type factoryWorkspaceEntry struct {
	Name        string `json:"name"`
	Role        string `json:"role"`
	Path        string `json:"path"`
	Present     bool   `json:"present"`
	Clean       bool   `json:"clean"`
	Head        string `json:"head,omitempty"`
	Verify      string `json:"verify,omitempty"`
	Tickets     int    `json:"tickets"`
	Checkpoints int    `json:"checkpoints"`
	Messages    int    `json:"messages"`
	Error       string `json:"error,omitempty"`
}

type factoryWorkspaceStatus struct {
	Root         string                  `json:"root"`
	Repositories []factoryWorkspaceEntry `json:"repositories"`
	Present      int                     `json:"present"`
	Clean        int                     `json:"clean"`
	Tickets      int                     `json:"tickets"`
	Checkpoints  int                     `json:"checkpoints"`
}

var factoryWorkspaceRoot string

var factoryWorkspaceRepos = []struct{ name, role string }{
	{"reflect", "agent and workflow"},
	{"relay", "model transport library"},
	{"resolve", "tickets"},
	{"record", "checkpoints"},
	{"recall", "memory"},
	{"rally", "conversation"},
	{"rivet", "verification logs"},
	{"rein", "failure policy"},
	{"ridge", "terminal dashboard"},
	{"graycode-platform", "company platform"},
	{"graycode-logo", "brand assets"},
}

func init() {
	workspace := &cobra.Command{Use: "workspace", Short: "Inspect and verify every GrayCode checkout"}
	workspace.PersistentFlags().StringVar(&factoryWorkspaceRoot, "root", "..", "directory containing the GrayCode repositories")
	workspace.AddCommand(&cobra.Command{Use: "status", Short: "Show all repository states as JSON", RunE: runFactoryWorkspaceStatus})
	workspace.AddCommand(&cobra.Command{Use: "summary", Short: "Show an all-repository summary for Ridge", RunE: runFactoryWorkspaceSummary})
	workspace.AddCommand(&cobra.Command{Use: "verify NAME|all", Short: "Run the local verification gate for one or all repositories", Args: cobra.ExactArgs(1), RunE: runFactoryWorkspaceVerify})
	factoryCmd.AddCommand(workspace)
}

func factoryWorkspacePath() (string, error) {
	root, err := filepath.Abs(factoryWorkspaceRoot)
	if err != nil {
		return "", err
	}
	if !factoryFileExists(filepath.Join(root, "reflect", "go.mod")) || !factoryFileExists(filepath.Join(root, "ridge", "Cargo.toml")) {
		return "", fmt.Errorf("%s is not a GrayCode workspace", root)
	}
	return root, nil
}

func collectFactoryWorkspace(ctx context.Context) (factoryWorkspaceStatus, error) {
	root, err := factoryWorkspacePath()
	if err != nil {
		return factoryWorkspaceStatus{}, err
	}
	status := factoryWorkspaceStatus{Root: root, Repositories: make([]factoryWorkspaceEntry, 0, len(factoryWorkspaceRepos))}
	for _, repo := range factoryWorkspaceRepos {
		path := filepath.Join(root, repo.name)
		entry := factoryWorkspaceEntry{Name: repo.name, Role: repo.role, Path: path}
		gitRoot, err := workGit(path, "rev-parse", "--show-toplevel")
		if err != nil || gitRoot != path {
			entry.Error = "checkout missing or not a separate Git repository"
			status.Repositories = append(status.Repositories, entry)
			continue
		}
		entry.Present = true
		status.Present++
		entry.Verify = factoryVerification(path)
		if entry.Verify == "" {
			entry.Error = "no verification command"
		} else if _, err := exec.LookPath(strings.Fields(entry.Verify)[0]); err != nil {
			entry.Error = strings.Fields(entry.Verify)[0] + " is not on PATH"
		}
		if repo.name == "reflect" {
			entry.Verify += " (local Relay overlay)"
		}
		if head, err := workGit(path, "rev-parse", "--short", "HEAD"); err == nil {
			entry.Head = head
		}
		if changes, err := workGit(path, "status", "--porcelain"); err == nil && changes == "" {
			entry.Clean = true
			status.Clean++
		}
		local, err := collectFactoryStatusAt(ctx, path)
		if err != nil {
			if entry.Error != "" {
				entry.Error += "; "
			}
			entry.Error += err.Error()
		} else {
			entry.Tickets = len(local.Tickets)
			entry.Checkpoints = local.VerifiedCheckpoints
			entry.Messages = local.Messages
			status.Tickets += entry.Tickets
			status.Checkpoints += entry.Checkpoints
		}
		status.Repositories = append(status.Repositories, entry)
	}
	return status, nil
}

func runFactoryWorkspaceStatus(cmd *cobra.Command, _ []string) error {
	status, err := collectFactoryWorkspace(cmd.Context())
	if err != nil {
		return err
	}
	return json.NewEncoder(cmd.OutOrStdout()).Encode(status)
}

func runFactoryWorkspaceSummary(cmd *cobra.Command, _ []string) error {
	status, err := collectFactoryWorkspace(cmd.Context())
	if err != nil {
		return err
	}
	var attention []string
	for _, repo := range status.Repositories {
		if !repo.Present || !repo.Clean || repo.Error != "" {
			attention = append(attention, repo.Name)
		}
	}
	if len(attention) == 0 {
		attention = append(attention, "none")
	}
	cmd.Printf("Factory workspace: %d/%d repos present, %d clean; %d tickets; %d verified checkpoints; attention: %s\n", status.Present, len(status.Repositories), status.Clean, status.Tickets, status.Checkpoints, strings.Join(attention, ", "))
	return nil
}

func runFactoryWorkspaceVerify(cmd *cobra.Command, args []string) error {
	root, err := factoryWorkspacePath()
	if err != nil {
		return err
	}
	selected := args[0]
	if selected != "all" {
		found := false
		for _, repo := range factoryWorkspaceRepos {
			found = found || repo.name == selected
		}
		if !found {
			return fmt.Errorf("unknown repository %q", selected)
		}
	}
	var failed []string
	for _, repo := range factoryWorkspaceRepos {
		if selected != "all" && selected != repo.name {
			continue
		}
		path := filepath.Join(root, repo.name)
		if gitRoot, err := workGit(path, "rev-parse", "--show-toplevel"); err != nil || gitRoot != path {
			failed = append(failed, repo.name+": checkout missing")
			continue
		}
		verify := factoryVerification(path)
		if verify == "" {
			failed = append(failed, repo.name+": no verification command")
			continue
		}
		cmd.Printf("Verifying %s: %s\n", repo.name, verify)
		if repo.name == "reflect" {
			if err := verifyReflectWithLocalRelay(cmd, path, filepath.Join(root, "relay")); err != nil {
				failed = append(failed, fmt.Sprintf("%s: %v", repo.name, err))
			}
			continue
		}
		check := workVerificationCommand(cmd.Context(), verify)
		check.Dir, check.Stdout, check.Stderr = path, cmd.OutOrStdout(), cmd.ErrOrStderr()
		if err := check.Run(); err != nil {
			failed = append(failed, fmt.Sprintf("%s: %v", repo.name, err))
		}
	}
	if len(failed) > 0 {
		return errors.New(strings.Join(failed, "; "))
	}
	return nil
}

func verifyReflectWithLocalRelay(cmd *cobra.Command, reflectPath, relayPath string) error {
	if !factoryFileExists(filepath.Join(relayPath, "go.mod")) {
		return errors.New("local Relay checkout missing")
	}
	overlay, err := os.MkdirTemp("", "graycode-relay-overlay-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(overlay) }()
	for _, file := range []struct{ source, dest string }{{"go.mod", "reflect.mod"}, {"go.sum", "reflect.sum"}} {
		data, err := os.ReadFile(filepath.Join(reflectPath, file.source))
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(overlay, file.dest), data, 0o600); err != nil {
			return err
		}
	}
	goBin := os.Getenv("GO_BIN")
	if goBin == "" {
		goBin = "go"
	}
	modfile := filepath.Join(overlay, "reflect.mod")
	edit := exec.CommandContext(cmd.Context(), goBin, "mod", "edit", "-modfile="+modfile, "-replace=github.com/GrayCodeAI/relay="+relayPath) // #nosec G204 -- goBin from GO_BIN env or "go"; modfile/relayPath from trusted local paths
	edit.Dir, edit.Env, edit.Stdout, edit.Stderr = reflectPath, append(os.Environ(), "GOWORK=off"), cmd.OutOrStdout(), cmd.ErrOrStderr()
	if err := edit.Run(); err != nil {
		return fmt.Errorf("prepare local Relay overlay: %w", err)
	}
	test := exec.CommandContext(cmd.Context(), goBin, "test", "-modfile="+modfile, "./...") // #nosec G204 -- goBin from GO_BIN env or "go"; modfile from trusted local path
	test.Dir, test.Env, test.Stdout, test.Stderr = reflectPath, append(os.Environ(), "GOWORK=off"), cmd.OutOrStdout(), cmd.ErrOrStderr()
	return test.Run()
}
