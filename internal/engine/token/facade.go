package token

import (
	"encoding/json"

	decodertoken "github.com/GrayCodeAI/decoder/internal/token"
)

// Stats is the compression result consumed by Decoder's runtime observations.
// The alias preserves the local token engine schema while keeping token imports
// inside this package.
type Stats = decodertoken.Stats

// UsageTracker and UsageLimits expose the session budget API through Decoder's
// token boundary without changing the local engine's accounting behavior.
type (
	UsageTracker       = decodertoken.UsageTracker
	UsageLimits        = decodertoken.UsageLimits
	CodeChunk          = decodertoken.CodeChunk
	ChunkOptions       = decodertoken.ChunkOptions
	SecretMatch        = decodertoken.SecretMatch
	SecretDetector     = decodertoken.SecretDetector
	BudgetDecision     = decodertoken.BudgetDecision
	RedactionSummary   = decodertoken.RedactionSummary
	RuntimeGraphInput  = decodertoken.RuntimeGraphInput
	RuntimeGraphExport = decodertoken.RuntimeGraphExport
)

// NewUsageTracker creates an in-memory usage tracker with the engine's defaults.
func NewUsageTracker() *UsageTracker { return decodertoken.NewUsageTracker() }

// ChunkCode splits source into semantically meaningful token-bounded chunks.
func ChunkCode(source string, opts ChunkOptions) []CodeChunk {
	return decodertoken.ChunkCode(source, opts)
}

// DefaultSecretDetector returns the engine's concurrency-safe built-in detector.
func DefaultSecretDetector() *SecretDetector { return decodertoken.DefaultSecretDetector() }

func BuildRuntimeGraph(input RuntimeGraphInput) (*RuntimeGraphExport, error) {
	return decodertoken.BuildRuntimeGraph(input)
}

// Compress applies context compression with a fixed token budget.
func Compress(text string, budget int) (string, Stats) {
	return decodertoken.Compress(text, budget)
}

// JSONInvariants renders verified-fact summaries for elided JSON records.
func JSONInvariants(dropped []json.RawMessage) string { return decodertoken.JSONInvariants(dropped) }

// ShrinkToolCatalog compresses an OpenAI-style function-tool catalog,
// preserving the selection surface byte-for-byte. Fail-open: unchanged input
// with ok=false when nothing can be safely reduced.
func ShrinkToolCatalog(catalog string) (string, bool) {
	return decodertoken.ShrinkToolCatalog(catalog)
}

// LintToolCatalog reports per-tool reductions without committing.
func LintToolCatalog(catalog string) ([]decodertoken.ToolShrinkStats, bool) {
	return decodertoken.LintToolCatalog(catalog)
}

// LogInvariants renders the level distribution of elided log lines.
func LogInvariants(lines []string) string { return decodertoken.LogInvariants(lines) }
