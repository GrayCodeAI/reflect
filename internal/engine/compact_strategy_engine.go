package engine

import (
	"context"

	"github.com/GrayCodeAI/decoder/internal/engine/compact"

	"github.com/GrayCodeAI/decoder/internal/types"
)

type CompactStrategy interface {
	Name() string
	ShouldTrigger(msgs []types.MuxMessage, tokenCount, threshold int) bool
	Compact(ctx context.Context, s *Session) (*compact.CompactResult, error)
}

type StrategyRegistry struct {
	strategies []CompactStrategy
	config     compact.CompactConfig
}

func NewStrategyRegistry(config compact.CompactConfig) *StrategyRegistry {
	r := &StrategyRegistry{config: config}
	target := config.ContextWindowSize - config.AutoCompactBuffer - config.MaxOutputTokens
	r.strategies = []CompactStrategy{
		&ProviderNativeCompactStrategy{},
		&MicroCompactStrategy{},
		&SessionMemoryStrategy{},
		&SmartCompactStrategy{},
		&RelevancePruneStrategy{TargetTokens: target},
		&TruncateStrategy{},
	}
	return r
}

func (r *StrategyRegistry) SelectStrategy(sess *Session, msgs []types.MuxMessage, tokenCount int) CompactStrategy {
	threshold := r.config.ContextWindowSize - r.config.AutoCompactBuffer - r.config.MaxOutputTokens
	for _, s := range r.strategies {
		if _, ok := s.(*ProviderNativeCompactStrategy); ok && (sess == nil || !sess.supportsNativeCompaction()) {
			continue
		}
		if s.ShouldTrigger(msgs, tokenCount, threshold) {
			return s
		}
	}
	return &TruncateStrategy{}
}
