package cmd

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/GrayCodeAI/reflect/internal/hosttools"
)

// hostToolsSubcommand lists registered host tools.
type hostToolsSubcommand struct{}

func (c *hostToolsSubcommand) Name() string      { return "host-tools" }
func (c *hostToolsSubcommand) Aliases() []string { return []string{"hosttools"} }
func (c *hostToolsSubcommand) Description() string {
	return "list host-provided tools (~/.reflect/host_tools)"
}
func (c *hostToolsSubcommand) Usage() string { return "/host-tools" }

func (c *hostToolsSubcommand) Handle(m *chatModel, args []string, text string) (tea.Model, tea.Cmd) {
	manifests, err := hosttools.Load()
	if err != nil {
		m.messages = append(m.messages, displayMsg{role: "error", content: err.Error()})
		return m, nil
	}
	if len(manifests) == 0 {
		m.messages = append(m.messages, displayMsg{role: "system", content: "No host tools registered. Add JSON manifests to " + hosttools.Dir()})
		return m, nil
	}
	var b strings.Builder
	b.WriteString("Host tools:\n")
	for _, m := range manifests {
		b.WriteString(fmt.Sprintf("  - %s: %s\n", m.Name, m.Description))
	}
	m.messages = append(m.messages, displayMsg{role: "system", content: b.String()})
	return m, nil
}

func init() {
	subcommandRegistry.Register(&hostToolsSubcommand{})
}
