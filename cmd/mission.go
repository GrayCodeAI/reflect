package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/GrayCodeAI/reflect/internal/engine/safety"

	reflectconfig "github.com/GrayCodeAI/reflect/internal/config"
	mission "github.com/GrayCodeAI/reflect/internal/multiagent"
	"github.com/GrayCodeAI/reflect/internal/observability/logger"
	"github.com/GrayCodeAI/reflect/internal/textutil"
	"github.com/GrayCodeAI/reflect/internal/tool"
	"github.com/GrayCodeAI/reflect/internal/ui/icons"
	"github.com/spf13/cobra"
)

var (
	missionWorkers int
	missionModel   string
	missionAuto    string
	missionTimeout time.Duration
	missionDryRun  bool
	missionGasp    bool

	missionPRD           bool
	missionVerify        bool
	missionFixRounds     int
	missionVerifyCommand string
)

var missionCmd = &cobra.Command{
	Use:   "mission [prompt]",
	Short: "Run a multi-agent mission (parallel feature execution)",
	Long: `Decompose a task into features and execute them in parallel git worktrees.

Each feature runs in its own worktree with a full engine session.
Results are committed on separate branches for review/merge.

With --prd and --verify the mission runs the full staged goal loop:
a product requirements document per feature, parallel execution,
verification (read-only validator, or --verify-command), and bounded
fix rounds per failed feature.

Examples:
  reflect mission "Add auth, rate limiting, and logging to the API"
  reflect mission --workers 6 "Refactor the database layer into 3 services"
  reflect mission --model claude-sonnet-4-6 "Add tests for all untested packages"
  reflect mission --prd --verify "Harden the API and prove it with tests"
  reflect mission --from-tasks`,
	Args: cobra.ArbitraryArgs,
	RunE: runMission,
}

func init() {
	missionCmd.Flags().IntVar(&missionWorkers, "workers", 4, "Max parallel workers")
	missionCmd.Flags().StringVarP(&missionModel, "model", "m", "", "Model for workers")
	missionCmd.Flags().StringVar(&missionAuto, "auto", "full", "Autonomy level for workers")
	missionCmd.Flags().DurationVar(&missionTimeout, "timeout", 30*time.Minute, "Mission timeout")
	missionCmd.Flags().BoolVar(&missionDryRun, "dry-run", false, "Plan only, don't execute workers")
	missionCmd.Flags().BoolVar(&missionGasp, "experimental-gasp", false, "Write an experimental GASP-style portable artifact after the run")
	missionCmd.Flags().BoolVar(&missionPRD, "prd", false, "Generate a product requirements document per feature before execution")
	missionCmd.Flags().BoolVar(&missionVerify, "verify", false, "Verify each completed feature and fix failures (read-only validator, or --verify-command)")
	missionCmd.Flags().IntVar(&missionFixRounds, "fix-rounds", 3, "Fix attempts per failed feature after --verify; 0 disables the fix loop")
	missionCmd.Flags().StringVar(&missionVerifyCommand, "verify-command", "", "Command run in each feature's implementation tree; exit 0 passes verification (requires --verify)")
}

