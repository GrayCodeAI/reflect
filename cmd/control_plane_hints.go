package cmd

import (
	"fmt"

	"github.com/GrayCodeAI/reflect/internal/engine"
)

// controlPlaneOnboardingHint is a short first-session tip (not a wall of text).
func controlPlaneOnboardingHint(sess *engine.Session) string {
	return ""
}

// workModeSwitchSummary is the polished confirmation after /mode plan|build|review.
func workModeSwitchSummary(sess *engine.Session, wm engine.WorkMode) string {
	hint := ""
	switch wm {
	case engine.WorkModePlan:
		hint = "Research and prepare one plan. Project files stay read-only; Shift+Tab switches to Build."
	case engine.WorkModeReview:
		hint = "Inspect with evidence — no writes; bash is read-only."
	default:
		hint = "Build mode — essential tools visible; use ToolSearch select:Name for more."
	}
	visible := 0
	if sess != nil && sess.Tools() != nil && sess.Tools().Registry() != nil {
		visible = len(sess.Tools().Registry().RelayTools())
	}
	return fmt.Sprintf("Work mode → %s\n%s\nModel-visible tools: %d  ·  /status for details", wm.DisplayName(), hint, visible)
}
