package token

import (
	"encoding/json"

	reflecttoken "github.com/GrayCodeAI/reflect/internal/token"
)

// Stats is the compression result consumed by Reflect's runtime observations.
// The alias preserves the local token engine schema while keeping token imports
// inside this package.
type Stats = reflecttoken.Stats

// UsageTracker and UsageLimits expose the session budget API through Reflect's
// token boundary without changing the local engine's accounting behavior.
type (
	UsageTracker       = reflecttoken.UsageTracker
	UsageLimits        = reflecttoken.UsageLimits
	CodeChunk          = reflecttoken.CodeChunk
	ChunkOptions       = reflecttoken.ChunkOptions
	SecretMatch        = reflecttoken.SecretMatch
	SecretDetector     = reflecttoken.SecretDetector
	BudgetDecision     = reflecttoken.BudgetDecision
	RedactionSummary   = reflecttoken.RedactionSummary
	RuntimeGraphInput  = reflecttoken.RuntimeGraphInput
	RuntimeGraphExport = reflecttoken.RuntimeGraphExport
)

// NewUsageTracker creates an in-memory usage tracker with the engine's defaults.
func NewUsageTracker() *UsageTracker { return reflecttoken.NewUsageTracker() }

// ChunkCode splits source into semantically meaningful token-bounded chunks.
func ChunkCode(source string, opts ChunkOptions) []CodeChunk {
	return reflecttoken.ChunkCode(source, opts)
}

// DefaultSecretDetector returns the engine's concurrency-safe built-in detector.
func DefaultSecretDetector() *SecretDetector { return reflecttoken.DefaultSecretDetector() }

func BuildRuntimeGraph(input RuntimeGraphInput) (*RuntimeGraphExport, error) {
	return reflecttoken.BuildRuntimeGraph(input)
}

// Compress applies context compression with a fixed token budget.
func Compress(text string, budget int) (string, Stats) {
	return reflecttoken.Compress(text, budget)
}

// JSONInvariants renders verified-fact summaries for elided JSON records.
func JSONInvariants(dropped []json.RawMessage) string { return reflecttoken.JSONInvariants(dropped) }

// ShrinkToolCatalog compresses an OpenAI-style function-tool catalog,
// preserving the selection surface byte-for-byte. Fail-open: unchanged input
// with ok=false when nothing can be safely reduced.
func ShrinkToolCatalog(catalog string) (string, bool) {
	return reflecttoken.ShrinkToolCatalog(catalog)
}

// LintToolCatalog reports per-tool reductions without committing.
func LintToolCatalog(catalog string) ([]reflecttoken.ToolShrinkStats, bool) {
	return reflecttoken.LintToolCatalog(catalog)
}

// LogInvariants renders the level distribution of elided log lines.
func LogInvariants(lines []string) string { return reflecttoken.LogInvariants(lines) }
