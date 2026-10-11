package token

import decodertoken "github.com/GrayCodeAI/decoder/internal/token"

// CountTokens returns a precise BPE-based token count for the given text.
func CountTokens(text string) int { return decodertoken.CountTokens(text) }

// CountTokensFast returns a fast heuristic token estimate for the given text.
func CountTokensFast(text string) int { return decodertoken.CountTokensFast(text) }

// CompressForContext compresses text to fit within a token budget,
// returning the compressed text and the final token count.
func CompressForContext(text string, budget int) (string, int) {
	compressed, stats := decodertoken.Compress(text, budget)
	return compressed, stats.FinalTokens
}