func runMission(_ *cobra.Command, args []string) error {
	prompt, err := missionPrompt(args)
	if err != nil {
		return err
	}
	if strings.TrimSpace(missionVerifyCommand) != "" && !missionVerify {
		return fmt.Errorf("--verify-command requires --verify")
	}

	cwd, _ := os.Getwd()
	baseBranch := getCurrentBranch(cwd)

	settings := reflectconfig.LoadSettings()
	effectiveModel, effectiveProvider := effectiveModelAndProvider(settings)
	if missionModel != "" {
		effectiveModel = missionModel
	}

	autonomy := safety.ParseAutonomyLevel(missionAuto)

	cfg := mission.Config{
		MaxWorkers:        missionWorkers,
		WorkerModel:       effectiveModel,
		RepoDir:           cwd,
		BaseBranch:        baseBranch,
		AutonomyLevel:     int(autonomy),
		EnablePRDPhase:    missionPRD,
		EnableVerifyPhase: missionVerify,
		MaxFixAttempts:    missionFixRounds,
	}

	m := mission.New(prompt, cfg)
	// Clean up the mission's temp directory when the command finishes,
	// whether it succeeded or failed. Without this, /tmp/reflect-missions/
	// accumulates one directory per run indefinitely (C6 fix).
	defer func() { _ = m.Cleanup() }()

	ctx, cancel := context.WithTimeout(context.Background(), missionTimeout)
	defer cancel()

	var waves [][]string
	if missionFromTasks {
		fmt.Printf("%s\n", auditTint(fmt.Sprintf("Mission %s: loading validated task graph...", m.ID), textPrimary))
		features, taskWaves, err := missionFeaturesFromTasks(tool.GetTaskStore(), m.ID)
		if err != nil {
			return fmt.Errorf("task graph: %w", err)
		}
		m.Features = features
		waves = taskWaves
	} else {
		fmt.Printf("%s\n", auditTint(fmt.Sprintf("Mission %s: planning...", m.ID), textPrimary))
		planFn := func(ctx context.Context, p string) ([]mission.Feature, error) {
			features, err := planWithLLM(ctx, p, effectiveProvider, effectiveModel, settings)
			if err != nil || len(features) > 0 {
				return features, err
			}
			// Retry once with a stricter JSON contract if the first pass yields
			// nothing parseable.
			strictPrompt := p + "\n\nReturn ONLY a JSON array of objects with \"description\" fields."
			return planWithLLM(ctx, strictPrompt, effectiveProvider, effectiveModel, settings)
		}
		if err := m.Plan(ctx, planFn); err != nil {
			return fmt.Errorf("planning: %w", err)
		}
	}

	fmt.Printf("%s\n", auditTint(fmt.Sprintf("Mission %s: %d features planned", m.ID, len(m.Features)), textPrimary))
	for i, f := range m.Features {
		fmt.Printf("%s\n", auditTint(fmt.Sprintf("  %d. %s", i+1, f.Description), textPrimary))
	}
	fmt.Println()

	if missionDryRun {
		fmt.Println(auditTint("(dry-run: not executing workers)", textMuted))
		return nil
	}

	// Build system prompt for workers
	systemPrompt, _ := buildSystemPrompt()

	// Run features in parallel
	workerFn := mission.EngineWorker(effectiveProvider, effectiveModel, systemPrompt)
	if missionFromTasks {
		workerFn = graphTrackingWorker(tool.GetTaskStore(), workerFn)
	}

	var prog *CLIProgress
	if !IsQuiet() {
		prog = NewCLIProgress("Mission", []string{fmt.Sprintf("Executing %d features with %d workers", len(m.Features), cfg.MaxWorkers)})
		defer prog.Abort()
		prog.StartStep(0)
	}

	// Staged goal loop: PRD -> execute -> verify -> bounded fix rounds. The
	// engine gates each phase on the mission config, so runs without --prd
	// and --verify keep the legacy parallel-execution behavior.
	stagedOpts := []mission.StagedOption{}
	if missionFromTasks {
		stagedOpts = append(stagedOpts, mission.WithExecutionWaves(waves))
	}
	if missionPRD {
		generate := func(ctx context.Context, feature *mission.Feature) (string, error) {
			return prdWithLLM(ctx, feature, effectiveProvider, effectiveModel, settings)
		}
		stagedOpts = append(stagedOpts, mission.WithPRD(missionPRDFunc(generate)))
	}
	if missionVerify {
		validator := mission.ReadOnlyValidationWorker(effectiveProvider, effectiveModel, systemPrompt)
		stagedOpts = append(stagedOpts, mission.WithVerify(missionVerifyFunc(m, missionVerifyCommand, validator)))
		if missionFixRounds > 0 {
			stagedOpts = append(stagedOpts, mission.WithFix(missionFixFunc(m, workerFn)))
		}
	}
	runErr := m.RunStaged(ctx, workerFn, stagedOpts...)
	if runErr != nil {
		if prog != nil {
			prog.FailStep(0, runErr.Error())
		}
		return runErr
	}
	if prog != nil {
		prog.CompleteStep(0)
		prog.Done()
	}

	if missionGasp {
		if err := m.ExportGASP(); err != nil {
			fmt.Printf("%s\n", auditTint(fmt.Sprintf("warning: GASP export failed: %v", err), textMuted))
		} else {
			fmt.Printf("%s\n", auditTint(fmt.Sprintf("GASP artifact written to %s/mission.gasp.json", m.Dir), textMuted))
		}
	}

	// Print results
	fmt.Println()
	fmt.Println(auditTint(m.Summary(), textPrimary))
	fmt.Println()
	for _, f := range m.Features {
		status := icons.CheckBold() + " "
		statusColor := doneGreen
		if f.Status == mission.FeatureFailed {
			status = icons.CloseThick() + " "
			statusColor = errorCoral
		}
		branch := f.Branch
		if f.Handoff != nil && f.Handoff.CommitID != "" {
			branch += " (" + f.Handoff.CommitID[:7] + ")"
		}
		fmt.Printf("  %s %s\n", auditTint(status, statusColor), auditTint(f.Description, textPrimary)+auditTint(" — "+branch, textMuted))
		if f.Status == mission.FeatureFailed && f.VerificationResult != "" {
			fmt.Printf("      %s\n", auditTint("verification: "+textutil.TruncateAppend(f.VerificationResult, 200), textMuted))
		}
	}
	if missionFromTasks && len(m.WaveJoins) > 0 {
		fmt.Println()
		fmt.Println(auditTint("Wave joins:", textPrimary))
		for _, join := range m.WaveJoins {
			fmt.Printf("  %s\n", auditTint(fmt.Sprintf("%d. %s", join.Wave, join.Summary), textMuted))
		}
	}

	// Goal-loop verdict: map the post-verification mission status to an
	// explicit Completed/Partial/Failed result line. The failure-exit check
	// below then propagates Failed and Partial as a non-zero exit.
	if missionVerify {
		verdict := missionVerdictText(m.Status)
		verdictColor := doneGreen
		switch m.Status {
		case mission.StatusFailed:
			verdictColor = errorCoral
		case mission.StatusPartial:
			verdictColor = warnAmber
		}
		fmt.Println()
		fmt.Printf("%s\n", auditTint("Mission verdict: "+verdict, verdictColor))
	}

	// Propagate feature failures as a non-zero exit so CI sees a real
	// failure instead of green: Mission.Run historically returned nil even
	// when every feature failed (H9), so `reflect mission` exited 0.
	failed := 0
	for _, f := range m.Features {
		if f.Status == mission.FeatureFailed {
			failed++
		}
	}
	if failed > 0 {
		return fmt.Errorf("mission %s: %d/%d features failed", m.ID, failed, len(m.Features))
	}

	return nil
}

