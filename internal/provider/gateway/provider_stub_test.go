package gateway

import (
	"context"
	"testing"

	"github.com/GrayCodeAI/reflect/internal/types"
	relayengine "github.com/GrayCodeAI/relay/engine"
)

// stubProvider proves the Provider interface is swappable: a test can inject a
// deterministic fake and drive the ChatClient adapter without constructing any
// Relay engine. This is the swappable-Engine guarantee Phase 3 delivers.
type stubProvider struct {
	resp *relayengine.GenerateResponse
	err  error
}

func (s *stubProvider) Resolve(context.Context, relayengine.SelectionRequest) (relayengine.Route, error) {
	return relayengine.Route{}, nil
}

func (s *stubProvider) Generate(context.Context, relayengine.GenerateRequest) (*relayengine.GenerateResponse, error) {
	return s.resp, s.err
}

func (s *stubProvider) Stream(context.Context, relayengine.GenerateRequest) (relayengine.EventStreamer, error) {
	return nil, nil
}

func (s *stubProvider) ListModels(context.Context, string, bool) ([]relayengine.Model, error) {
	return nil, nil
}

func (s *stubProvider) ListLiveModels(context.Context, string) ([]relayengine.Model, error) {
	return nil, nil
}

func (s *stubProvider) ListPublicModels(context.Context, string) ([]relayengine.Model, error) {
	return nil, nil
}

func (s *stubProvider) ModelInfo(context.Context, string) (relayengine.Model, bool, error) {
	return relayengine.Model{}, false, nil
}
func (s *stubProvider) ModelProviders(context.Context) ([]string, error)    { return nil, nil }
func (s *stubProvider) DefaultModel(context.Context, string, string) string { return "" }
func (s *stubProvider) PreferredModel(context.Context, string, relayengine.ModelClass, string) string {
	return ""
}

func (s *stubProvider) PreferredModels(context.Context, string, relayengine.ModelClass, int) []string {
	return nil
}

func (s *stubProvider) ModelClassOf(context.Context, string) relayengine.ModelClass {
	return ModelClassEconomical
}
func (s *stubProvider) ProviderForModel(context.Context, string) string { return "" }
func (s *stubProvider) PrimaryModel(context.Context) string             { return "" }
func (s *stubProvider) ModelNames(context.Context) []string             { return nil }
func (s *stubProvider) StatePaths() relayengine.StatePaths {
	return relayengine.StatePaths{}
}
func (s *stubProvider) DefaultProviderFilter(context.Context) string { return "" }
func (s *stubProvider) Catalog(context.Context) (relayengine.CatalogSnapshot, error) {
	return relayengine.CatalogSnapshot{}, nil
}

func (s *stubProvider) RefreshCatalog(context.Context, string) (relayengine.CatalogSnapshot, error) {
	return relayengine.CatalogSnapshot{}, nil
}

func (s *stubProvider) ApplyCredentials(context.Context, string) (relayengine.CatalogSnapshot, error) {
	return relayengine.CatalogSnapshot{}, nil
}

func (s *stubProvider) SaveCredential(context.Context, string, string) (relayengine.CredentialStatus, error) {
	return relayengine.CredentialStatus{}, nil
}
func (s *stubProvider) RemoveCredential(context.Context, string) error { return nil }
func (s *stubProvider) CredentialStatus(context.Context, string) (relayengine.CredentialStatus, error) {
	return relayengine.CredentialStatus{}, nil
}
func (s *stubProvider) SaveCredentialEnv(context.Context, string, string) error { return nil }
func (s *stubProvider) HasCredentialEnv(context.Context, string) bool           { return false }
func (s *stubProvider) CredentialEnvKeys(string) []string                       { return nil }
func (s *stubProvider) ResolveCredential(context.Context, string) relayengine.CredentialResolution {
	return relayengine.CredentialResolution{}
}

func (s *stubProvider) CredentialProviders(context.Context) []relayengine.CredentialProvider {
	return nil
}

func (s *stubProvider) GatewayDefinitions() []relayengine.Gateway { return nil }

func (s *stubProvider) Gateways(context.Context) []relayengine.Gateway         { return nil }
func (s *stubProvider) GatewayRegion(string) (string, bool)                    { return "", false }
func (s *stubProvider) SetGatewayRegion(context.Context, string, string) error { return nil }
func (s *stubProvider) GatewayForModel(context.Context, string) string         { return "" }
func (s *stubProvider) CanonicalModel(context.Context, string) string          { return "" }
func (s *stubProvider) ApplyGatewayEnvironment(context.Context, string)        {}
func (s *stubProvider) DeploymentRoutingEnabled(*bool) bool                    { return false }
func (s *stubProvider) DeploymentStatus(context.Context, string) (string, error) {
	return "", nil
}

func (s *stubProvider) DeploymentSummary(context.Context, string) (relayengine.DeploymentSummary, error) {
	return relayengine.DeploymentSummary{}, nil
}
func (s *stubProvider) RoutingPreview(context.Context, string) (string, error) { return "", nil }
func (s *stubProvider) CatalogHealth(context.Context) relayengine.CatalogHealth {
	return relayengine.CatalogHealth{}
}

func (s *stubProvider) Preflight(context.Context) relayengine.PreflightReport {
	return relayengine.PreflightReport{}
}

func (s *stubProvider) PreflightWithOptions(context.Context, relayengine.PreflightOptions) relayengine.PreflightReport {
	return relayengine.PreflightReport{}
}

func (s *stubProvider) ActiveSelection(context.Context) relayengine.Route {
	return relayengine.Route{}
}

func (s *stubProvider) EffectiveSelection(context.Context, relayengine.SelectionOptions) relayengine.Selection {
	return relayengine.Selection{}
}
func (s *stubProvider) SetActiveProvider(context.Context, string) error    { return nil }
func (s *stubProvider) SetActiveModel(context.Context, string) error       { return nil }
func (s *stubProvider) SetSelection(context.Context, string, string) error { return nil }
func (s *stubProvider) ClearSelection(context.Context) error               { return nil }
func (s *stubProvider) ProviderStateSecurityStatus() relayengine.ProviderStateSecurity {
	return relayengine.ProviderStateSecurity{}
}

func (s *stubProvider) SupportsNativeCompaction(context.Context, string, string) bool { return false }

func (s *stubProvider) CompactNative(context.Context, relayengine.NativeCompactionRequest) (string, error) {
	return "", nil
}

func TestStubProviderDrivesChatClient(t *testing.T) {
	stub := &stubProvider{resp: &relayengine.GenerateResponse{Content: "from stub", FinishReason: "end_turn"}}
	gw := &Gateway{Generator: stub}
	client := gw.ChatClient()

	got, err := client.Chat(context.Background(), []types.RelayMessage{{Role: "user", Content: "hi"}}, types.ChatOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Content != "from stub" {
		t.Fatalf("stub content not propagated: %+v", got)
	}
	if !client.ManagesResilience() {
		t.Fatal("expected resilience managed flag")
	}
}

var _ Provider = (*stubProvider)(nil)
