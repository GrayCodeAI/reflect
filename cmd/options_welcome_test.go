package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	decoderconfig "github.com/GrayCodeAI/decoder/internal/config"
	"github.com/GrayCodeAI/decoder/internal/provider/gateway"
)

func isolateCredentialHome(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	decoderDir := filepath.Join(home, ".decoder")
	_ = os.MkdirAll(decoderDir, 0o700)
	t.Setenv("HOME", home)
	t.Setenv("DECODER_CONFIG_DIR", decoderDir)
	t.Setenv("MUX_CONFIG_DIR", filepath.Join(home, "mux"))
}

func TestEffectiveModelAndProvider_ClearsWithoutCredentials(t *testing.T) {
	decoderconfig.InvalidateConfigUICache()
	isolateCredentialHome(t)
	store := &gateway.MapStore{}
	gateway.SetDefaultStore(store)
	t.Cleanup(func() {
		gateway.SetDefaultStore(nil)
		decoderconfig.InvalidateConfigUICache()
	})

	ctx := context.Background()
	if err := decoderconfig.SetActiveProvider(ctx, "openrouter"); err != nil {
		t.Fatal(err)
	}
	if err := decoderconfig.SetActiveModel(ctx, "gpt-4o"); err != nil {
		t.Fatal(err)
	}

	model, provider := effectiveModelAndProvider(decoderconfig.Settings{})
	if model != "" || provider != "" {
		t.Fatalf("expected empty selection without credentials, got model=%q provider=%q", model, provider)
	}
}

func TestEffectiveModelAndProvider_KeepsWithCredentials(t *testing.T) {
	decoderconfig.InvalidateConfigUICache()
	isolateCredentialHome(t)
	store := &gateway.MapStore{}
	gateway.SetDefaultStore(store)
	t.Cleanup(func() {
		gateway.SetDefaultStore(nil)
		decoderconfig.InvalidateConfigUICache()
	})

	ctx := context.Background()
	_ = store.Set(ctx, gateway.AccountForEnv("OPENROUTER_API_KEY"), "sk-or-test-key-1234567890")
	decoderconfig.InvalidateConfigUICache()
	if err := decoderconfig.SetActiveProvider(ctx, "openrouter"); err != nil {
		t.Fatal(err)
	}
	if err := decoderconfig.SetActiveModel(ctx, "gpt-4o"); err != nil {
		t.Fatal(err)
	}

	model, provider := effectiveModelAndProvider(decoderconfig.Settings{})
	if provider == "" {
		t.Fatalf("expected provider with credentials, got model=%q provider=%q", model, provider)
	}
	if strings.TrimSpace(model) == "" {
		t.Fatalf("expected model preserved, got model=%q provider=%q", model, provider)
	}
}