// missionPRDFunc adapts a PRD generator for the staged pipeline. The engine
// records the document on the feature; folding it into ExpectedBehavior is
// what makes the PRD phase actionable, because the implementation worker
// builds its prompt from those feature fields.
func missionPRDFunc(generate func(ctx context.Context, feature *mission.Feature) (string, error)) mission.PRDFunc {
	return func(ctx context.Context, feature *mission.Feature) (string, error) {
		prd, err := generate(ctx, feature)
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(feature.ExpectedBehavior) == "" {
			feature.ExpectedBehavior = prd
		} else {
			feature.ExpectedBehavior = strings.TrimSpace(feature.ExpectedBehavior) + "\n\nPRD:\n" + prd
		}
		return prd, nil
	}
}

// missionVerifyFunc builds the staged pipeline's verification function. When
// a verify command is configured it runs against the feature's
// implementation (exit 0 passes); otherwise the read-only validation
// worker's report decides: PASS marker(s) and no FAIL marker.
func missionVerifyFunc(m *mission.Mission, command string, validator mission.WorkerFunc) mission.VerifyFunc {
	return func(ctx context.Context, feature *mission.Feature, handoff *mission.Handoff) (bool, string, error) {
		if handoff == nil {
			// Nothing to verify (e.g. a task-graph feature that was already
			// complete before this run). Skip rather than fail the mission.
			return true, "skipped: no implementation handoff", nil
		}
		if strings.TrimSpace(command) != "" {
			return missionRunVerifyCommand(ctx, command, m.Config.RepoDir, feature, handoff)
		}
		// The implementation worker removes its worktree (and branch) when
		// it returns, but the commit stays in the object store: point the
		// validator at the handed-off commit so it judges the change, not
		// the base branch.
		valCfg := m.Config
		if handoff.CommitID != "" {
			valCfg.BaseBranch = handoff.CommitID
		}
		report, err := validator(ctx, feature, m.Dir, valCfg)
		if err != nil {
			return false, "", err
		}
		if report == nil {
			return false, "", fmt.Errorf("validation worker returned no report for %s", feature.ID)
		}
		return missionVerdictPassed(report.Summary), report.Summary, nil
	}
}

