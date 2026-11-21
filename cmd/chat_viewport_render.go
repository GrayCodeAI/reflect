package cmd

import (
	"fmt"
	"strings"

	"github.com/GrayCodeAI/reflect/internal/engine"
	"github.com/GrayCodeAI/reflect/internal/ui/icons"
)

// Viewport render cache — avoids re-wrapping and re-rendering markdown for the
// entire scrollback on every 50ms stream tick. Stable prefix is cached; only
// new/changed messages and the live tail (partial + spinner) are rebuilt.

func (m *chatModel) invalidateViewportCache() {
	m.vpStableContent = ""
	m.vpRenderedMsgs = 0
	m.vpRenderWidth = 0
	m.vpLastMsgLen = 0
	m.vpTailStart = 0
	m.vpTailOffset = 0
	m.streamMDPrefixRaw = ""
	m.streamMDPrefixOut = ""
	m.streamMDWidth = 0
}

// Tool result collapse thresholds. Tool results longer than
// toolResultCollapseLines are collapsed to toolResultPreviewLines with a
// toggle indicator; the user presses Enter in scrollback focus to expand.
const (
	toolResultCollapseLines = 3 // collapse rendered output longer than this
	toolResultPreviewLines  = 2 // preview lines shown when collapsed
)

func renderDisplayMessage(msg displayMsg, i int, messages []displayMsg, viewWidth int, expanded map[int]bool) string {
	reflectC := ansiOrange
	rst := ansiReset
	bgDark := "\033[48;2;30;30;40m"

	// Sanitize untrusted content once, before any branch renders it. This is
	// the single choke point for the whole scrollback: tool results, model
	// output, thinking, and system messages all flow through here.
	msg.content = sanitizeDisplay(msg.content)
	if msg.resolved {
		label := strings.ToUpper(msg.role[:1]) + msg.role[1:]
		parts := strings.SplitN(msg.content, "\n", 3)
		detail := parts[0]
		if len(parts) > 1 && !strings.HasPrefix(parts[1], "Effects:") && !strings.HasSuffix(parts[0], "] "+parts[1]) {
			detail += " · " + parts[1]
		}
		return dimStyle.Render(clipFooterLine(label+": "+singleLineActivity(detail), max(1, viewWidth))) + "\n"
	}

	var b strings.Builder

	switch msg.role {
	case "user":
		if i > 0 {
			b.WriteByte('\n')
		}
		wrapped := wrapText(msg.content, viewWidth-1, 3)
		wrappedLines := strings.Split(wrapped, "\n")
		for li, wl := range wrappedLines {
			if li == 0 {
				b.WriteString(bgDark + reflectC + "█" + rst + bgDark + "  " + wl)
			} else {
				b.WriteString(bgDark + "   " + wl)
			}
			visW := 3 + visibleWidth(wl)
			if pad := viewWidth - visW; pad > 0 {
				b.WriteString(strings.Repeat(" ", pad))
			}
			b.WriteString(rst)
			if li < len(wrappedLines)-1 {
				b.WriteByte('\n')
			}
		}
	case "assistant":
		content := strings.TrimLeft(msg.content, "\n\r")
		if strings.TrimSpace(content) == "" {
			// The model called a tool without any preceding text — skip the
			// message entirely rather than leaving an orphan "◈" line.
			return ""
		}
		b.WriteString(reflectC + icons.Robot() + " " + rst + renderMarkdown(content, viewWidth-3))
	case "tool_use":
		b.WriteString(renderToolActivity(msg, viewWidth))
	case "tool_result":
		var inner strings.Builder
		if looksLikeGitDiff(msg.content) {
			rendered := renderGitDiffOutput(msg.content, viewWidth-6)
			inner.WriteString("    " + strings.ReplaceAll(rendered, "\n", "\n    "))
		} else if strings.Contains(msg.content, "diff ") && strings.Contains(msg.content, " lines") {
			parts := strings.SplitN(msg.content, "\ndiff ", 2)
			mainContent := parts[0]
			diffPart := ""
			if len(parts) > 1 {
				diffPart = "diff " + parts[1]
			}
			toolWrapped := wrapText(mainContent, viewWidth-6, 0)
			inner.WriteString(toolDimStyle.Render("    " + strings.ReplaceAll(toolWrapped, "\n", "\n    ")))
			if diffPart != "" {
				inner.WriteString("\n")
				diffStyled := renderDiffSummary(diffPart, viewWidth-6)
				inner.WriteString("    " + diffStyled)
			}
		} else if strings.Contains(msg.content, "Self-review found issues") {
			inner.WriteString(errorStyle.Render("    " + icons.CloseThick() + " " + msg.content))
		} else if strings.Contains(msg.content, "## Self-Reflection") {
			parts := strings.SplitN(msg.content, "## Self-Reflection", 2)
			mainContent := parts[0]
			reflectionPart := ""
			if len(parts) > 1 {
				reflectionPart = "## Self-Reflection" + parts[1]
			}
			toolWrapped := wrapText(mainContent, viewWidth-6, 0)
			inner.WriteString(toolDimStyle.Render("    " + strings.ReplaceAll(toolWrapped, "\n", "\n    ")))
			if reflectionPart != "" {
				inner.WriteString("\n")
				reflStyled := renderReflectionBox(reflectionPart, viewWidth-6)
				inner.WriteString("    " + reflStyled)
			}
		} else {
			display := formatToolResultDisplay(msg.content)
			toolWrapped := wrapText(display, viewWidth-6, 0)
			inner.WriteString(toolDimStyle.Render("    " + strings.ReplaceAll(toolWrapped, "\n", "\n    ")))
		}
		failed := msg.toolState == engine.ToolStateFailed || msg.toolState == engine.ToolStateTimedOut
		b.WriteString(collapseToolResult(inner.String(), i, expanded, failed))
	case "thinking":
		thinkWrapped := wrapText(msg.content, viewWidth-4, 3)
		b.WriteString(dimStyle.Render(icons.Brain() + " " + thinkWrapped))
	case "welcome":
		// rendered in fixed welcome pane, not scrollback
	case "system":
		sysWrapped := wrapText(msg.content, viewWidth-2, 0)
		b.WriteString(dimStyle.Render(sysWrapped))
	case "setup_complete":
		b.WriteString(renderSetupCompleteMessage(msg.content))
	case "permission":
		b.WriteString(renderPermissionBox(msg.content, viewWidth, msg.timeoutAt))
	case "approval":
		b.WriteString(renderApprovalBox(msg.content, viewWidth, msg.timeoutAt))
	case "credential":
		b.WriteString(renderCredentialBox(msg.content, viewWidth, msg.timeoutAt))
	case "question":
		b.WriteString(renderQuestionBox(msg.content, viewWidth, msg.timeoutAt))
	case "usage":
		b.WriteString(dimStyle.Render("  " + msg.content))
	case "warning":
		warnWrapped := wrapText(msg.content, viewWidth-8, 0)
		b.WriteString(warnStyle.Render(warnWrapped))
	case "error":
		errWrapped := wrapText(msg.content, viewWidth-8, 7)
		b.WriteString(errorStyle.Render("Error: " + errWrapped))
	}

	switch msg.role {
	case "user":
		if i+1 < len(messages) && messages[i+1].role == "tool_use" {
			// A tool call the model runs immediately in response to this
			// command reads as a continuation of it, not a new block — no
			// blank-line gap.
			b.WriteByte('\n')
		} else {
			b.WriteString("\n\n")
		}
	case "tool_use":
		if i+1 < len(messages) && (messages[i+1].role == "tool_result" || messages[i+1].role == "tool_use") {
			b.WriteByte('\n')
		} else {
			b.WriteString("\n\n")
		}
	case "tool_result":
		if i+1 < len(messages) && messages[i+1].role == "tool_use" {
			b.WriteByte('\n')
		} else {
			b.WriteString("\n\n")
		}
	case "usage":
		b.WriteByte('\n')
	case "welcome":
		// no trailing space
	default:
		if msg.role != "" {
			b.WriteString("\n\n")
		}
	}
	return b.String()
}

