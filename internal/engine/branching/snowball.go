package branching

import (
	"fmt"
	"strings"
)

// SnowballDetector detects when token consumption is growing faster than
// progress, signalling that the agent is stuck in a snowball pattern where
// each turn consumes more tokens without proportional progress.
type SnowballDetector struct {
	turnTokens   []int     // tokens consumed per turn
	turnProgress []float64 // estimated progress per turn (0-1)
	threshold    float64   // growth rate threshold for alarm
	maxTokens    int       // absolute ceiling
}

// NewSnowballDetector creates a detector with the given absolute token ceiling.
// A non-positive ceiling leaves token limits to the configured session budget.
// The default growth threshold is 2.0x (last 3 turns vs first 3 turns).
func NewSnowballDetector(maxTokens int) *SnowballDetector {
	return &SnowballDetector{
		threshold: 2.0,
		maxTokens: maxTokens,
	}
}

// RecordTurn records token usage and estimated progress for a single turn.
// Progress should be in the range [0, 1].
func (sd *SnowballDetector) RecordTurn(tokens int, progress float64) {
	sd.turnTokens = append(sd.turnTokens, tokens)
	sd.turnProgress = append(sd.turnProgress, progress)
}

// IsSnowballing reports rising usage accompanied by declining activity.
// Growing prompts alone are normal during productive work and must not stop it.
func (sd *SnowballDetector) IsSnowballing() bool {
	n := len(sd.turnTokens)
	if n < 6 {
		return false // need at least 6 turns to compare first 3 vs last 3
	}

	// Average tokens for first 3 turns
	firstAvg := avgInt(sd.turnTokens[:3])
	// Average tokens for last 3 turns
	lastAvg := avgInt(sd.turnTokens[n-3:])

	if firstAvg == 0 {
		return false
	}

	growthRate := float64(lastAvg) / float64(firstAvg)
	return growthRate >= sd.threshold && sd.progressTrend() == "declining"
}

// ShouldAbort returns true if total tokens exceed maxTokens or the growth rate
// exceeds 3x between the first and last 3-turn windows while activity declines.
func (sd *SnowballDetector) ShouldAbort() bool {
	total := sumInt(sd.turnTokens)
	if sd.maxTokens > 0 && total > sd.maxTokens {
		return true
	}

	n := len(sd.turnTokens)
	if n < 6 {
		return false
	}

	firstAvg := avgInt(sd.turnTokens[:3])
	lastAvg := avgInt(sd.turnTokens[n-3:])
	if firstAvg == 0 {
		return false
	}

	return float64(lastAvg)/float64(firstAvg) >= 3.0 && sd.progressTrend() == "declining"
}

// StopReason explains the active stop condition without exposing debug output
// or claiming that a large cumulative input count means context is full.
func (sd *SnowballDetector) StopReason() string {
	if sd.maxTokens > 0 && sumInt(sd.turnTokens) > sd.maxTokens {
		return fmt.Sprintf("Token limit reached (%s/%s total input and output tokens).", formatTokenCount(sumInt(sd.turnTokens)), formatTokenCount(sd.maxTokens))
	}
	return "Token usage increased sharply while tool activity declined."
}

// Summary returns a human-readable summary of the snowball state.
func (sd *SnowballDetector) Summary() string {
	n := len(sd.turnTokens)
	total := sumInt(sd.turnTokens)

	totalStr := formatTokenCount(total)

	var growthStr string
	if n >= 6 {
		firstAvg := avgInt(sd.turnTokens[:3])
		lastAvg := avgInt(sd.turnTokens[n-3:])
		if firstAvg > 0 {
			growthStr = fmt.Sprintf("%.1fx initial usage", float64(lastAvg)/float64(firstAvg))
		} else {
			growthStr = "no baseline"
		}
	} else {
		growthStr = "insufficient data"
	}

	progressTrend := sd.progressTrend()

	var recommendation string
	if sd.ShouldAbort() {
		recommendation = "abort and retry with fresh context"
	} else if sd.IsSnowballing() {
		recommendation = "consider aborting"
	} else {
		recommendation = "continue"
	}

	return fmt.Sprintf("Turns: %d | Tokens: %s (%s) | Progress: %s | Recommendation: %s",
		n, totalStr, growthStr, progressTrend, recommendation)
}

// Reset clears all recorded data.
func (sd *SnowballDetector) Reset() {
	sd.turnTokens = nil
	sd.turnProgress = nil
}

// progressTrend returns a string describing whether progress is increasing,
// stable, or declining.
func (sd *SnowballDetector) progressTrend() string {
	n := len(sd.turnProgress)
	if n < 4 {
		return "insufficient data"
	}

	mid := n / 2
	firstHalf := avgFloat(sd.turnProgress[:mid])
	secondHalf := avgFloat(sd.turnProgress[mid:])

	if secondHalf < firstHalf*0.8 {
		return "declining"
	}
	if secondHalf > firstHalf*1.2 {
		return "increasing"
	}
	return "stable"
}

// formatTokenCount formats a token count as a human-readable string (e.g. "45K").
func formatTokenCount(tokens int) string {
	if tokens >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(tokens)/1_000_000)
	}
	if tokens >= 1_000 {
		return fmt.Sprintf("%dK", tokens/1_000)
	}
	return fmt.Sprintf("%d", tokens)
}

func sumInt(s []int) int {
	total := 0
	for _, v := range s {
		total += v
	}
	return total
}

func avgInt(s []int) int {
	if len(s) == 0 {
		return 0
	}
	return sumInt(s) / len(s)
}

func avgFloat(s []float64) float64 {
	if len(s) == 0 {
		return 0
	}
	total := 0.0
	for _, v := range s {
		total += v
	}
	return total / float64(len(s))
}

// String implements the Stringer interface for SnowballDetector.
func (sd *SnowballDetector) String() string {
	var b strings.Builder
	b.WriteString("SnowballDetector{")
	b.WriteString(fmt.Sprintf("turns=%d", len(sd.turnTokens)))
	b.WriteString(fmt.Sprintf(", maxTokens=%d", sd.maxTokens))
	b.WriteString(fmt.Sprintf(", threshold=%.1f", sd.threshold))
	b.WriteString("}")
	return b.String()
}
