package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GrayCodeAI/decoder"
	decoderconfig "github.com/GrayCodeAI/decoder/internal/config"
	"github.com/GrayCodeAI/decoder/internal/provider/gateway"
)

func TestDisplayVersion_EmbeddedVersionIgnoresWorkingDirectory(t *testing.T) {
	preserveCLICompilerVersionState(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "VERSION"), []byte("9.9.9\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	for _, value := range []string{"", "dev"} {
		SetVersion(value)
		if got := DisplayVersion(); got != decoder.Version {
			t.Fatalf("got %q, want embedded %q", got, decoder.Version)
		}
	}
}

func TestDisplayVersion_ReleaseBuild(t *testing.T) {
	preserveCLICompilerVersionState(t)
	SetVersion("1.4.2")
	if got := DisplayVersion(); got != "1.4.2" {
		t.Fatalf("got %q, want 1.4.2", got)
	}
}

func TestChatConnectionStatus_NoCredentials(t *testing.T) {
	decoderconfig.InvalidateConfigUICache()
	store := &gateway.MapStore{}
	gateway.SetDefaultStore(store)
	t.Cleanup(func() {
		gateway.SetDefaultStore(nil)
		decoderconfig.InvalidateConfigUICache()
	})

	m := chatModel{session: nil}
	got := m.chatConnectionStatus()
	if got != "" {
		t.Fatalf("connection status = %q, want empty when unconfigured", got)
	}
}

func TestChatBottomRightStatus_NoCredentials(t *testing.T) {
	decoderconfig.InvalidateConfigUICache()
	store := &gateway.MapStore{}
	gateway.SetDefaultStore(store)
	t.Cleanup(func() {
		gateway.SetDefaultStore(nil)
		decoderconfig.InvalidateConfigUICache()
	})

	m := chatModel{inputIndicator: &InputIndicator{}}
	got := m.chatBottomRightStatus()
	if got != "" {
		t.Fatalf("status = %q, want empty when no keys", got)
	}
}
