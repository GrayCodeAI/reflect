package cmd

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/GrayCodeAI/reflect/internal/session"
)

// workspacesSubcommand lists saved sessions (Mirror-style workspace roster).
type workspacesSubcommand struct{}

func (c *workspacesSubcommand) Name() string        { return "workspaces" }
func (c *workspacesSubcommand) Aliases() []string   { return []string{"worktree-list"} }
func (c *workspacesSubcommand) Description() string { return "list saved sessions/workspaces" }
func (c *workspacesSubcommand) Usage() string       { return "/workspaces" }

func (c *workspacesSubcommand) Handle(m *chatModel, args []string, text string) (tea.Model, tea.Cmd) {
	entries, err := session.List()
	if err != nil {
		m.messages = append(m.messages, displayMsg{role: "error", content: err.Error()})
		return m, nil
	}
	if len(entries) == 0 {
		m.messages = append(m.messages, displayMsg{role: "system", content: "No saved sessions."})
		return m, nil
	}
	var b strings.Builder
	b.WriteString("Sessions:\n")
	for i, e := range entries {
		b.WriteString(fmt.Sprintf("  %d. %s (id=%s)\n", i+1, e.Preview, e.ID))
	}
	m.messages = append(m.messages, displayMsg{role: "system", content: b.String()})
	return m, nil
}

func init() {
	subcommandRegistry.Register(&workspacesSubcommand{})
}
