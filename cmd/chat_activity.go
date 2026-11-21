package cmd

import (
	"strings"
	"time"
	"unicode"

	"github.com/GrayCodeAI/reflect/internal/engine"
	sessionfeature "github.com/GrayCodeAI/reflect/internal/features/session"
	"github.com/GrayCodeAI/reflect/internal/ui/icons"
)

// A saved call without an observed result is incomplete, not a live tool or
// an assumed success. This is a host display state, never a Relay wire value.
const toolStateIncomplete engine.ToolState = "incomplete"

func displayFromHydrated(message sessionfeature.HydratedDisplayMessage) displayMsg {
	msg := displayMsg{role: message.Role, content: message.Content}
	if message.Call != nil {
		msg.toolID, msg.toolName = message.Call.ID, message.Call.Name
		msg.summary = engine.ToolActivitySummary(message.Call.Name, message.Call.Arguments)
		if msg.role == "tool_use" {
			msg.content = message.Call.Name
			msg.toolState = toolStateIncomplete
		}
	}
	if message.Result != nil {
		msg.toolID = message.Result.ToolUseID
		msg.toolState = engine.ToolStateCompleted
		if message.Result.IsError {
			msg.toolState = engine.ToolStateFailed
		}
	}
	return msg
}

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
		marker, style = icons.Check(), statusDimStyle
	case engine.ToolStateFailed, engine.ToolStateTimedOut:
		marker, style = "!", errorStyle
	case engine.ToolStateCancelled:
		marker, style = "×", statusDimStyle
	case toolStateIncomplete:
		marker, style = "?", warnStyle
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