// collapseToolResult truncates long tool result output to a preview with a
// [+N lines] toggle indicator when the result is not expanded. Short results
// (≤ toolResultCollapseLines) pass through unchanged. The expanded map is keyed
// by message index.
func collapseToolResult(rendered string, msgIdx int, expanded map[int]bool, failed bool) string {
	if expanded[msgIdx] {
		return rendered
	}
	// Count visible lines. ANSI styling codes don't contain '\n', so a simple
	// split accurately reflects the rendered line count.
	lines := strings.Split(rendered, "\n")
	threshold := toolResultCollapseLines
	if failed {
		threshold = 2 * toolResultPreviewLines
	}
	if len(lines) <= threshold {
		return rendered
	}
	preview := strings.Join(lines[:toolResultPreviewLines], "\n")
	hidden := len(lines) - toolResultPreviewLines
	if failed {
		preview += "\n…\n" + strings.Join(lines[len(lines)-toolResultPreviewLines:], "\n")
		hidden -= toolResultPreviewLines
	}
	toggle := fmt.Sprintf("[+%d lines — Enter to expand]", hidden)
	return preview + "\n" + toolDimStyle.Render("    "+toggle)
}

func renderMessagesRange(messages []displayMsg, start, end int, viewWidth int, expanded map[int]bool) string {
	var b strings.Builder
	end = min(end, len(messages))
	for i := start; i < end; {
		block := nextDisplayBlock(messages, i, end)
		b.WriteString(renderDisplayBlock(messages, block, viewWidth, expanded))
		i = block.end
	}
	return b.String()
}

