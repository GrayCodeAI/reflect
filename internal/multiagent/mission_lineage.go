package mission

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LoadMission reads a persisted mission from dir/mission.json.
func LoadMission(dir string) (*Mission, error) {
	path := filepath.Join(dir, "mission.json")
	data, err := os.ReadFile(path) // #nosec G304 -- caller supplies a mission directory path
	if err != nil {
		return nil, err
	}
	var m Mission
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// LineageMarkdown renders a deterministic, read-only explanation of why the
// mission's artifacts exist: prompt, features (status, branch, attempts),
// wave joins, and handoffs. It never mutates state.
func (m *Mission) LineageMarkdown() string {
	if m == nil {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# Mission %s\n\n", m.ID)
	fmt.Fprintf(&b, "**Status:** %s  \n", m.Status)
	fmt.Fprintf(&b, "**Prompt:** %s\n\n", strings.ReplaceAll(m.Prompt, "\n", " "))
	if !m.StartedAt.IsZero() {
		fmt.Fprintf(&b, "**Started:** %s\n", m.StartedAt.UTC().Format(time.RFC3339))
	}
	if !m.CompletedAt.IsZero() {
		fmt.Fprintf(&b, "**Completed:** %s\n", m.CompletedAt.UTC().Format(time.RFC3339))
	}
	fmt.Fprintf(&b, "\n## Features (%d)\n\n", len(m.Features))
	for _, f := range m.Features {
		status := string(f.Status)
		if status == "" {
			status = "unknown"
		}
		fmt.Fprintf(&b, "### %s — %s\n", f.ID, status)
		fmt.Fprintf(&b, "- **Description:** %s\n", f.Description)
		if f.Branch != "" {
			fmt.Fprintf(&b, "- **Branch:** %s\n", f.Branch)
		}
		if f.RetryCount > 0 {
			fmt.Fprintf(&b, "- **Retries:** %d\n", f.RetryCount)
		}
		if f.Handoff != nil {
			h := f.Handoff
			if h.Attempts > 0 {
				fmt.Fprintf(&b, "- **Attempts:** %d\n", h.Attempts)
			}
			if len(h.FilesChanged) > 0 {
				fmt.Fprintf(&b, "- **Files:** %s\n", strings.Join(h.FilesChanged, ", "))
			}
			if h.Summary != "" {
				fmt.Fprintf(&b, "- **Summary:** %s\n", h.Summary)
			}
			fmt.Fprintf(&b, "- **Tests passed:** %t\n", h.TestsPassed)
		}
		if f.VerificationResult != "" {
			fmt.Fprintf(&b, "- **Verification:** %s\n", f.VerificationResult)
		}
		fmt.Fprintln(&b)
	}
	if len(m.WaveJoins) > 0 {
		fmt.Fprintf(&b, "## Wave joins (%d)\n\n", len(m.WaveJoins))
		for _, w := range m.WaveJoins {
			fmt.Fprintf(&b, "- Wave %d: %d features, %d completed, %d failed; %s\n",
				w.Wave, len(w.FeatureIDs), len(w.CompletedIDs), len(w.FailedIDs), w.Summary)
		}
	}
	return b.String()
}

// ExportGASP writes an experimental portable-artifact snapshot. It is only an
// alias of the portable graph today; consumers that need strict GASP
// conformance should validate it with the upstream checker.
func (m *Mission) ExportGASP() error {
	if m == nil || strings.TrimSpace(m.Dir) == "" {
		return fmt.Errorf("mission has no directory")
	}
	data, err := os.ReadFile(filepath.Join(m.Dir, portableGraphFilename))
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(m.Dir, "mission.gasp.json"), data, 0o600)
}
