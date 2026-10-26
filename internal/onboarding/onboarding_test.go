package onboarding

import (
	"bufio"
	"io"
	"os"
	"strings"
	"testing"

	reflectconfig "github.com/GrayCodeAI/reflect/internal/config"
)

func TestNeedsSetup_AlwaysFalseForTUI(t *testing.T) {
	if NeedsSetup() {
		t.Error("NeedsSetup() should be false; use /config or reflect setup instead")
	}
}

func TestPreferredSetupModelChoosesVerifiedToolModel(t *testing.T) {
	models := []reflectconfig.EngineModel{
		{ID: "model-unknown", CanonicalID: "provider/model-unknown"},
		{ID: "model-tools", CanonicalID: "provider/model-tools", Capabilities: []string{"tools"}},
	}
	if got := preferredSetupModel(models, "provider/model-unknown", "model-tools"); got != 1 {
		t.Fatalf("default model = %d, want verified tool model", got)
	}
	if got := preferredSetupModel(models, "provider/model-tools", "model-unknown"); got != 1 {
		t.Fatalf("saved canonical model = %d, want matching tool model", got)
	}
}

func TestReadSetupSecretFromNonTerminalInput(t *testing.T) {
	got, err := readSetupSecret(bufio.NewReader(strings.NewReader("  opaque-provider-key  \n")))
	if err != nil || got != "opaque-provider-key" {
		t.Fatalf("secret read = %q, err=%v", got, err)
	}
}

func TestSetupProvidersHaveRelayCredentialMappings(t *testing.T) {
	for _, provider := range []string{"anthropic", "openai", "gemini", "openrouter", "groq", "poolside"} {
		if !reflectconfig.IsSetupGateway(provider) || reflectconfig.SetupGatewayCredentialEnv(provider) == "" {
			t.Errorf("%s cannot be configured through Relay", provider)
		}
	}
	if !reflectconfig.IsSetupGateway("ollama") {
		t.Fatal("Ollama is missing from Relay gateways")
	}
}

func TestValidateAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		key      string
		valid    bool
	}{
		{"valid anthropic", "anthropic", "sk-ant-api01-abcdefghijklmnopqrstuvwxyz", true},
		{"valid openai", "openai", "sk-abcdefghijklmnopqrstuvwxyz123456", true},
		{"short opaque key", "anthropic", "sk-ant", true},
		{"custom anthropic prefix", "anthropic", "wrong-prefix-long-enough-key", true},
		{"custom openai prefix", "openai", "not-sk-prefix-long-enough-key", true},
		{"unknown provider accepts any", "gemini", "any-key-long-enough-to-be-valid", true},
		{"empty key", "anthropic", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, valid := validateAPIKey(tt.provider, tt.key)
			if valid != tt.valid {
				t.Errorf("validateAPIKey(%q, %q) valid = %v, want %v", tt.provider, tt.key, valid, tt.valid)
			}
		})
	}
}

func TestWelcome(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	oldStdout := os.Stdout
	os.Stdout = writer
	t.Cleanup(func() { os.Stdout = oldStdout })
	Welcome("1.0.0")
	_ = writer.Close()
	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(output), "Reflect  v1.0.0") || strings.Contains(string(output), "██") {
		t.Fatalf("welcome should use the plain wordmark: %q", output)
	}
}

func TestColorConstants(t *testing.T) {
	// Color codes are gated on the environment: present when color is forced,
	// empty when NO_COLOR is set (the wizard stays plain in scripted output).
	t.Setenv("NO_COLOR", "")
	t.Setenv("FORCE_COLOR", "1")
	initColorCodes()
	if teal == "" {
		t.Error("teal color should not be empty when color is enabled")
	}
	if reset == "" {
		t.Error("reset should not be empty when color is enabled")
	}
	if bold == "" {
		t.Error("bold should not be empty when color is enabled")
	}

	t.Setenv("NO_COLOR", "1")
	initColorCodes()
	if teal != "" {
		t.Error("teal should be empty when NO_COLOR is set")
	}
	if reset != "" {
		t.Error("reset should be empty when NO_COLOR is set")
	}
}