// renderStreamTail renders the live streaming partial. The stable prefix
// (every completed markdown block) is sanitized+rendered once and cached, so
// each 50ms tick only re-renders the small tail after the last block
// boundary instead of the whole accumulated response (which is O(n²) over a
// long stream). The cache self-validates via HasPrefix, so width changes,
// stream restarts, and partial resets all fall back to a clean rebuild.
func (m *chatModel) renderStreamTail(viewWidth int) string {
	raw := strings.TrimLeft(m.partial.String(), "\n\r")
	if raw == "" {
		if m.hasActiveInteractivePrompt() {
			return ""
		}
		return m.renderWaitingSpinnerLine() + "\n\n"
	}

	if m.streamMDWidth != viewWidth || !strings.HasPrefix(raw, m.streamMDPrefixRaw) {
		m.streamMDPrefixRaw = ""
		m.streamMDPrefixOut = ""
		m.streamMDWidth = viewWidth
	}
	// Fold any newly-completed blocks into the cached prefix, rendering ONLY
	// the new blocks (not the whole prefix) so cost stays linear over the
	// stream. The scan resumes from the last boundary, always outside a fence.
	if boundary := streamStableBoundary(raw, len(m.streamMDPrefixRaw)); boundary > len(m.streamMDPrefixRaw) {
		newBlocks := raw[len(m.streamMDPrefixRaw):boundary]
		rendered := renderMarkdown(sanitizeDisplay(sanitizeIdentity(newBlocks)), viewWidth-3)
		m.streamMDPrefixOut = appendRendered(m.streamMDPrefixOut, rendered, m.streamMDPrefixRaw)
		m.streamMDPrefixRaw = raw[:boundary]
	}

	out := m.streamMDPrefixOut
	if tail := raw[len(m.streamMDPrefixRaw):]; tail != "" {
		rendered := renderMarkdown(sanitizeDisplay(sanitizeIdentity(tail)), viewWidth-3)
		out = appendRendered(out, rendered, m.streamMDPrefixRaw)
	}
	return ansiOrange + icons.Robot() + " " + ansiReset + out + "\n\n"
}

// streamStableBoundary returns the offset just past the last blank-line block
// separator in raw that lies outside a fenced code block, mirroring
// renderMarkdown's fence rules (any ``` opens; a bare ``` closes). Splitting
// there and rendering the halves independently reproduces the full render.
// The scan starts at from, which callers guarantee is a previous boundary
// (always outside a fence), so the whole buffer is not re-scanned each tick.
func streamStableBoundary(raw string, from int) int {
	boundary := from
	inFence := false
	pos := from
	for {
		nl := strings.IndexByte(raw[pos:], '\n')
		if nl < 0 {
			break
		}
		trimmed := strings.TrimSpace(raw[pos : pos+nl])
		if !inFence && strings.HasPrefix(trimmed, "```") {
			inFence = true
		} else if inFence && trimmed == "```" {
			inFence = false
		}
		lineEnd := pos + nl + 1
		if !inFence && lineEnd < len(raw) && raw[lineEnd] == '\n' {
			boundary = lineEnd + 1
		}
		pos = lineEnd
	}
	return boundary
}

