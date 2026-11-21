package cmd

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"testing"

	mission "github.com/GrayCodeAI/reflect/internal/multiagent"
)

func TestMissionStagedFlags_Defaults(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"prd", "false"},
		{"verify", "false"},
		{"fix-rounds", "3"},
		{"verify-command", ""},
	}
	for _, tt := range tests {
		flag := missionCmd.Flags().Lookup(tt.name)
		if flag == nil {
			t.Errorf("flag --%s is not registered on mission", tt.name)
			continue
		}
		if flag.DefValue != tt.want {
			t.Errorf("--%s default = %q, want %q", tt.name, flag.DefValue, tt.want)
		}
	}
}

func TestMissionStagedFlags_Parse(t *testing.T) {
	oldPRD, oldVerify, oldRounds, oldCommand := missionPRD, missionVerify, missionFixRounds, missionVerifyCommand
	defer func() {
		missionPRD, missionVerify, missionFixRounds, missionVerifyCommand = oldPRD, oldVerify, oldRounds, oldCommand
	}()

	if err := missionCmd.Flags().Parse([]string{
		"--prd", "--verify",
		"--fix-rounds", "2",
		"--verify-command", "go test ./...",
	}); err != nil {
		t.Fatalf("parse flags: %v", err)
	}
	if !missionPRD {
		t.Error("--prd not applied")
	}
	if !missionVerify {
		t.Error("--verify not applied")
	}
	if missionFixRounds != 2 {
		t.Errorf("--fix-rounds = %d, want 2", missionFixRounds)
	}
	if missionVerifyCommand != "go test ./..." {
		t.Errorf("--verify-command = %q, want %q", missionVerifyCommand, "go test ./...")
	}
}

func TestMissionVerdictPassed(t *testing.T) {
	tests := []struct {
		name   string
		report string
		want   bool
	}{
		{"single pass", "criterion 1: PASS (api/auth.go:42)", true},
		{"multiple passes", "PASS\nPASS", true},
		{"pass and fail", "criterion 1: PASS (a.go:1)\ncriterion 2: FAIL (b.go:2 missing handler)", false},
		{"fail only", "FAIL: no tests found", false},
		{"uppercase failed", "criterion 1: FAILED", false},
		{"empty report", "", false},
		{"no markers", "the code looks reasonable", false},
	}
	for _, tt := range tests {
		if got := missionVerdictPassed(tt.report); got != tt.want {
			t.Errorf("%s: missionVerdictPassed(%q) = %v, want %v", tt.name, tt.report, got, tt.want)
		}
	}
}

func TestMissionVerdictText(t *testing.T) {
	tests := []struct {
		status mission.Status
		want   string
	}{
		{mission.StatusCompleted, "Completed"},
		{mission.StatusPartial, "Partial"},
		{mission.StatusFailed, "Failed"},
		{mission.StatusRunning, "Completed"},
	}
	for _, tt := range tests {
		if got := missionVerdictText(tt.status); got != tt.want {
			t.Errorf("missionVerdictText(%q) = %q, want %q", tt.status, got, tt.want)
		}
	}
}

func TestMissionPRDFunc(t *testing.T) {
	feature := &mission.Feature{ID: "feat-1", Description: "do A", ExpectedBehavior: "A works"}
	prd, err := missionPRDFunc(func(context.Context, *mission.Feature) (string, error) {
		return "goals and criteria", nil
	})(context.Background(), feature)
	if err != nil {
		t.Fatalf("PRD func: %v", err)
	}
	if prd != "goals and criteria" {
		t.Errorf("prd = %q", prd)
	}
	if !strings.Contains(feature.ExpectedBehavior, "A works") || !strings.Contains(feature.ExpectedBehavior, "PRD:\ngoals and criteria") {
		t.Errorf("PRD not folded into expected behavior: %q", feature.ExpectedBehavior)
	}
}

func TestMissionPRDFunc_EmptyBehavior(t *testing.T) {
	feature := &mission.Feature{ID: "feat-1", Description: "do A"}
	_, err := missionPRDFunc(func(context.Context, *mission.Feature) (string, error) {
		return "the only spec", nil
	})(context.Background(), feature)
	if err != nil {
		t.Fatalf("PRD func: %v", err)
	}
	if feature.ExpectedBehavior != "the only spec" {
		t.Errorf("expected PRD as behavior, got %q", feature.ExpectedBehavior)
	}
}

