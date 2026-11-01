package cmd

import (
	"os"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/GrayCodeAI/reflect/internal/engine"
	"github.com/GrayCodeAI/reflect/internal/engine/safety"
	chatfeature "github.com/GrayCodeAI/reflect/internal/features/chat"
	"github.com/GrayCodeAI/reflect/internal/storage"
)

func TestEverydayWorkflowCycleKeepsPermissionsAndSpecIndependent(t *testing.T) {
	m := singlePaneTestModel(160, 40)
	before := m.session.PermSvc().RuntimeState().Autonomy
	beforeMessages := len(m.session.RawMessages())
	for _, want := range []engine.WorkMode{engine.WorkModeReview, engine.WorkModePlan, engine.WorkModeAct} {
		next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
		*m = next.(chatModel)
		if m.session.WorkMode() != want {
			t.Fatalf("mode = %s, want %s", m.session.WorkMode(), want)
		}
		if currentSpecStage(m.session) != safety.SpecStageNone || m.session.PermSvc().RuntimeState().Autonomy != before {
			t.Fatal("mode cycle changed spec stage or permission tier")
		}
		if m.waiting || len(m.session.RawMessages()) != beforeMessages {
			t.Fatal("mode cycle started model work")
		}
	}
	if !strings.Contains(stripANSI(m.View().Content), "Build") {
		t.Fatal("Build label missing")
	}
	m.session.PermSvc().SetSpecStage(safety.SpecStageProposal)
	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	*m = next.(chatModel)
	if currentSpecStage(m.session) != safety.SpecStageProposal {
		t.Fatal("mode cycle reset an active advanced spec")
	}
}

func TestEverydayPlanSavedAndReopenedThroughRegisteredCommand(t *testing.T) {
	storage.SetTestDirs(t, t.TempDir())
	m := newTestChatModel()
	project := t.TempDir()
	m.session.Tools().SetWorkingDir(project)
	t.Cleanup(func() {
		if m.wal != nil {
			_ = m.wal.Close()
		}
		if m.sleepCancel != nil {
			m.sleepCancel()
		}
	})
	m.handleCommand("/plan add pagination")
	if m.session.WorkMode() != engine.WorkModePlan || !m.waiting || m.streamUsesSpec || currentSpecStage(m.session) != safety.SpecStageNone {
		t.Fatal("/plan did not start everyday planning")
	}
	plan := "# Pagination\n\n1. Read the endpoint.\n2. Add page cursors.\n3. Verify tests."
	m.partial.WriteString(plan)
	next, _ := m.handleStreamDone()
	*m = next.(chatModel)
	path := chatfeature.PlanDocumentPath(project, m.sessionID)
	got, err := os.ReadFile(path)
	if err != nil || string(got) != plan+"\n" {
		t.Fatalf("saved plan = %q, %v", got, err)
	}
	m.messages = nil
	m.handleCommand("/plan show")
	if len(m.messages) != 1 || strings.TrimSpace(m.messages[0].content) != plan {
		t.Fatal("/plan show did not reopen the saved plan")
	}
	entries, err := os.ReadDir(project)
	if err != nil || len(entries) != 0 {
		t.Fatalf("ordinary planning created project artifacts: %v, %v", entries, err)
	}
	// A cancelled follow-up must preserve the last complete plan.
	m.startStream()
	m.partial.WriteString("incomplete replacement")
	m.streamCancelled = true
	next, _ = m.handleStreamDone()
	*m = next.(chatModel)
	got, err = os.ReadFile(path)
	if err != nil || string(got) != plan+"\n" {
		t.Fatalf("cancelled turn replaced plan: %q, %v", got, err)
	}
}

func TestEverydayPlanDoesNotReplaceAdvancedOrBuildOutput(t *testing.T) {
	storage.SetTestDirs(t, t.TempDir())
	m := newTestChatModel()
	m.session.Tools().SetWorkingDir(t.TempDir())
	t.Cleanup(func() {
		if m.wal != nil {
			_ = m.wal.Close()
		}
	})
	for _, advanced := range []bool{false, true} {
		if advanced {
			_ = m.session.SetWorkMode(engine.WorkModePlan)
			m.session.PermSvc().SetSpecStage(safety.SpecStageProposal)
		}
		m.startStream()
		m.partial.WriteString("This is not an everyday plan.")
		next, _ := m.handleStreamDone()
		*m = next.(chatModel)
		if _, err := os.Stat(chatfeature.PlanDocumentPath(m.session.WorkingDir(), m.sessionID)); !os.IsNotExist(err) {
			t.Fatalf("unexpected plan, advanced=%v: %v", advanced, err)
		}
	}
	m.handleCommand("/plan add pagination")
	if m.waiting || currentSpecStage(m.session) != safety.SpecStageProposal {
		t.Fatal("/plan silently replaced an active advanced workflow")
	}
}