// missionRunVerifyCommand runs the operator's verification command against
// one feature's implementation. Exit 0 passes. It prefers the handed-off
// worktree when it still exists and otherwise checks the handed-off commit
// out into a throwaway detached worktree; the command runs with that tree as
// its working directory.
func missionRunVerifyCommand(ctx context.Context, command, repoDir string, feature *mission.Feature, handoff *mission.Handoff) (bool, string, error) {
	dir := ""
	if handoff.RepoPath != "" {
		if stat, err := os.Stat(handoff.RepoPath); err == nil && stat.IsDir() {
			dir = handoff.RepoPath
		}
	}
	cleanup := func() {}
	if dir == "" && handoff.CommitID != "" {
		tmp, err := os.MkdirTemp("", "reflect-verify-")
		if err != nil {
			return false, "", err
		}
		// #nosec G204 -- fixed git executable; the commit comes from the worker handoff
		wt := exec.CommandContext(ctx, "git", "worktree", "add", "--detach", tmp, handoff.CommitID)
		wt.Dir = repoDir
		if out, err := wt.CombinedOutput(); err != nil {
			_ = os.RemoveAll(tmp)
			return false, "", fmt.Errorf("checkout %s for verification: %w: %s", handoff.CommitID, err, strings.TrimSpace(string(out)))
		}
		dir = tmp
		cleanup = func() {
			removeCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			// #nosec G204 -- fixed git executable; tmp path is internal
			rm := exec.CommandContext(removeCtx, "git", "worktree", "remove", "--force", tmp)
			rm.Dir = repoDir
			_ = rm.Run()
		}
	}
	if dir == "" {
		return false, "", fmt.Errorf("feature %s has no worktree or commit to verify", feature.ID)
	}
	defer cleanup()

	var output bytes.Buffer
	verify := workVerificationCommand(ctx, command)
	verify.Dir, verify.Stdout, verify.Stderr = dir, &output, &output
	err := verify.Run()
	if err == nil {
		return true, fmt.Sprintf("verify command %q: PASS", command), nil
	}
	return false, fmt.Sprintf("verify command %q: FAIL: %v\n%s", command, err, textutil.TruncateAppend(strings.TrimSpace(output.String()), 500)), nil
}

// missionVerdictPassed interprets a read-only validation worker's report.
// The validator is instructed to give a concrete PASS or FAIL per acceptance
// criterion, so a report passes only when it carries at least one PASS
// marker and no FAIL marker; a report with neither is not a pass.
func missionVerdictPassed(report string) bool {
	return strings.Contains(report, "PASS") && !strings.Contains(report, "FAIL")
}

// missionFixFunc adapts the implementation worker into the staged
// pipeline's fix function. It folds the failed verification into the feature
// context (the worker prompt is built from those fields) and continues from
// the handed-off commit so the fix lands on top of the previous attempt
// instead of the base branch.
func missionFixFunc(m *mission.Mission, workerFn mission.WorkerFunc) mission.FixFunc {
	return func(ctx context.Context, feature *mission.Feature, verificationResult string, handoff *mission.Handoff) (*mission.Handoff, error) {
		if handoff == nil {
			return workerFn(ctx, feature, m.Dir, m.Config)
		}
		origBehavior := feature.ExpectedBehavior
		feature.ExpectedBehavior = strings.TrimSpace(origBehavior) + "\n\nThe previous attempt failed verification:\n" + verificationResult
		defer func() { feature.ExpectedBehavior = origBehavior }()
		fixCfg := m.Config
		if handoff.CommitID != "" {
			fixCfg.BaseBranch = handoff.CommitID
		}
		return workerFn(ctx, feature, m.Dir, fixCfg)
	}
}

// missionVerdictText maps the mission status after the staged pipeline to
// the operator-facing goal-loop verdict.
func missionVerdictText(status mission.Status) string {
	switch status {
	case mission.StatusFailed:
		return "Failed"
	case mission.StatusPartial:
		return "Partial"
	default:
		return "Completed"
	}
}

