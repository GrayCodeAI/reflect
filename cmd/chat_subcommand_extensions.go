package cmd

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/GrayCodeAI/reflect/internal/extensions"
)

// extensionsSubcommand lists registered extensions.
type extensionsSubcommand struct{}

func (c *extensionsSubcommand) Name() string        { return "extensions" }
func (c *extensionsSubcommand) Aliases() []string   { return []string{"ext"} }
func (c *extensionsSubcommand) Description() string { return "list extensions (~/.reflect/extensions)" }
func (c *extensionsSubcommand) Usage() string       { return "/extensions" }

func (c *extensionsSubcommand) Handle(m *chatModel, args []string, text string) (tea.Model, tea.Cmd) {
	exts, err := extensions.Load()
	if err != nil {
		m.messages = append(m.messages, displayMsg{role: "error", content: err.Error()})
		return m, nil
	}
	if len(exts) == 0 {
		m.messages = append(m.messages, displayMsg{role: "system", content: "No extensions installed. Add directories under " + extensions.Dir()})
		return m, nil
	}
	var b strings.Builder
	b.WriteString("Extensions:\n")
	for _, e := range exts {
		b.WriteString(fmt.Sprintf("  - %s: %s\n", e.Name, e.Description))
		for _, c := range e.Commands {
			b.WriteString(fmt.Sprintf("      /%s — %s\n", c.Name, c.Description))
		}
	}
	m.messages = append(m.messages, displayMsg{role: "system", content: b.String()})
	return m, nil
}

func init() {
	subcommandRegistry.Register(&extensionsSubcommand{})
}
