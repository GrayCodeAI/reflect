package cmd

import (
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/GrayCodeAI/reflect/internal/engine"
	"github.com/GrayCodeAI/reflect/internal/engine/safety"
	chatfeature "github.com/GrayCodeAI/reflect/internal/features/chat"
)

type planSubcommand struct{}

func (*planSubcommand) Name() string      { return "plan" }
func (*planSubcommand) Aliases() []string { return nil }
func (*planSubcommand) Description() string {
	return "prepare one implementation plan, or show the saved plan"
}
func (*planSubcommand) Usage() string { return "/plan [task] | /plan show" }
func (*planSubcommand) Handle(m *chatModel, args []string, _ string) (tea.Model, tea.Cmd) {
	if m.session == nil {
		m.messages = append(m.messages, displayMsg{role: "error", content: "No active session."})
		return m, nil
	}
	if len(args) == 1 && strings.EqualFold(args[0], "show") {
		if m.sessionID == "" {
			m.messages = append(m.messages, displayMsg{role: "system", content: "No saved plan for this conversation yet."})
			return m, nil
		}
		path := chatfeature.PlanDocumentPath(m.session.WorkingDir(), m.sessionID)
		content, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			m.messages = append(m.messages, displayMsg{role: "system", content: "No saved plan yet. Use /plan <task> to prepare one."})
		} else if err != nil {
			m.messages = append(m.messages, displayMsg{role: "error", content: fmt.Sprintf("Read plan: %v", err)})
		} else {
			m.messages = append(m.messages, displayMsg{role: "assistant", content: string(content)})
		}
		m.viewDirty = true
		return m, nil
	}
	if currentSpecStage(m.session) != safety.SpecStageNone {
		m.messages = append(m.messages, displayMsg{role: "system", content: "The advanced spec workflow is active. Use /spec to manage it, or /spec reset to return to everyday planning."})
		return m, nil
	}
	m.changeWorkMode(engine.WorkModePlan)
	if len(args) == 0 {
		return m, nil
	}
	return m.startPromptCommand("/plan "+strings.Join(args, " "), strings.Join(args, " "))
}

func init() { subcommandRegistry.Register(&planSubcommand{}) }
