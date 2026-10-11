package cmd

import (
	"context"
	"os"

	decoderconfig "github.com/GrayCodeAI/decoder/internal/config"
)

var (
	refreshCatalogFlag     bool
	skipCatalogRefreshFlag bool
)

func ensureCatalogBeforeAgent(ctx context.Context, strict bool) error {
	opts := decoderconfig.CatalogStartupOptions{
		ForceRefresh:    refreshCatalogFlag,
		SkipAutoRefresh: skipCatalogRefreshFlag,
		VerboseOutput:   refreshCatalogFlag,
	}
	if strict {
		return decoderconfig.PrepareCatalogForSession(ctx, os.Stderr, opts)
	}
	decoderconfig.StartupCatalogPrefetch(ctx)
	return nil
}

func startBackgroundCatalogRefresh(ctx context.Context) {
	if skipCatalogRefreshFlag {
		return
	}
	decoderconfig.ScheduleBackgroundCatalogRefresh(ctx)
}