// appendRendered concatenates a freshly rendered segment onto already-rendered
// output, re-inserting the blank-line separator renderMarkdown trims. prevRaw
// is the raw text behind out, whose trailing blank lines are the separator.
func appendRendered(out, rendered, prevRaw string) string {
	if out == "" {
		return rendered
	}
	return out + trailingNewlines(prevRaw) + rendered
}

// trailingNewlines returns the newlines in s's trailing whitespace run —
// the blank-line separator renderMarkdown trims from a prefix render.
func trailingNewlines(s string) string {
	n := 0
	for i := len(s) - 1; i >= 0; i-- {
		switch s[i] {
		case '\n':
			n++
		case ' ', '\t', '\r':
			// blank-line content; keep scanning
		default:
			return strings.Repeat("\n", n)
		}
	}
	return strings.Repeat("\n", n)
}

// toolResultIndexAtViewportCenter returns the first transcript index of the
// visible activity group nearest the viewport center. Rendering and targeting share the
// same projection, including expanded groups and the welcome pane.
func (m *chatModel) toolResultIndexAtViewportCenter(viewWidth int) int {
	block, _, ok := m.displayBlockAtViewportCenter(viewWidth, true)
	if ok && block.activity {
		return block.start
	}
	return -1
}

func (m *chatModel) displayBlockAtViewportCenter(viewWidth int, activityOnly bool) (displayBlock, int, bool) {
	if m.viewport.Height() <= 0 {
		return displayBlock{}, 0, false
	}
	if m.vpRenderWidth > 0 {
		viewWidth = m.vpRenderWidth
	} else if m.viewport.Width() > 0 {
		viewWidth = m.viewport.Width()
	}
	centerY := m.viewport.YOffset() + m.viewport.Height()/2
	cumulative := 0
	var nearest displayBlock
	nearestDistance, nearestY := int(^uint(0)>>1), 0
	if welcome := m.renderWelcomeScreen(viewWidth); welcome != "" {
		cumulative = strings.Count(welcome+"\n\n", "\n")
	}
	for i := 0; i < len(m.messages); {
		block := nextDisplayBlock(m.messages, i, len(m.messages))
		rendered := renderDisplayBlock(m.messages, block, viewWidth, m.toolResultExpanded)
		lineCount := strings.Count(rendered, "\n")
		if !activityOnly && centerY >= cumulative && centerY < cumulative+lineCount {
			return block, centerY - cumulative, true
		}
		if activityOnly && block.activity && cumulative+lineCount > m.viewport.YOffset() && cumulative < m.viewport.YOffset()+m.viewport.Height() {
			distance := max(max(cumulative-centerY, centerY-(cumulative+lineCount-1)), 0)
			if distance < nearestDistance {
				nearest, nearestDistance, nearestY = block, distance, centerY-cumulative
			}
		}
		cumulative += lineCount
		i = block.end
	}
	return nearest, nearestY, activityOnly && nearestDistance != int(^uint(0)>>1)
}

// codeBlockAtViewportCenter finds the fenced code block in an assistant message
// whose rendered content contains the viewport's center line. Returns the raw
// code content (without fences) and true if found.
//
// Uses the same display blocks as rendering, so collapsed output cannot shift
// the target or make tool output look like an assistant code block.
func (m *chatModel) codeBlockAtViewportCenter() (string, bool) {
	viewWidth := m.width
	if viewWidth <= 0 {
		viewWidth = 80
	}

	block, relativeY, ok := m.displayBlockAtViewportCenter(viewWidth, false)
	if ok && !block.activity && m.messages[block.start].role == "assistant" {
		if m.vpRenderWidth > 0 {
			viewWidth = m.vpRenderWidth
		}
		code := findClosestCodeBlock(m.messages[block.start].content, relativeY, viewWidth)
		return code, code != ""
	}
	return "", false
}

