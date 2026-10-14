// Package session holds lifeboat state that is captured before context
// compaction so an agent can continue work without dropping the active
// thread, decisions, or open questions.
//
// Lifecycle: a compaction call invokes session.SaveLifeboat *before* it
// mutates the message list. The result is a small JSON record beside the
// session's transcript that later /restore flows and the SessionStart
// hook can re-inject.
package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/GrayCodeAI/decoder/internal/types"
)

// LifeboatVersion is the on-disk schema version. Bump on non-additive
// changes and migrate in Load.
const LifeboatVersion = 1

// LifeboatState captures the minimal working memory needed to survive a
// context compaction without loss.
type LifeboatState struct {
	Version         int      `json:"version"`
	SessionID       string   `json:"session_id"`
	CapturedAt      string   `json:"captured_at"`
	MessageCount    int      `json:"message_count"`
	ActiveTask      string   `json:"active_task,omitempty"`
	RecentDecisions []string `json:"recent_decisions,omitempty"`
	OpenQuestions   []string `json:"open_questions,omitempty"`
	WorkingFiles    []string `json:"working_files,omitempty"`
}

// NewLifeboatState builds a fresh, empty lifeboat for sessionID.
func NewLifeboatState(sessionID string) LifeboatState {
	return LifeboatState{
		Version:    LifeboatVersion,
		SessionID:  sessionID,
		CapturedAt: time.Now().UTC().Format(time.RFC3339),
	}
}

// BuildLifeboat derives a lifeboat from a conversation's messages.
// Extraction is heuristic on purpose: lifeboats are meant to be lightweight,
// cheap, and never to throw on content we do not understand.
func BuildLifeboat(sessionID string, messages []types.MuxMessage) LifeboatState {
	state := NewLifeboatState(sessionID)
	state.MessageCount = len(messages)

	// Working files are everything the assistant recorded it touched.
	// Working files are collected by scanning assistant content for
	// absolute unix paths or relative project files. Best-effort.
	fileSet := make(map[string]bool)
	pathRe := regexp.MustCompile(`/[A-Za-z0-9._/\-]+\.(go|md|ts|py|json)`)
	for _, m := range messages {
		for _, p := range pathRe.FindAllString(m.Content, -1) {
			if len(p) >= 6 {
				fileSet[p] = true
			}
		}
	}
	for f := range fileSet {
		state.WorkingFiles = append(state.WorkingFiles, f)
	}
	sort.Strings(state.WorkingFiles)

	// Active task is the last substantive user prompt.
	for i := len(messages) - 1; i >= 0 && state.ActiveTask == ""; i-- {
		if messages[i].Role != "user" {
			continue
		}
		t := strings.TrimSpace(messages[i].Content)
		if t == "" {
			continue
		}
		if len(t) > 280 {
			t = t[:280] + "…"
		}
		state.ActiveTask = t
	}

	seen := make(map[string]bool)
	addDecision := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			return
		}
		seen[s] = true
		state.RecentDecisions = append(state.RecentDecisions, s)
	}
	addQuestion := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			return
		}
		seen[s] = true
		state.OpenQuestions = append(state.OpenQuestions, s)
	}

	for _, m := range messages {
		if m.Role != "assistant" {
			continue
		}
		text := m.Content
		for _, line := range strings.Split(text, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			lower := strings.ToLower(line)
			if strings.Contains(lower, "decision:") || strings.Contains(lower, "decided to") || strings.Contains(lower, "will use") {
				addDecision(trimLine(line, 200))
			}
			if strings.HasSuffix(line, "?") {
				addQuestion(trimLine(line, 200))
			}
		}
	}
	// Keep only the most recent handful to stay small.
	if len(state.RecentDecisions) > 8 {
		state.RecentDecisions = state.RecentDecisions[len(state.RecentDecisions)-8:]
	}
	if len(state.OpenQuestions) > 6 {
		state.OpenQuestions = state.OpenQuestions[len(state.OpenQuestions)-6:]
	}
	return state
}

func trimLine(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// LifeboatPath returns the on-disk location of sessionID's lifeboat.
func LifeboatPath(id string) (string, error) {
	if err := ValidateID(id); err != nil {
		return "", err
	}
	return filepath.Join(sessionsDir(), id+".lifeboat.json"), nil
}

// SaveLifeboat writes state atomically with owner-only permissions.
func SaveLifeboat(id string, state LifeboatState) (string, error) {
	path, err := LifeboatPath(id)
	if err != nil {
		return "", err
	}
	if state.SessionID == "" {
		state.SessionID = id
	}
	if state.Version == 0 {
		state.Version = LifeboatVersion
	}
	if state.CapturedAt == "" {
		state.CapturedAt = time.Now().UTC().Format(time.RFC3339)
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal lifeboat: %w", err)
	}
	data = append(data, '\n')

	// 0600: like the session transcript itself, the lifeboat carries
	// working-memory context we do not want other users reading.
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", fmt.Errorf("create sessions dir: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return "", fmt.Errorf("write lifeboat tmp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("commit lifeboat: %w", err)
	}
	return path, nil
}

// LoadLifeboat reads a previously saved lifeboat. A missing file is a
// normal "no data" condition served by ErrNotFound.
func LoadLifeboat(id string) (LifeboatState, error) {
	path, err := LifeboatPath(id)
	if err != nil {
		return LifeboatState{}, err
	}
	f, err := os.Open(path) // #nosec G304 -- path constructed from validated session id
	if err != nil {
		if os.IsNotExist(err) {
			return LifeboatState{}, ErrNotFound
		}
		return LifeboatState{}, fmt.Errorf("open lifeboat: %w", err)
	}
	defer func() { _ = f.Close() }()

	var state LifeboatState
	if err := json.NewDecoder(f).Decode(&state); err != nil {
		return LifeboatState{}, fmt.Errorf("decode lifeboat: %w", err)
	}
	if state.Version == 0 {
		state.Version = LifeboatVersion
	}
	if state.Version > LifeboatVersion {
		return state, fmt.Errorf("lifeboat version %d newer than supported %d", state.Version, LifeboatVersion)
	}
	return state, nil
}

// RemoveLifeboat deletes the lifeboat; missing is not an error.
func RemoveLifeboat(id string) error {
	path, err := LifeboatPath(id)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// HasLifeboat reports whether a lifeboat exists for id.
func HasLifeboat(id string) (bool, error) {
	path, err := LifeboatPath(id)
	if err != nil {
		return false, err
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
