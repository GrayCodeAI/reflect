package session

import (
	"fmt"
	"strings"

	store "github.com/GrayCodeAI/decoder/internal/session"
)

// QuitResumeMessage renders the stable exit message for a session-aware CLI.
func QuitResumeMessage(sessionID string) string {
	if strings.TrimSpace(sessionID) == "" {
		return "Thank you for using Decoder!\n"
	}
	return fmt.Sprintf("Thank you for using Decoder!\n\nTo resume this session, run: decoder --resume %s\n", sessionID)
}

// RecoveryReport renders recovery candidates consistently across the TUI and
// non-interactive CLI. The durable session package remains the source of its
// established report format while this feature owns the presentation seam.
func RecoveryReport(candidates []store.RecoveryCandidate) string {
	return store.FormatRecoveryCandidates(candidates)
}