// findClosestCodeBlock finds the code block in the content whose position is
// closest to the target line offset within the rendered message. It extracts
// code blocks with their raw positions, maps the target rendered line back to
// an approximate content line, and returns the nearest block.
func findClosestCodeBlock(content string, targetLine int, viewWidth int) string {
	// Extract code blocks with their line positions in the raw content.
	type rawBlock struct {
		lineIdx int // line index where opening ``` appears
		code    string
	}
	var rawBlocks []rawBlock

	lines := strings.Split(content, "\n")
	inFence := false
	fenceStart := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			if !inFence {
				inFence = true
				fenceStart = i
			} else {
				inFence = false
				// Extract code between fences.
				var code strings.Builder
				for j := fenceStart + 1; j < i; j++ {
					if code.Len() > 0 {
						code.WriteByte('\n')
					}
					code.WriteString(lines[j])
				}
				if code.Len() > 0 {
					rawBlocks = append(rawBlocks, rawBlock{lineIdx: fenceStart, code: code.String()})
				}
			}
		}
	}

	if len(rawBlocks) == 0 {
		return ""
	}

	// Map the target rendered line to an approximate content line.
	// Code blocks render with roughly: 1 label line + N code lines + padding.
	// A simple proportional mapping works well enough for "closest" selection.
	totalContentLines := len(lines)
	rendered := renderMarkdown(content, viewWidth-3)
	totalRenderedLines := strings.Count(rendered, "\n") + 1
	if totalRenderedLines == 0 {
		return rawBlocks[0].code
	}
	contentTarget := targetLine * totalContentLines / totalRenderedLines

	// Find the raw block closest to the mapped target.
	bestDist := int(^uint(0) >> 1)
	bestCode := rawBlocks[0].code
	for _, b := range rawBlocks {
		dist := abs(b.lineIdx - contentTarget)
		if dist < bestDist {
			bestDist = dist
			bestCode = b.code
		}
	}
	return bestCode
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

// assembleViewportContent builds scrollback using the render cache. Returns the
// full viewport string ready for SetContent.
func (m *chatModel) assembleViewportContent(viewWidth int) string {
	fullRebuild := m.vpRenderWidth != viewWidth ||
		m.vpRenderedMsgs > len(m.messages) ||
		m.vpStableContent == ""

	changed := m.vpRenderedMsgs != len(m.messages) ||
		len(m.messages) > 0 && len(m.messages[len(m.messages)-1].content) != m.vpLastMsgLen
	if fullRebuild || changed {
		start, offset := m.vpTailStart, m.vpTailOffset
		if fullRebuild {
			start, offset = 0, 0
		}
		var b strings.Builder
		b.WriteString(m.vpStableContent[:offset])
		for i := start; i < len(m.messages); {
			block := nextDisplayBlock(m.messages, i, len(m.messages))
			m.vpTailStart, m.vpTailOffset = block.start, b.Len()
			b.WriteString(renderDisplayBlock(m.messages, block, viewWidth, m.toolResultExpanded))
			i = block.end
		}
		m.vpStableContent = b.String()
		m.vpRenderedMsgs = len(m.messages)
		m.vpRenderWidth = viewWidth
	}

	if m.vpRenderedMsgs > 0 {
		m.vpLastMsgLen = len(m.messages[m.vpRenderedMsgs-1].content)
	} else {
		m.vpLastMsgLen = 0
	}

	welcome := m.renderWelcomeScreen(viewWidth)

	// Welcome-only state: no real messages yet — return just the welcome
	// screen so the viewport starts at top and the content fills it
	// naturally. Once the user sends a message, real content takes over.
	if m.hasRealMessages() == 0 && !m.waiting && welcome != "" {
		return welcome
	}

	content := m.vpStableContent
	if welcome != "" {
		content = welcome + "\n\n" + content
	}
	if m.waiting && !m.manualCompacting {
		content += m.renderStreamTail(viewWidth)
	}
	return content
}
