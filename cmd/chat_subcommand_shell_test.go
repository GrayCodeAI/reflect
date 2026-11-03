package cmd

import (
	"strings"
	"testing"

	"github.com/GrayCodeAI/reflect/internal/engine/safety"
)

func TestRunSubcommand_UsesBashToolForSafeCommand(t *testing.T) {
	m := newTestChatModel()
	m.session.PermSvc().SetAutonomy(safety.AutonomyYOLO)

	result, _ := (&runSubcommand{}).Handle(m, []string{"printf", "hello"}, "/run printf hello")
	cm := requireChatModel(t, result)

	if len(cm.messages) == 0 {
		t.Fatal("expected display message")
	}
	last := cm.messages[len(cm.messages)-1]
	if last.role != "system" {
		t.Fatalf("last role = %q, want system", last.role)
	}
	if !strings.Contains(last.content, "$ printf hello") || !strings.Contains(last.content, "hello") {
		t.Fatalf("display output = %q, want command and hello", last.content)
	}
	msgs := cm.session.RawMessages()
	if len(msgs) != 1 || !strings.Contains(msgs[0].Content, "hello") {
		t.Fatalf("session messages = %#v, want command output context", msgs)
	}
}

func TestRunSubcommand_ReportsFailedPipelineWithDiagnostics(t *testing.T) {
	m := newTestChatModel()
	m.session.PermSvc().SetAutonomy(safety.AutonomyYOLO)
	result, _ := (&runSubcommand{}).Handle(m, []string{"printf 'build diagnostic\\n'; false | tail -5"}, "/run printf 'build diagnostic\\n'; false | tail -5")
	cm := requireChatModel(t, result)
	last := cm.messages[len(cm.messages)-1]
	if last.role != "error" || !strings.Contains(last.content, "build diagnostic") || !strings.Contains(last.content, "exit code:") {
		t.Fatalf("failed check appeared successful or lost its diagnostic: role=%s output=%q", last.role, last.content)
	}
}

func TestRunSubcommand_RespectsSpecStageGate(t *testing.T) {
	m := newTestChatModel()
	m.session.PermSvc().SetSpecStage(safety.SpecStageSpecify)

	result, _ := (&runSubcommand{}).Handle(m, []string{"printf", "hello"}, "/run printf hello")
	cm := requireChatModel(t, result)

	if len(cm.messages) == 0 {
		t.Fatal("expected display message")
	}
	last := cm.messages[len(cm.messages)-1]
	if last.role != "error" {
		t.Fatalf("last role = %q, want error", last.role)
	}
	if !strings.Contains(last.content, "Spec stage active") {
		t.Fatalf("display output = %q, want spec-gate denial", last.content)
	}
	if got := len(cm.session.RawMessages()); got != 0 {
		t.Fatalf("session messages = %d, want 0 after denied command", got)
	}
}

func TestRunSubcommand_BashToolBlocksEnvDumpEvenWhenBypassed(t *testing.T) {
	m := newTestChatModel()
	m.session.PermSvc().SetAutonomy(safety.AutonomyYOLO)

	result, _ := (&runSubcommand{}).Handle(m, []string{"env"}, "/run env")
	cm := requireChatModel(t, result)

	if len(cm.messages) == 0 {
		t.Fatal("expected display message")
	}
	last := cm.messages[len(cm.messages)-1]
	if last.role != "error" {
		t.Fatalf("last role = %q, want error", last.role)
	}
	if !strings.Contains(last.content, "blocked: dumping environment variables") {
		t.Fatalf("display output = %q, want BashTool env dump block", last.content)
	}
}

func TestRunSubcommand_AllowsNamedGoDiagnostics(t *testing.T) {
	m := newTestChatModel()
	m.session.PermSvc().SetAutonomy(safety.AutonomyYOLO)

	result, _ := (&runSubcommand{}).Handle(m, []string{"go", "env", "GOTOOLCHAIN"}, "/run go env GOTOOLCHAIN")
	cm := requireChatModel(t, result)
	if len(cm.messages) == 0 {
		t.Fatal("expected display message")
	}
	last := cm.messages[len(cm.messages)-1]
	if last.role == "error" || strings.Contains(last.content, "blocked:") || !strings.Contains(last.content, "$ go env GOTOOLCHAIN") {
		t.Fatalf("safe Go diagnostic failed in CLI: role=%s output=%q", last.role, last.content)
	}
}

func TestTestSubcommand_DetectsNonzeroExit(t *testing.T) {
	m := newTestChatModel()
	m.session.PermSvc().SetAutonomy(safety.AutonomyYOLO)

	result, _ := (&testSubcommand{}).Handle(m, []string{"false"}, "/test false")
	cm := requireChatModel(t, result)

	if len(cm.messages) == 0 {
		t.Fatal("expected display message")
	}
	last := cm.messages[len(cm.messages)-1]
	if last.role != "system" {
		t.Fatalf("last role = %q, want system", last.role)
	}
	if !strings.Contains(last.content, "Tests failed:") || !strings.Contains(last.content, "exit code") {
		t.Fatalf("display output = %q, want failed test output", last.content)
	}
	msgs := cm.session.RawMessages()
	if len(msgs) != 1 || !strings.Contains(msgs[0].Content, "Please fix these test failures") {
		t.Fatalf("session messages = %#v, want failure context", msgs)
	}
}

func TestLintSubcommandQueuesActualFailureOnly(t *testing.T) {
	for _, tc := range []struct {
		command string
		failed  bool
	}{
		{command: "printf 'compiler diagnostic'; false | tail -5", failed: true},
		{command: "printf '0 issues.'"},
	} {
		t.Run(tc.command, func(t *testing.T) {
			m := newTestChatModel()
			m.session.PermSvc().SetAutonomy(safety.AutonomyYOLO)
			result, _ := (&lintSubcommand{}).Handle(m, []string{tc.command}, "/lint "+tc.command)
			cm := requireChatModel(t, result)
			messages := cm.session.RawMessages()
			if tc.failed {
				if len(messages) != 1 || !strings.Contains(messages[0].Content, "Please fix these lint issues") || !strings.Contains(messages[0].Content, "compiler diagnostic") {
					t.Fatalf("lost repair diagnostics: %#v", messages)
				}
			} else if len(messages) != 0 || !strings.Contains(cm.messages[len(cm.messages)-1].content, "No lint issues") {
				t.Fatalf("successful linter triggered unnecessary repairs: %#v", messages)
			}
		})
	}
}

func TestTestSubcommandDoesNotTreatDenialAsTestFailure(t *testing.T) {
	m := newTestChatModel()
	m.session.PermSvc().SetAutonomy(safety.AutonomyYOLO)
	result, _ := (&testSubcommand{}).Handle(m, []string{"env"}, "/test env")
	cm := requireChatModel(t, result)
	if len(cm.session.RawMessages()) != 0 || cm.messages[len(cm.messages)-1].role != "error" {
		t.Fatal("blocked command was queued as a code repair")
	}
}
