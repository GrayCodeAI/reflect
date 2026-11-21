package cmd

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/GrayCodeAI/reflect/internal/engine/safety"

	tea "charm.land/bubbletea/v2"

	reflectconfig "github.com/GrayCodeAI/reflect/internal/config"
	"github.com/GrayCodeAI/reflect/internal/engine"
	"github.com/GrayCodeAI/reflect/internal/engine/streaming"
	"github.com/GrayCodeAI/reflect/internal/features/shellmode"
	"github.com/GrayCodeAI/reflect/internal/session"
	"github.com/GrayCodeAI/reflect/internal/tool"
	"github.com/GrayCodeAI/reflect/internal/ui/icons"
)

// submitUserMessage handles Enter on a non-empty prompt (slash commands, shell, or agent turn).
func (m chatModel) submitUserMessage() (chatModel, tea.Cmd) {
	text := strings.TrimSpace(m.input.Value())
	if text == "" {
		return m, nil
	}
	// A pending YOLO confirmation consumes the next submitted input: exact
	// (case-insensitive) match of the token confirms, anything else cancels.
	if m.pendingYOLOConfirm {
		m.pendingYOLOConfirm = false
		m.input.Reset()
		m.viewDirty = true
		if strings.EqualFold(text, yoloConfirmToken) && m.session != nil {
			m.session.PermSvc().SetAutonomy(safety.AutonomyYOLO)
			m.settings.Autonomy = permissionTierSettingValue(safety.AutonomyYOLO)
			m.settings.AutonomyExplicit = true
			m.messages = append(m.messages, displayMsg{role: "system", content: formatAutonomyTierMessage(safety.AutonomyYOLO) + " — enabled. " + icons.CloseThick() + " You will not be prompted for permission."})
		} else {
			m.messages = append(m.messages, displayMsg{role: "system", content: "Autonomy change cancelled — stayed on the previous tier."})
		}
		m.updateViewportContent()
		return m, nil
	}
	if sugs := m.slashSuggestionsFor(text); len(sugs) > 0 {
		if m.slashSel < 0 || m.slashSel >= len(sugs) {
			m.slashSel = 0
		}
		m.input.SetValue(applySlashSuggestion(sugs[m.slashSel]))
		m.input.CursorEnd()
		return m, nil
	}
	m.pushHistory(text)
	m.input.Reset()
	if strings.HasPrefix(text, "/") {
		result, cmd := m.handleCommand(text)
		if cm, ok := result.(chatModel); ok {
			m = cm
		}
		m.viewDirty = true
		m.updateViewportContent()
		return m, cmd
	}
	if strings.HasPrefix(text, "!") {
		m.termCtx.MarkCommand(text[1:])
		result, cmd := m.handleShellEscape(text[1:])
		if cm, ok := result.(chatModel); ok {
			return cm, cmd
		}
		return m, cmd
	}
	classification := m.modeManager.ClassifyWithMode(text)
	if classification == shellmode.ClassShell && !strings.HasPrefix(text, "!") {
		m.termCtx.MarkCommand(text)
		result, cmd := m.handleShellEscape(text)
		if cm, ok := result.(chatModel); ok {
			return cm, cmd
		}
		return m, cmd
	}
	if setup := reflectconfig.EvaluateSetupCached(context.Background()); setup.NeedsSetup {
		hint := setup.Hint
		if hint == "" {
			hint = "Complete setup in /config (keychain + model)."
		}
		m.messages = append(m.messages, displayMsg{role: "system", content: hint})
		m.viewDirty = true
		m.updateViewportContent()
		return m, nil
	}
	if err := m.ensureSessionReadyForChat(); err != nil {
		m.messages = append(m.messages, displayMsg{role: "error", content: err.Error()})
		m.viewDirty = true
		m.updateViewportContent()
		return m, nil
	}
	text = m.handleMentions(text)
	userDisplay := text
	text = m.termCtx.BuildContext(text)
	scale := engine.ClassifyScale(text)
	behavior := engine.GetBehavior(scale)
	_ = behavior
	if lessons := m.selfImprover.ForPrompt(5); lessons != "" {
		m.session.AppendSystemContext(lessons)
	}
	if soul := m.codingSoul.ForPrompt(); soul != "" {
		m.session.AppendSystemContext(soul)
	}
	cwd, _ := os.Getwd()
	if hints := m.hintsLoader.LoadHints(cwd); hints != "" {
		m.session.AppendSystemContext(hints)
	}
	m.messages = append(m.messages, displayMsg{role: "user", content: userDisplay})

	if imgPath := extractImagePath(text); imgPath != "" {
		if att, err := ReadImageFile(imgPath); err == nil {
			if m.session.AddUserWithAttachment(text, att.Base64, att.MIMEType) {
				m.messages = append(m.messages, displayMsg{role: "system", content: fmt.Sprintf("%s Attached image: %s", icons.Image(), filepath.Base(imgPath))})
			} else {
				m.messages = append(m.messages, displayMsg{role: "system", content: fmt.Sprintf("%s Model %q has no vision support — image %s sent as text-only note.", icons.Alert(), m.session.Model(), filepath.Base(imgPath))})
			}
		} else {
			m.session.AddUser(text)
		}
	} else if pdfPath := extractPDFPath(text); pdfPath != "" {
		if extracted, err := ReadPDFText(pdfPath); err == nil && strings.TrimSpace(extracted) != "" {
			m.session.AddUserWithDocumentText(text, filepath.Base(pdfPath), extracted)
			m.messages = append(m.messages, displayMsg{role: "system", content: fmt.Sprintf("%s Extracted text from PDF: %s", icons.FileDocument(), filepath.Base(pdfPath))})
		} else {
			m.session.AddUser(text)
			m.messages = append(m.messages, displayMsg{role: "system", content: fmt.Sprintf("%s Could not extract text from PDF: %s", icons.Alert(), filepath.Base(pdfPath))})
		}
	} else {
		m.session.AddUser(text)
	}
	m.ensureWAL()
	if m.wal != nil {
		m.walSeq++
		m.recordWALError(m.wal.Append(session.Message{Role: "user", Content: text}))
	}
	m.turn.Reset()
	m.waiting = true
	m.autoScroll = true
	m.viewDirty = true
	m.spinnerVerb = spinnerVerbs[rand.Intn(len(spinnerVerbs))] // #nosec G404 -- non-cryptographic use (random spinner verb selection)
	m.brailleSpinner.SetLabel(m.spinnerVerb)
	m.turnInputTokens = 0
	m.turnOutputTokens = 0
	m.turnEstimatedOutputRunes = 0
	m.startedAt = time.Now()
	m.partial.Reset()
	m.startStream()
	return m, tea.Batch(m.spinner.Tick, spinnerVerbTickCmd())
}

