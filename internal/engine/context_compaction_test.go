package engine

import (
	"context"
	"testing"

	"github.com/GrayCodeAI/decoder/internal/provider/gateway"
	"github.com/GrayCodeAI/mux/credentials"
	muxengine "github.com/GrayCodeAI/mux/engine"
)

func TestContextUsedTokens_PrefersAPI(t *testing.T) {
	s := newTestSession()
	s.AddUser("hello")
	if got := s.ContextUsedTokens(); got <= 0 {
		t.Fatalf("expected estimate > 0, got %d", got)
	}
	s.RecordAPIUsage(42_000, 500)
	if got := s.ContextUsedTokens(); got != 42_000 {
		t.Fatalf("expected API prompt tokens 42000, got %d", got)
	}
}

func TestNativeCompactionSupportUsesMuxCredentialStore(t *testing.T) {
	ctx := context.Background()
	store := &credentials.MapStore{}
	runtime, err := muxengine.New(muxengine.Options{SecretStore: store})
	if err != nil {
		t.Fatal(err)
	}
	s := NewSessionWithClient(gateway.NewFromEngine(runtime).ChatClient(), "anthropic", "claude-sonnet-4-6", "sys", nil, true)
	if s.supportsNativeCompaction() {
		t.Fatal("expected no support before Mux has a credential")
	}
	if err := store.Set(ctx, credentials.AccountForEnv("ANTHROPIC_API_KEY"), "sk-test"); err != nil {
		t.Fatal(err)
	}
	if !s.supportsNativeCompaction() {
		t.Fatal("expected support from Mux's injected credential store")
	}
}
