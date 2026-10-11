package config

import (
	"context"
	"fmt"
	"time"

	"github.com/GrayCodeAI/decoder/internal/provider/gateway"
)

// ApplyCredentialsResult is Decoder's UI-safe view of an Mux catalog/routing
// application. It intentionally excludes Mux setup/config implementation
// types from the product boundary.
type ApplyCredentialsResult struct {
	Catalog gateway.CatalogSnapshot
}

// ApplyMuxCredentialsForProvider refreshes live models and writes sanitized
// deployment routing after /config saves a key.
func ApplyMuxCredentialsForProvider(ctx context.Context, providerID string) (*ApplyCredentialsResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	engine, err := newMuxEngine()
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

// ApplyMuxCredentials refreshes all configured providers and writes
// sanitized deployment routing.
func ApplyMuxCredentials(ctx context.Context) (*ApplyCredentialsResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	engine, err := newMuxEngine()
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
	engine, err := newMuxEngine()
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
