package config

import (
	"context"
)

// DefaultModelProviderFilter picks which relay provider to list models for when the UI
// has no explicit filter. Host prefs (settings) win; otherwise relay routing/deployments decide.
func DefaultModelProviderFilter(ctx context.Context) string {
	if p := ActiveGateway(ctx); p != "" {
		return p
	}
	engine, err := newRelayEngine()
	if err != nil {
		return ""
	}
	return engine.DefaultProviderFilter(ctx)
}
