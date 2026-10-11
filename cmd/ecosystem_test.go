package cmd

import (
	"strings"
	"testing"

	decoderconfig "github.com/GrayCodeAI/decoder/internal/config"
)

func TestEcosystemCmdRuns(t *testing.T) {
	settings := decoderconfig.Settings{}
	model, provider := effectiveModelAndProvider(settings)
	if provider == "" {
		provider = "auto"
	}
	out := decoderconfig.FormatEcosystemPanel(t.Context(), provider, model)
	if !strings.Contains(out, "Ecosystem (mux · token engine)") {
		t.Fatalf("unexpected panel: %q", out)
	}
	if err := ecosystemCmd.RunE(ecosystemCmd, nil); err != nil {
		t.Fatalf("ecosystem command: %v", err)
	}
}