func TestMissionPRDFunc_Error(t *testing.T) {
	feature := &mission.Feature{ID: "feat-1", ExpectedBehavior: "A works"}
	_, err := missionPRDFunc(func(context.Context, *mission.Feature) (string, error) {
		return "", errors.New("model unavailable")
	})(context.Background(), feature)
	if err == nil {
		t.Fatal("expected generator error to propagate")
	}
	if feature.ExpectedBehavior != "A works" {
		t.Errorf("behavior mutated on error: %q", feature.ExpectedBehavior)
	}
}

func TestMissionVerifyFunc_ValidatorVerdict(t *testing.T) {
	tests := []struct {
		name     string
		report   string
		wantPass bool
	}{
		{name: "validator passes", report: "criterion 1: PASS (a.go:1)", wantPass: true},
		{name: "validator fails", report: "criterion 1: FAIL (a.go:1)", wantPass: false},
	}
	for _, tt := range tests {
		m := mission.New("test", mission.Config{RepoDir: "/repo", BaseBranch: "main"})
		m.Dir = t.TempDir()
		handoff := &mission.Handoff{CommitID: "abc123def"}
		validator := func(context.Context, *mission.Feature, string, mission.Config) (*mission.Handoff, error) {
			return &mission.Handoff{Summary: tt.report}, nil
		}
		passed, result, err := missionVerifyFunc(m, "", validator)(context.Background(), &mission.Feature{ID: "feat-1"}, handoff)
		if err != nil {
			t.Fatalf("%s: err = %v", tt.name, err)
		}
		if passed != tt.wantPass {
			t.Errorf("%s: passed = %v, want %v", tt.name, passed, tt.wantPass)
		}
		if result != tt.report {
			t.Errorf("%s: result = %q, want validator report", tt.name, result)
		}
	}
}

func TestMissionVerifyFunc_ValidatorSeesImplementationCommit(t *testing.T) {
	m := mission.New("test", mission.Config{RepoDir: "/repo", BaseBranch: "main"})
	m.Dir = t.TempDir()
	handoff := &mission.Handoff{CommitID: "implcommit123"}
	var gotCfg mission.Config
	validator := func(_ context.Context, _ *mission.Feature, _ string, cfg mission.Config) (*mission.Handoff, error) {
		gotCfg = cfg
		return &mission.Handoff{Summary: "PASS"}, nil
	}
	passed, _, err := missionVerifyFunc(m, "", validator)(context.Background(), &mission.Feature{ID: "feat-1"}, handoff)
	if err != nil {
		t.Fatalf("verify func: %v", err)
	}
	if !passed {
		t.Error("expected PASS report to pass")
	}
	if gotCfg.BaseBranch != "implcommit123" {
		t.Errorf("validator BaseBranch = %q, want the handed-off commit", gotCfg.BaseBranch)
	}
	if gotCfg.RepoDir != "/repo" {
		t.Errorf("validator RepoDir = %q, want /repo", gotCfg.RepoDir)
	}
}

func TestMissionVerifyFunc_ValidatorError(t *testing.T) {
	m := mission.New("test", mission.Config{RepoDir: "/repo"})
	validator := func(context.Context, *mission.Feature, string, mission.Config) (*mission.Handoff, error) {
		return nil, errors.New("stream failed")
	}
	passed, _, err := missionVerifyFunc(m, "", validator)(context.Background(), &mission.Feature{ID: "feat-1"}, &mission.Handoff{})
	if err == nil || passed {
		t.Fatalf("expected validator error, got passed=%v err=%v", passed, err)
	}
}

func TestMissionVerifyFunc_NoHandoffSkips(t *testing.T) {
	m := mission.New("test", mission.Config{RepoDir: "/repo"})
	validator := func(context.Context, *mission.Feature, string, mission.Config) (*mission.Handoff, error) {
		t.Fatal("validator must not run without a handoff")
		return nil, nil
	}
	passed, result, err := missionVerifyFunc(m, "", validator)(context.Background(), &mission.Feature{ID: "feat-1"}, nil)
	if err != nil {
		t.Fatalf("verify func: %v", err)
	}
	if !passed {
		t.Error("feature without handoff should pass (skipped)")
	}
	if !strings.Contains(result, "skipped") {
		t.Errorf("result = %q, want skip note", result)
	}
}

