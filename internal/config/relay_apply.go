package config

import (
	"context"
	"fmt"
	"time"

	"github.com/GrayCodeAI/reflect/internal/provider/gateway"
)

// ApplyCredentialsResult is Reflect's UI-safe view of an Relay catalog/routing
// application. It intentionally excludes Relay setup/config implementation
// types from the product boundary.
type ApplyCredentialsResult struct {
	Catalog gateway.CatalogSnapshot
}

// ApplyRelayCredentialsForProvider refreshes live models and writes sanitized
// deployment routing after /config saves a key.
func ApplyRelayCredentialsForProvider(ctx context.Context, providerID string) (*ApplyCredentialsResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	engine, err := newRelayEngine()
	if err != nil {
		return nil, err
	}
	snapshot, err := engine.ApplyCredentials(ctx, providerID)
	if err != nil {
		return nil, err
	}
	_ = SaveProjectOrGlobalDeploymentRouting(true)
	return &ApplyCredentialsResult{Catalog: snapshot}, nil
}

// ApplyRelayCredentials refreshes all configured providers and writes
// sanitized deployment routing.
func ApplyRelayCredentials(ctx context.Context) (*ApplyCredentialsResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	engine, err := newRelayEngine()
	if err != nil {
		return nil, err
	}
	snapshot, err := engine.ApplyCredentials(ctx, "")
	if err != nil {
		return nil, err
	}
	_ = SaveProjectOrGlobalDeploymentRouting(true)
	return &ApplyCredentialsResult{Catalog: snapshot}, nil
}

// RefreshGatewayCatalog refreshes catalog state through the facade.
func RefreshGatewayCatalog(ctx context.Context, providerID string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	engine, err := newRelayEngine()
	if err != nil {
		return "", err
	}
	snapshot, err := engine.RefreshCatalog(ctx, providerID)
	if err != nil {
		return "", err
	}
	return formatCatalogSnapshot(snapshot), nil
}

func FormatApplyCredentialsSummary(result *ApplyCredentialsResult) string {
	if result == nil {
		return ""
	}
	return formatCatalogSnapshot(result.Catalog)
}

func formatCatalogSnapshot(snapshot gateway.CatalogSnapshot) string {
	if snapshot.CachePath == "" {
		return fmt.Sprintf("Catalog ready: %d models", len(snapshot.Models))
	}
	return fmt.Sprintf("Catalog ready: %d models → %s", len(snapshot.Models), snapshot.CachePath)
}
