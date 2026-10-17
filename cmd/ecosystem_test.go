package cmd

import (
	"strings"
	"testing"

	reflectconfig "github.com/GrayCodeAI/reflect/internal/config"
)

func TestEcosystemCmdRuns(t *testing.T) {
	settings := reflectconfig.Settings{}
	model, provider := effectiveModelAndProvider(settings)
	if provider == "" {
		provider = "auto"
	}
	out := reflectconfig.FormatEcosystemPanel(t.Context(), provider, model)
	if !strings.Contains(out, "Ecosystem (relay · token engine)") {
		t.Fatalf("unexpected panel: %q", out)
	}
	if err := ecosystemCmd.RunE(ecosystemCmd, nil); err != nil {
		t.Fatalf("ecosystem command: %v", err)
	}
}