// --- Mid-run steering -------------------------------------------------------

// steerActiveRun enqueues text on the session's steering queue so the engine
// splices it into the live generation between tool batches (the same queue
// the schedule manager feeds; see the drain point in engine stream.go). It
// returns false when no session or steering queue is available, leaving the
// caller on the legacy deferred-turn queue.
func (m chatModel) steerActiveRun(text string) bool {
	if m.session == nil {
		return false
	}
	p := m.session.Persistence()
	if p == nil {
		return false
	}
	sq := p.Steering()
	if sq == nil {
		return false
	}
	sq.Enqueue(streaming.SteeringMessage{Content: text})
	return true
}

// reclaimUnconsumedSteering returns steering messages still pending after a
// run ended without draining them (e.g. a purely conversational turn with no
// tool batches) to the deferred message queue, so the existing stream-done
// auto-submit path delivers them instead of stranding them for an unrelated
// future run.
func (m *chatModel) reclaimUnconsumedSteering() {
	if m.session == nil {
		return
	}
	p := m.session.Persistence()
	if p == nil {
		return
	}
	sq := p.Steering()
	if sq == nil {
		return
	}
	for _, msg := range sq.Drain() {
		if content := strings.TrimSpace(msg.Content); content != "" {
			m.enqueueMessage(content)
		}
	}
}

// drainPersistedPromptQueue feeds persisted /prompt-queue items into the live
// steering queue as a run starts. This is the persisted queue's consumer:
// prompts queued while idle splice into the next generation between tool
// batches (or resurface as follow-up turns via reclaimUnconsumedSteering
// when the run ends without tool batches).
func (m *chatModel) drainPersistedPromptQueue() {
	if m.session == nil {
		return
	}
	p := m.session.Persistence()
	if p == nil {
		return
	}
	sq := p.Steering()
	if sq == nil {
		return
	}
	for {
		item, err := tool.DequeuePrompt()
		if err != nil {
			// Persisted queue state is unreadable; skip rather than fail the run.
			return
		}
		if item == nil {
			return
		}
		if content := strings.TrimSpace(item.Prompt); content != "" {
			sq.Enqueue(streaming.SteeringMessage{Content: content})
		}
	}
}
