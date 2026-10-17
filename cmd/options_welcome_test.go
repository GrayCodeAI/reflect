package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	reflectconfig "github.com/GrayCodeAI/reflect/internal/config"
	"github.com/GrayCodeAI/reflect/internal/provider/gateway"
)

func isolateCredentialHome(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	reflectDir := filepath.Join(home, ".reflect")
	_ = os.MkdirAll(reflectDir, 0o700)
	t.Setenv("HOME", home)
	t.Setenv("REFLECT_CONFIG_DIR", reflectDir)
	t.Setenv("RELAY_CONFIG_DIR", filepath.Join(home, "relay"))
}

func TestEffectiveModelAndProvider_ClearsWithoutCredentials(t *testing.T) {
	reflectconfig.InvalidateConfigUICache()
	isolateCredentialHome(t)
	store := &gateway.MapStore{}
	gateway.SetDefaultStore(store)
	t.Cleanup(func() {
		gateway.SetDefaultStore(nil)
		reflectconfig.InvalidateConfigUICache()
	})

	ctx := context.Background()
	if err := reflectconfig.SetActiveProvider(ctx, "openrouter"); err != nil {
		t.Fatal(err)
	}
	if err := reflectconfig.SetActiveModel(ctx, "gpt-4o"); err != nil {
		t.Fatal(err)
	}

	model, provider := effectiveModelAndProvider(reflectconfig.Settings{})
	if model != "" || provider != "" {
		t.Fatalf("expected empty selection without credentials, got model=%q provider=%q", model, provider)
	}
}

func TestEffectiveModelAndProvider_KeepsWithCredentials(t *testing.T) {
	reflectconfig.InvalidateConfigUICache()
	isolateCredentialHome(t)
	store := &gateway.MapStore{}
	gateway.SetDefaultStore(store)
	t.Cleanup(func() {
		gateway.SetDefaultStore(nil)
		reflectconfig.InvalidateConfigUICache()
	})

	ctx := context.Background()
	_ = store.Set(ctx, gateway.AccountForEnv("OPENROUTER_API_KEY"), "sk-or-test-key-1234567890")
	reflectconfig.InvalidateConfigUICache()
	if err := reflectconfig.SetActiveProvider(ctx, "openrouter"); err != nil {
		t.Fatal(err)
	}
	if err := reflectconfig.SetActiveModel(ctx, "gpt-4o"); err != nil {
		t.Fatal(err)
	}

	model, provider := effectiveModelAndProvider(reflectconfig.Settings{})
	if provider == "" {
		t.Fatalf("expected provider with credentials, got model=%q provider=%q", model, provider)
	}
	if strings.TrimSpace(model) == "" {
		t.Fatalf("expected model preserved, got model=%q provider=%q", model, provider)
	}
}
