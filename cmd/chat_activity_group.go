package cmd

import (
	"fmt"
	"strings"

	"github.com/GrayCodeAI/reflect/internal/engine"
	"github.com/GrayCodeAI/reflect/internal/ui/icons"
)

// displayBlock is a projection over the transcript, never a replacement for
// it. Rendering and keyboard targeting use the same boundaries and indices.
// Prompts, warnings, answers, and user messages always keep their own blocks.
type displayBlock struct {
	start, end int
	activity   bool
}

func isToolMessage(msg displayMsg) bool {
	return msg.role == "tool_use" || msg.role == "tool_result"
}

func nextDisplayBlock(messages []displayMsg, start, end int) displayBlock {
	block := displayBlock{start: start, end: start + 1, activity: isToolMessage(messages[start])}
	if block.activity {
		for block.end < end && isToolMessage(messages[block.end]) {
			block.end++
		}
	}
	return block
}

type activityCall struct {
	name  string
	state engine.ToolState
}

// activityCalls joins results by call ID, keeping launch order even when calls
// finish out of order. ID-less legacy messages pair with the latest pending
// call of the same name. They never collapse distinct identified calls.
func activityCalls(messages []displayMsg, block displayBlock) []activityCall {
	var calls []activityCall
	byID := make(map[string]int)
	pending := make([]bool, 0)
	for _, msg := range messages[block.start:block.end] {
		index, found := byID[msg.toolID]
		found = found && msg.toolID != ""
		if !found && msg.role == "tool_result" && msg.toolID == "" {
			for i := len(calls) - 1; i >= 0; i-- {
				if pending[i] && (msg.toolName == "" || calls[i].name == msg.toolName) {
					index, found = i, true
					break
				}
			}
		}
		if !found {
			name := msg.toolName
			if name == "" && msg.role == "tool_use" {
				name = msg.content
			}
			index = len(calls)
			calls = append(calls, activityCall{name: name, state: msg.toolState})
			pending = append(pending, msg.role == "tool_use")
			if msg.toolID != "" {
				byID[msg.toolID] = index
			}
		}
		if msg.role == "tool_result" {
			calls[index].state = msg.toolState
			if calls[index].state == "" {
				calls[index].state = engine.ToolStateCompleted
			}
			pending[index] = false
		}
	}
	return calls
}

func activityCategory(name string) int {
	switch strings.ToLower(name) {
	case "read", "read_file":
		return 0
	case "grep", "glob", "ls", "code_search", "codesearch", "search":
		return 1
	case "bash", "powershell", "shell":
		return 2
	case "write", "edit", "multiedit", "notebookedit", "structurededit", "write_file", "edit_file", "apply_patch":
		return 3
	default:
		return 4
	}
}

func renderActivityGroup(messages []displayMsg, block displayBlock, width int, expanded map[int]bool) string {
	calls := activityCalls(messages, block)
	var counts [5]int
	var running, failed, cancelled, incomplete int
	for _, call := range calls {
		counts[activityCategory(call.name)]++
		switch call.state {
		case engine.ToolStateCompleted:
		case engine.ToolStateFailed, engine.ToolStateTimedOut:
			failed++
		case engine.ToolStateCancelled:
			cancelled++
		case toolStateIncomplete:
			incomplete++
		default:
			running++
		}
	}
	marker, style := icons.Check(), statusDimStyle
	var parts []string
	if failed > 0 {
		marker, style = "!", errorStyle
		parts = append(parts, fmt.Sprintf("%d failed", failed))
	}
	if cancelled > 0 {
		if failed == 0 {
			marker = "×"
		}
		parts = append(parts, fmt.Sprintf("%d cancelled", cancelled))
	}
	if running > 0 {
		if failed == 0 {
			marker, style = "·", toolStyle
		}
		parts = append(parts, fmt.Sprintf("%d running", running))
	}
	if incomplete > 0 {
		if failed == 0 && running == 0 {
			marker, style = "?", warnStyle
		}
		parts = append(parts, fmt.Sprintf("%d incomplete", incomplete))
	}
	labels := [5][2]string{{"Read %d file", "Read %d files"}, {"%d search", "%d searches"}, {"Ran %d command", "Ran %d commands"}, {"%d edit", "%d edits"}, {"%d tool call", "%d tool calls"}}
	for category, count := range counts {
		if count > 0 {
			plural := 0
			if count != 1 {
				plural = 1
			}
			parts = append(parts, fmt.Sprintf(labels[category][plural], count))
		}
	}
	hint := "Tab, Enter: details"
	if expanded[block.start] {
		hint = "Enter: collapse"
	}
	heading := marker + " " + strings.Join(parts, " · ") + " · " + hint
	var b strings.Builder
	b.WriteString(style.Render(clipFooterLine(heading, max(1, width))))
	b.WriteByte('\n')
	if expanded[block.start] {
		// Keep individual results collapsed until the whole group is opened.
		// Opening reveals the existing bounded detail view, including commands.
		details := make(map[int]bool, block.end-block.start)
		for i := block.start; i < block.end; i++ {
			details[i] = true
		}
		for i := block.start; i < block.end; i++ {
			b.WriteString(renderDisplayMessage(messages[i], i, messages, width, details))
		}
	} else if failed > 0 {
		// A short diagnostic remains visible without opening routine output.
		for i := block.end - 1; i >= block.start; i-- {
			msg := messages[i]
			if msg.role == "tool_result" && (msg.toolState == engine.ToolStateFailed || msg.toolState == engine.ToolStateTimedOut) {
				lines := strings.Split(strings.TrimSpace(sanitizeDisplay(msg.content)), "\n")
				if tail := singleLineActivity(lines[len(lines)-1]); tail != "" {
					b.WriteString(errorStyle.Render(clipFooterLine("  "+tail, max(1, width))))
					b.WriteByte('\n')
				}
				break
			}
		}
	}
	b.WriteByte('\n')
	return b.String()
}

func renderDisplayBlock(messages []displayMsg, block displayBlock, width int, expanded map[int]bool) string {
	if block.activity {
		return renderActivityGroup(messages, block, width, expanded)
	}
	return renderDisplayMessage(messages[block.start], block.start, messages, width, expanded)
}
