package config

import (
	"context"
)

// DefaultModelProviderFilter picks which mux provider to list models for when the UI
// has no explicit filter. Host prefs (settings) win; otherwise mux routing/deployments decide.
func DefaultModelProviderFilter(ctx context.Context) string {
	if p := ActiveGateway(ctx); p != "" {
		return p
	}
	engine, err := newMuxEngine()
	if err != nil {
		return ""
	}
	return engine.DefaultProviderFilter(ctx)
}
