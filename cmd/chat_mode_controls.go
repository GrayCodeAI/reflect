package cmd

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
	"github.com/GrayCodeAI/reflect/internal/engine"
	"github.com/charmbracelet/x/ansi"
)

type workModeChoice struct {
	mode              engine.WorkMode
	name, description string
}

var workModeChoices = []workModeChoice{
	{engine.WorkModePlan, "Plan", "Research the repository and prepare a plan"},
	{engine.WorkModeAct, "Build", "Implement the plan under the selected permission policy"},
	{engine.WorkModeReview, "Review", "Inspect code and report findings with file references"},
}

type WorkModePicker struct {
	open     bool
	selected int
}

func (p *WorkModePicker) Open(current engine.WorkMode) {
	p.open = true
	p.selected = 0
	for i, c := range workModeChoices {
		if c.mode == current {
			p.selected = i
		}
	}
}

func (p *WorkModePicker) Update(msg tea.KeyMsg) (engine.WorkMode, bool) {
	switch msg.String() {
	case "esc", "f2":
		p.open = false
	case "up":
		p.selected = (p.selected + len(workModeChoices) - 1) % len(workModeChoices)
	case "down":
		p.selected = (p.selected + 1) % len(workModeChoices)
	case "enter":
		p.open = false
		return workModeChoices[p.selected].mode, true
	}
	return "", false
}

func (p WorkModePicker) Render(width int) string {
	w := max(1, min(72, width-4))
	rows := []string{lipgloss.NewStyle().Foreground(textPrimary).Bold(true).Render("Work mode"), statusDimStyle.Render("↑/↓ choose · Enter select · Esc cancel"), ""}
	for i, c := range workModeChoices {
		prefix := "  "
		style := statusDimStyle
		if i == p.selected {
			prefix = "> "
			style = lipgloss.NewStyle().Foreground(reflectColor).Bold(true)
		}
		rows = append(rows, style.Render(prefix+c.name), statusDimStyle.Render(ansi.Wrap("  "+c.description, max(1, w-4), "")), "")
	}
	return paletteBoxStyle().Width(w).Render(strings.TrimRight(strings.Join(rows, "\n"), "\n"))
}

func (m *chatModel) openWorkModePicker() {
	if m.modePicker == nil {
		m.modePicker = &WorkModePicker{}
	}
	mode := engine.WorkModeAct
	if m.session != nil {
		mode = m.session.WorkMode()
	}
	m.modePicker.Open(mode)
	m.viewDirty = true
}

func (m *chatModel) changeWorkMode(mode engine.WorkMode) {
	if m.session == nil {
		return
	}
	if err := m.session.SetWorkMode(mode); err != nil {
		m.messages = append(m.messages, displayMsg{role: "error", content: err.Error()})
	} else {
		m.messages = append(m.messages, displayMsg{role: "system", content: workModeSwitchSummary(m.session, mode)})
		m.refreshInputPlaceholder()
	}
	m.viewDirty = true
}

func (m *chatModel) handleDashboardShortcut(msg tea.KeyMsg) bool {
	if m.configOpen || m.hasActiveInteractivePrompt() {
		return false
	}
	switch msg.String() {
	case "f2":
		m.openWorkModePicker()
	case "f3":
		if m.autonomyPicker == nil {
			m.autonomyPicker = NewAutonomyPicker(m.chatColumnWidth())
		}
		m.autonomyPicker.Open(effectivePermissionTier(m.session))
	case "shift+tab":
		if m.session != nil {
			m.changeWorkMode(engine.NextWorkMode(m.session.WorkMode()))
		}
	default:
		return false
	}
	m.viewDirty = true
	return true
}

// A blocking request owns the keyboard and its visible conversation pane.
// Closing a transient menu cannot grant or reject that request.
func (m *chatModel) focusInteractiveRequest() {
	if m.modePicker != nil {
		m.modePicker.open = false
	}
	if m.commandPalette != nil {
		m.commandPalette.Close()
	}
	if m.specPicker != nil {
		m.specPicker.Close()
	}
	if m.autonomyPicker != nil {
		m.autonomyPicker.Close()
	}
	if m.themePicker != nil {
		m.themePicker.Close()
	}
	m.historySearchOpen = false
	m.sessionPickerOpen = false
	m.hudOpen = false
	m.configOpen = false
	m.useConfigInput = false
	m.uiFocus = focusPrompt
	m.input.Focus()
	m.invalidateInputLayoutCache()
}

// Chat uses the original full terminal width.
func (m chatModel) chatColumnWidth() int {
	if m.width > 0 {
		return m.width
	}
	return 80
}