func TestMissionVerifyFunc_VerifyCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test fixture uses POSIX shell")
	}
	m := mission.New("test", mission.Config{RepoDir: "/repo"})
	workdir := t.TempDir()
	feature := &mission.Feature{ID: "feat-1"}
	validator := func(context.Context, *mission.Feature, string, mission.Config) (*mission.Handoff, error) {
		t.Fatal("validator must not run when a verify command is set")
		return nil, nil
	}

	passed, result, err := missionVerifyFunc(m, "exit 0", validator)(context.Background(), feature, &mission.Handoff{RepoPath: workdir})
	if err != nil {
		t.Fatalf("exit 0 should pass: %v", err)
	}
	if !passed {
		t.Error("exit 0 verification must pass")
	}
	if !strings.Contains(result, "PASS") {
		t.Errorf("result = %q, want PASS", result)
	}

	passed, result, err = missionVerifyFunc(m, "echo boom; exit 3", validator)(context.Background(), feature, &mission.Handoff{RepoPath: workdir})
	if err != nil {
		t.Fatalf("command failure should be a verdict, not an error: %v", err)
	}
	if passed {
		t.Error("non-zero exit verification must fail")
	}
	if !strings.Contains(result, "FAIL") || !strings.Contains(result, "boom") {
		t.Errorf("result = %q, want FAIL with command output", result)
	}
}

func TestMissionFixFunc(t *testing.T) {
	m := mission.New("test", mission.Config{RepoDir: "/repo", BaseBranch: "main"})
	m.Dir = t.TempDir()
	feature := &mission.Feature{ID: "feat-1", Description: "do A", ExpectedBehavior: "A works"}
	var seenCfg mission.Config
	var seenBehavior string
	worker := func(_ context.Context, f *mission.Feature, _ string, cfg mission.Config) (*mission.Handoff, error) {
		seenCfg = cfg
		seenBehavior = f.ExpectedBehavior
		return &mission.Handoff{Summary: "fixed", CommitID: "fixcommit456"}, nil
	}

	handoff, err := missionFixFunc(m, worker)(context.Background(), feature, "criterion 1: FAIL (a.go:1)", &mission.Handoff{CommitID: "implcommit123"})
	if err != nil {
		t.Fatalf("fix func: %v", err)
	}
	if handoff == nil || handoff.CommitID != "fixcommit456" {
		t.Fatalf("handoff = %+v, want the fix worker handoff", handoff)
	}
	if seenCfg.BaseBranch != "implcommit123" {
		t.Errorf("fix BaseBranch = %q, want the handed-off commit", seenCfg.BaseBranch)
	}
	if !strings.Contains(seenBehavior, "A works") || !strings.Contains(seenBehavior, "criterion 1: FAIL") {
		t.Errorf("fix prompt context = %q, want behavior plus verification result", seenBehavior)
	}
	if feature.ExpectedBehavior != "A works" {
		t.Errorf("ExpectedBehavior not restored after fix: %q", feature.ExpectedBehavior)
	}
}

func TestMissionFixFunc_NoHandoffUsesBaseBranch(t *testing.T) {
	m := mission.New("test", mission.Config{RepoDir: "/repo", BaseBranch: "main"})
	var seenCfg mission.Config
	worker := func(_ context.Context, _ *mission.Feature, _ string, cfg mission.Config) (*mission.Handoff, error) {
		seenCfg = cfg
		return &mission.Handoff{Summary: "fixed"}, nil
	}
	if _, err := missionFixFunc(m, worker)(context.Background(), &mission.Feature{ID: "feat-1"}, "broken", nil); err != nil {
		t.Fatalf("fix func: %v", err)
	}
	if seenCfg.BaseBranch != "main" {
		t.Errorf("fix BaseBranch = %q, want the configured base branch", seenCfg.BaseBranch)
	}
}
