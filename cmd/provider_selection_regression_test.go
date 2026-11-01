package cmd

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	reflectconfig "github.com/GrayCodeAI/reflect/internal/config"
	"github.com/GrayCodeAI/reflect/internal/provider/gateway"
	"github.com/GrayCodeAI/reflect/internal/storage"
)

func TestReflectionSelectionAndDashboardUsePublishedRuntime(t *testing.T) {
	isolateCredentialHome(t)
	storage.SetTestDirs(t, t.TempDir())
	store := &gateway.MapStore{}
	gateway.SetDefaultStore(store)
	reflectconfig.InvalidateConfigUICache()
	t.Cleanup(func() { gateway.SetDefaultStore(nil); reflectconfig.InvalidateConfigUICache() })
	ctx := context.Background()
	if err := store.Set(ctx, gateway.AccountForEnv("REFLECTION_API_KEY"), "reflection-test-credential"); err != nil {
		t.Fatal(err)
	}
	state := `{"config_version":2,"active_provider":"reflection","active_model":"reflection/Beam-1.0-501B-A23B","deployments":{"reflection-direct":{}}}`
	path := storage.ProviderConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(state), 0o600); err != nil {
		t.Fatal(err)
	}
	selection := startupSelection(reflectconfig.Settings{})
	if selection.Provider != "reflection" || selection.Model != "reflection/Beam-1.0-501B-A23B" {
		t.Fatalf("startup replaced explicit selection: %#v", selection)
	}
	m := singlePaneTestModel(160, 40)
	m.session = newStartupReflectSession(selection, "test", nil)
	if got, _, _ := m.connectionStatusParts(); got != "Reflection AI" {
		t.Fatalf("dashboard provider = %q", got)
	}
	for _, warning := range validateStartup(reflectconfig.Settings{Provider: selection.Provider}) {
		if warning.Check == "api_key" || warning.Check == "provider" {
			t.Fatalf("stored Reflection credential was not recognized: %s", warning)
		}
	}
}

func TestValidateStartupCredentialHintsUseInvocationSettings(t *testing.T) {
	isolateCredentialHome(t)
	storage.SetTestDirs(t, t.TempDir())
	t.Setenv("REFLECTION_API_KEY", "")
	store := &gateway.MapStore{}
	gateway.SetDefaultStore(store)
	t.Cleanup(func() { gateway.SetDefaultStore(nil); reflectconfig.InvalidateConfigUICache() })
	for _, tt := range []struct {
		provider, env string
		custom        bool
	}{
		{"reflection", "REFLECTION_API_KEY", false},
		{"custom-test", "CUSTOM_TEST_API_KEY", true},
		{"unsupported-test", "", false},
	} {
		t.Run(tt.provider, func(t *testing.T) {
			settings := reflectconfig.Settings{Provider: tt.provider}
			if tt.custom {
				settings.CustomProviders = []reflectconfig.CustomProviderConfig{{Name: tt.provider, BaseURL: "https://example.com/v1", APIKeyEnv: tt.env, Model: "test-model"}}
			}
			if tt.env != "" {
				t.Setenv(tt.env, "")
			}
			found := false
			for _, warning := range validateStartup(settings) {
				if warning.Check != "api_key" {
					continue
				}
				found = true
				if strings.Contains(warning.Message, "Set  ") || !strings.Contains(warning.Message, "/config") || tt.env != "" && !strings.Contains(warning.Message, tt.env) {
					t.Fatalf("invalid credential hint: %q", warning.Message)
				}
			}
			if !found {
				t.Fatal("missing credential diagnostic")
			}
		})
	}
	// A custom no-key endpoint must not be reported as missing credentials.
	settings := reflectconfig.Settings{Provider: "local-test", CustomProviders: []reflectconfig.CustomProviderConfig{{Name: "local-test", BaseURL: "http://localhost:8000/v1", Model: "test-model"}}}
	for _, warning := range validateStartup(settings) {
		if warning.Check == "api_key" {
			t.Fatalf("no-key gateway warned: %s", warning)
		}
	}
}

func TestReflectionRegistryIncludesCredentialAndDeployment(t *testing.T) {
	storage.SetTestDirs(t, t.TempDir())
	runtime, err := reflectconfig.NewRelayEngineForSettings(reflectconfig.Settings{})
	if err != nil {
		t.Fatal(err)
	}
	for _, definition := range runtime.GatewayDefinitions() {
		if definition.ID != "reflection" {
			continue
		}
		if definition.CredentialEnv != "REFLECTION_API_KEY" {
			t.Fatalf("credential environment = %q", definition.CredentialEnv)
		}
		inference, err := reflectconfig.CredentialInferenceForProvider("reflection")
		if err != nil || inference.DeploymentID != "reflection-direct" {
			t.Fatalf("deployment = %q, %v", inference.DeploymentID, err)
		}
		return
	}
	metadata, _ := json.Marshal(runtime.GatewayDefinitions())
	t.Fatalf("the pinned Relay runtime omits Reflection support: %s", metadata)
}
