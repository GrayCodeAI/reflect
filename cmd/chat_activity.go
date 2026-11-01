package cmd

import (
	"strings"
	"time"
	"unicode"

	"github.com/GrayCodeAI/reflect/internal/engine"
)

func singleLineActivity(value string) string {
	value = strings.NewReplacer("\r", `\r`, "\n", `\n`, "\t", " ").Replace(sanitizeDisplay(value))
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || unicode.In(r, unicode.Cf) {
			return -1
		}
		return r
	}, value)
}

func renderToolActivity(msg displayMsg, width int) string {
	marker := "·"
	style := toolStyle
	switch msg.toolState {
	case engine.ToolStateCompleted:
		marker, style = "✓", statusDimStyle
	case engine.ToolStateFailed, engine.ToolStateTimedOut:
		marker, style = "!", errorStyle
	case engine.ToolStateCancelled:
		marker, style = "×", statusDimStyle
	}
	heading := style.Render(marker + " " + singleLineActivity(msg.content))
	if detail := singleLineActivity(msg.summary); strings.TrimSpace(detail) != "" {
		heading += "  " + toolDimStyle.Render(detail)
	}
	return clipFooterLine(heading, max(1, width))
}

// Retain the request and the existing decision log, but release its large card
// once no interactive request owns the keyboard.
func (m *chatModel) retirePromptCards() {
	changed := false
	for i := range m.messages {
		entry := &m.messages[i]
		switch entry.role {
		case "permission", "approval", "credential", "question":
			if !entry.resolved {
				entry.resolved = true
				entry.timeoutAt = time.Time{}
				changed = true
			}
		}
	}
	if changed {
		m.invalidateViewportCache()
		m.viewDirty = true
		m.updateViewportContent()
	}
}
