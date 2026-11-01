package cmd

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	contracts "github.com/GrayCodeAI/reflect/internal/contracts/policy"
	"github.com/GrayCodeAI/reflect/internal/engine"
	"github.com/GrayCodeAI/reflect/internal/engine/safety"
)

func singlePaneTestModel(width, height int) *chatModel {
	m := newTestChatModel()
	m.width, m.height = width, height
	m.sessionID = ""
	m.statusLeftKey = "/project"
	m.statusLeftVal = "~/Desktop/GrayCodeAI/reflect"
	m.statusLeftBranch = "feat/cli-dashboard"
	m.statusLeftAt = time.Now()
	m.input.ShowLineNumbers = false
	m.input.SetHeight(1)
	m.input.Prompt = "> "
	m.input.Placeholder = "Describe a task or ask a question"
	m.welcomeCache = "Reflect v0.0.1"
	*m = m.withSyncedLayout()
	m.primeInitialViewportContent()
	return m
}

func TestChatShortcutsKeepModePermissionsAndSpecIndependent(t *testing.T) {
	m := singlePaneTestModel(160, 40)
	before := m.session.PermSvc().RuntimeState().Autonomy
	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyF2})
	*m = next.(chatModel)
	if m.modePicker == nil || !m.modePicker.open {
		t.Fatal("F2 did not open work modes")
	}
	m.modePicker.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	next, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	*m = next.(chatModel)
	if m.session.WorkMode() != engine.WorkModePlan || m.session.PermSvc().RuntimeState().Autonomy != before {
		t.Fatal("mode picker changed permission policy")
	}
	if currentSpecStage(m.session) != safety.SpecStageNone {
		t.Fatal("mode picker started a spec workflow")
	}
	next, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	*m = next.(chatModel)
	if m.session.WorkMode() != engine.WorkModeAct || currentSpecStage(m.session) != safety.SpecStageNone || m.session.PermSvc().RuntimeState().Autonomy != before {
		t.Fatal("Shift+Tab must switch Plan to Build without changing permissions or spec")
	}
	m.handleCommand("/spec")
	if m.specPicker == nil || !m.specPicker.IsOpen() {
		t.Fatal("/spec must open the advanced spec menu")
	}
	if !strings.Contains(stripANSI(m.View().Content), "Start") {
		t.Fatal("spec menu rendered outside the visible conversation pane")
	}
	m.specPicker.Close()
	next, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyF3})
	*m = next.(chatModel)
	if m.autonomyPicker == nil || !m.autonomyPicker.IsOpen() {
		t.Fatal("F3 did not open permissions")
	}
	m.autonomyPicker.Close()
}

func TestChatApprovalPreemptsMenusWithoutMakingDecision(t *testing.T) {
	m := singlePaneTestModel(160, 40)
	m.openWorkModePicker()
	req := safety.PermissionRequest{PermissionRequest: contracts.PermissionRequest{ToolName: "Bash", Summary: "git status"}, Response: make(chan bool, 1)}
	next, _ := m.Update(permissionAskMsg{req: req})
	*m = next.(chatModel)
	if m.modePicker.open || m.permReq == nil {
		t.Fatal("a menu hid the active approval")
	}
	next, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyF2})
	*m = next.(chatModel)
	if m.modePicker.open || len(req.Response) != 0 {
		t.Fatal("shortcut stole the approval keyboard or made a decision")
	}
	if !strings.Contains(stripANSI(m.View().Content), "Permission request") {
		t.Fatal("approval is not visible")
	}
	next, _ = m.Update(tea.KeyPressMsg{Code: 'n', Text: "n"})
	*m = next.(chatModel)
	select {
	case allowed := <-req.Response:
		if allowed {
			t.Fatal("deny became allow")
		}
	default:
		t.Fatal("denial did not reach the engine")
	}
}
