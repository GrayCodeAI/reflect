package gateway

import (
	"context"
	"testing"

	"github.com/GrayCodeAI/decoder/internal/types"
	muxengine "github.com/GrayCodeAI/mux/engine"
)

// stubProvider proves the Provider interface is swappable: a test can inject a
// deterministic fake and drive the ChatClient adapter without constructing any
// Mux engine. This is the swappable-Engine guarantee Phase 3 delivers.
type stubProvider struct {
	resp *muxengine.GenerateResponse
	err  error
}

func (s *stubProvider) Resolve(context.Context, muxengine.SelectionRequest) (muxengine.Route, error) {
	return muxengine.Route{}, nil
}

func (s *stubProvider) Generate(context.Context, muxengine.GenerateRequest) (*muxengine.GenerateResponse, error) {
	return s.resp, s.err
}

func (s *stubProvider) Stream(context.Context, muxengine.GenerateRequest) (muxengine.EventStreamer, error) {
	return nil, nil
}

func (s *stubProvider) ListModels(context.Context, string, bool) ([]muxengine.Model, error) {
	return nil, nil
}

func (s *stubProvider) ListLiveModels(context.Context, string) ([]muxengine.Model, error) {
	return nil, nil
}

func (s *stubProvider) ListPublicModels(context.Context, string) ([]muxengine.Model, error) {
	return nil, nil
}

func (s *stubProvider) ModelInfo(context.Context, string) (muxengine.Model, bool, error) {
	return muxengine.Model{}, false, nil
}
func (s *stubProvider) ModelProviders(context.Context) ([]string, error)    { return nil, nil }
func (s *stubProvider) DefaultModel(context.Context, string, string) string { return "" }
func (s *stubProvider) PreferredModel(context.Context, string, muxengine.ModelClass, string) string {
	return ""
}

func (s *stubProvider) PreferredModels(context.Context, string, muxengine.ModelClass, int) []string {
	return nil
}

func (s *stubProvider) ModelClassOf(context.Context, string) muxengine.ModelClass {
	return ModelClassEconomical
}
func (s *stubProvider) ProviderForModel(context.Context, string) string { return "" }
func (s *stubProvider) PrimaryModel(context.Context) string             { return "" }
func (s *stubProvider) ModelNames(context.Context) []string             { return nil }
func (s *stubProvider) StatePaths() muxengine.StatePaths {
	return muxengine.StatePaths{}
}
func (s *stubProvider) DefaultProviderFilter(context.Context) string { return "" }
func (s *stubProvider) Catalog(context.Context) (muxengine.CatalogSnapshot, error) {
	return muxengine.CatalogSnapshot{}, nil
}

func (s *stubProvider) RefreshCatalog(context.Context, string) (muxengine.CatalogSnapshot, error) {
	return muxengine.CatalogSnapshot{}, nil
}

func (s *stubProvider) ApplyCredentials(context.Context, string) (muxengine.CatalogSnapshot, error) {
	return muxengine.CatalogSnapshot{}, nil
}

func (s *stubProvider) SaveCredential(context.Context, string, string) (muxengine.CredentialStatus, error) {
	return muxengine.CredentialStatus{}, nil
}
func (s *stubProvider) RemoveCredential(context.Context, string) error { return nil }
func (s *stubProvider) CredentialStatus(context.Context, string) (muxengine.CredentialStatus, error) {
	return muxengine.CredentialStatus{}, nil
}
func (s *stubProvider) SaveCredentialEnv(context.Context, string, string) error { return nil }
func (s *stubProvider) HasCredentialEnv(context.Context, string) bool           { return false }
func (s *stubProvider) CredentialEnvKeys(string) []string                       { return nil }
func (s *stubProvider) ResolveCredential(context.Context, string) muxengine.CredentialResolution {
	return muxengine.CredentialResolution{}
}

func (s *stubProvider) CredentialProviders(context.Context) []muxengine.CredentialProvider {
	return nil
}

func (s *stubProvider) GatewayDefinitions() []muxengine.Gateway { return nil }

func (s *stubProvider) Gateways(context.Context) []muxengine.Gateway           { return nil }
func (s *stubProvider) GatewayRegion(string) (string, bool)                    { return "", false }
func (s *stubProvider) SetGatewayRegion(context.Context, string, string) error { return nil }
func (s *stubProvider) GatewayForModel(context.Context, string) string         { return "" }
func (s *stubProvider) CanonicalModel(context.Context, string) string          { return "" }
func (s *stubProvider) ApplyGatewayEnvironment(context.Context, string)        {}
func (s *stubProvider) DeploymentRoutingEnabled(*bool) bool                    { return false }
func (s *stubProvider) DeploymentStatus(context.Context, string) (string, error) {
	return "", nil
}

func (s *stubProvider) DeploymentSummary(context.Context, string) (muxengine.DeploymentSummary, error) {
	return muxengine.DeploymentSummary{}, nil
}
func (s *stubProvider) RoutingPreview(context.Context, string) (string, error) { return "", nil }
func (s *stubProvider) CatalogHealth(context.Context) muxengine.CatalogHealth {
	return muxengine.CatalogHealth{}
}

func (s *stubProvider) Preflight(context.Context) muxengine.PreflightReport {
	return muxengine.PreflightReport{}
}

func (s *stubProvider) PreflightWithOptions(context.Context, muxengine.PreflightOptions) muxengine.PreflightReport {
	return muxengine.PreflightReport{}
}

func (s *stubProvider) ActiveSelection(context.Context) muxengine.Route {
	return muxengine.Route{}
}

func (s *stubProvider) EffectiveSelection(context.Context, muxengine.SelectionOptions) muxengine.Selection {
	return muxengine.Selection{}
}
func (s *stubProvider) SetActiveProvider(context.Context, string) error    { return nil }
func (s *stubProvider) SetActiveModel(context.Context, string) error       { return nil }
func (s *stubProvider) SetSelection(context.Context, string, string) error { return nil }
func (s *stubProvider) ClearSelection(context.Context) error               { return nil }
func (s *stubProvider) ProviderStateSecurityStatus() muxengine.ProviderStateSecurity {
	return muxengine.ProviderStateSecurity{}
}

func (s *stubProvider) SupportsNativeCompaction(context.Context, string, string) bool { return false }

func (s *stubProvider) CompactNative(context.Context, muxengine.NativeCompactionRequest) (string, error) {
	return "", nil
}

func TestStubProviderDrivesChatClient(t *testing.T) {
	stub := &stubProvider{resp: &muxengine.GenerateResponse{Content: "from stub", FinishReason: "end_turn"}}
	gw := &Gateway{Generator: stub}
	client := gw.ChatClient()

	got, err := client.Chat(context.Background(), []types.MuxMessage{{Role: "user", Content: "hi"}}, types.ChatOptions{})
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
