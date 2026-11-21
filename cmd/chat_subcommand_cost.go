package cmd

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/GrayCodeAI/reflect/internal/usage"
)

// costSubcommand implements the /cost slash command. It prints the
// session's cost summary (token usage, USD cost) plus a per-helper
// breakdown when attributed helper usage exists in the ledger.
type costSubcommand struct{}

func (c *costSubcommand) Name() string        { return "cost" }
func (c *costSubcommand) Aliases() []string   { return nil }
func (c *costSubcommand) Description() string { return "print session cost and token usage summary" }
func (c *costSubcommand) Usage() string       { return "" }
func (c *costSubcommand) Handle(m *chatModel, args []string, text string) (tea.Model, tea.Cmd) {
	content := m.session.CostValue().Summary()
	if helpers := helperSpendSummary(); helpers != "" {
		content += "\n" + helpers
	}
	m.messages = append(m.messages, displayMsg{role: "system", content: content})
	return m, nil
}

// helperSpendSummary renders a one-line per-helper (subagent / mission
// worker) spend breakdown from the usage ledger over the last day, or ""
// when no attributed usage exists in that window. The ledger is global, so
// the window keeps the line relevant to the session at hand; attribution
// is stamped where helper sessions are constructed.
func helperSpendSummary() string {
	records, err := usage.Read()
	if err != nil {
		return ""
	}
	since := time.Now().Add(-24 * time.Hour).UnixMilli()
	sum := usage.Summarize(records, since)
	if len(sum.ByHelper) == 0 {
		return ""
	}
	parts := make([]string, 0, len(sum.ByHelper))
	for _, h := range sum.ByHelper {
		parts = append(parts, fmt.Sprintf("%s %d gens $%.4f", h.Helper, h.Generations, h.TotalCostUSD))
	}
	return "Helpers (ledger, last 24h): " + strings.Join(parts, " | ")
}

func init() {
	subcommandRegistry.Register(&costSubcommand{})
}
