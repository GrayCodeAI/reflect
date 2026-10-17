package token

import reflecttoken "github.com/GrayCodeAI/reflect/internal/token"

// CountTokens returns a precise BPE-based token count for the given text.
func CountTokens(text string) int { return reflecttoken.CountTokens(text) }

// CountTokensFast returns a fast heuristic token estimate for the given text.
func CountTokensFast(text string) int { return reflecttoken.CountTokensFast(text) }

// CompressForContext compresses text to fit within a token budget,
// returning the compressed text and the final token count.
func CompressForContext(text string, budget int) (string, int) {
	compressed, stats := reflecttoken.Compress(text, budget)
	return compressed, stats.FinalTokens
}
