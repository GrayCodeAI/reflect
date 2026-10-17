package cmd

import (
	"context"
	"os"

	reflectconfig "github.com/GrayCodeAI/reflect/internal/config"
)

var (
	refreshCatalogFlag     bool
	skipCatalogRefreshFlag bool
)

func ensureCatalogBeforeAgent(ctx context.Context, strict bool) error {
	opts := reflectconfig.CatalogStartupOptions{
		ForceRefresh:    refreshCatalogFlag,
		SkipAutoRefresh: skipCatalogRefreshFlag,
		VerboseOutput:   refreshCatalogFlag,
	}
	if strict {
		return reflectconfig.PrepareCatalogForSession(ctx, os.Stderr, opts)
	}
	reflectconfig.StartupCatalogPrefetch(ctx)
	return nil
}

func startBackgroundCatalogRefresh(ctx context.Context) {
	if skipCatalogRefreshFlag {
		return
	}
	reflectconfig.ScheduleBackgroundCatalogRefresh(ctx)
}