func planWithLLM(ctx context.Context, prompt, provider, model string, settings reflectconfig.Settings) ([]mission.Feature, error) {
	planPrompt := fmt.Sprintf(
		"Decompose this task into independent features that can be implemented in parallel.\n\n"+
			"Task: %s\n\n"+
			"Return ONLY a JSON array of objects, each with a\\\"description\\\" string (and optional\\n"+
			"\\\"expected_behavior\\\" string). Each feature must be:\n"+
			"- Independent (can be implemented without the others)\n"+
			"- Specific (clear what to implement)\n"+
			"- Testable (has clear success criteria)\n\n"+
			"Do not include any text outside the JSON array.",
		prompt,
	)

	registry, _ := defaultRegistry(settings)
	sess, err := newConfiguredReflectSession(settings, provider, model, planPrompt, registry, logger.New(io.Discard, logger.Error))
	if err != nil {
		return nil, err
	}
	_ = sess.SetMaxTurns(1)
	sess.PermSvc().SetPermissionFn(func(req safety.PermissionRequest) {
		if req.Response != nil {
			req.Response <- true
		}
	})

	sess.AddUser(planPrompt)
	events, err := sess.Stream(ctx)
	if err != nil {
		return nil, err
	}

	var response strings.Builder
	for ev := range events {
		if ev.Type == "content" {
			response.WriteString(ev.Content)
		}
	}

	return parseFeatures(response.String()), nil
}

// prdWithLLM generates a product requirements document for one feature via a
// single-turn LLM call, mirroring planWithLLM's session setup.
func prdWithLLM(ctx context.Context, feature *mission.Feature, provider, model string, settings reflectconfig.Settings) (string, error) {
	prdPrompt := fmt.Sprintf(
		"Write a concise product requirements document for one feature of a larger mission.\n\n"+
			"Feature: %s\nDescription: %s\nExpected behavior: %s\n\n"+
			"Cover the goal, acceptance criteria (each independently checkable), and constraints. "+
			"Keep it under 300 words. Do not include any text outside the document.",
		feature.ID, feature.Description, feature.ExpectedBehavior,
	)

	registry, _ := defaultRegistry(settings)
	sess, err := newConfiguredReflectSession(settings, provider, model, prdPrompt, registry, logger.New(io.Discard, logger.Error))
	if err != nil {
		return "", err
	}
	_ = sess.SetMaxTurns(1)
	sess.PermSvc().SetPermissionFn(func(req safety.PermissionRequest) {
		if req.Response != nil {
			req.Response <- true
		}
	})

	sess.AddUser(prdPrompt)
	events, err := sess.Stream(ctx)
	if err != nil {
		return "", err
	}

	var response strings.Builder
	for ev := range events {
		if ev.Type == "content" {
			response.WriteString(ev.Content)
		}
	}

	return strings.TrimSpace(response.String()), nil
}

func parseFeatures(text string) []mission.Feature {
	text = strings.TrimSpace(text)
	// Prefer structured JSON when the model emits it. This keeps descriptions
	// (and optional metadata) out of fragile line parsing and avoids dropped
	// features when a description contains newlines.
	if strings.HasPrefix(text, "[") {
		var parsed []struct {
			Description      string `json:"description"`
			ExpectedBehavior string `json:"expected_behavior,omitempty"`
			OutputSchema     string `json:"output_schema,omitempty"`
		}
		if err := json.Unmarshal([]byte(text), &parsed); err == nil && len(parsed) > 0 {
			var features []mission.Feature
			for _, p := range parsed {
				if strings.TrimSpace(p.Description) == "" {
					continue
				}
				features = append(features, mission.Feature{
					Description:      p.Description,
					ExpectedBehavior: p.ExpectedBehavior,
					OutputSchema:     p.OutputSchema,
				})
			}
			if len(features) > 0 {
				return features
			}
		}
	}

	var features []mission.Feature
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Strip numbering: "1. ", "1) ", "- "
		for _, prefix := range []string{"- ", "* "} {
			line = strings.TrimPrefix(line, prefix)
		}
		if len(line) > 2 && line[0] >= '0' && line[0] <= '9' {
			idx := strings.IndexAny(line, ".)")
			if idx > 0 && idx < 4 {
				line = strings.TrimSpace(line[idx+1:])
			}
		}
		if line == "" {
			continue
		}
		features = append(features, mission.Feature{
			Description: line,
		})
	}
	return features
}

func getCurrentBranch(dir string) string {
	cmd := exec.CommandContext(context.Background(), "git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "main"
	}
	return strings.TrimSpace(string(out))
}
