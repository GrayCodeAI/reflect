// Package gateway is Decoder's single boundary to Mux's provider runtime. It is
// the only package that imports Mux; everything else speaks the decoder-owned
// Provider interface and the internal/types DTOs.
//
// decoder = product face (UX/agent/sessions) · mux = provider engine
// One-way dependency only: mux never imports decoder. See README ecosystems.
package gateway

import (
	"context"
	"log/slog"
	"sync"

	muxengine "github.com/GrayCodeAI/mux/engine"
	"github.com/GrayCodeAI/mux/llm"
)

// Gateway is Decoder's single boundary to the Mux provider runtime. It embeds
// Provider so every engine method is forwarded, and it is the only type that
// constructs one (via New). All other Decoder packages hold a *Gateway or speak
// the Provider interface — never an *muxengine.Engine.
//
// Construction is centralized here: New is the only call to muxengine.New and
// the place Decoder declares its identity to the credential store.
// Gateway is Decoder's single boundary to the Mux provider runtime. It embeds the
// Provider roles so every engine method is forwarded, and it is the only type
// that constructs one (via New). All other Decoder packages hold a *Gateway or speak
// the Provider interface — never an *muxengine.Engine. *Gateway satisfies the
// composite Provider interface.
type Gateway struct {
	Generator
	NativeCompactor
	ModelCatalog
	CredentialManager
	SelectionManager
	GatewayInspector
	CatalogMaintenance
}

// declareDecoderIdentity selects the "decoder" OS keychain namespace instead of
// Mux's host-neutral default. Credentials must be configured in this namespace.
// It is idempotent and runs exactly once. Called from New so the
// identity is always declared before any credential read, no matter which New
// path runs first.
var declareDecoderIdentity = sync.OnceFunc(func() {
	muxengine.SetSecretStoreServiceName("decoder")
})

// New composes the Mux engine for one effective settings snapshot and wraps it
// as a Provider. It is the single composition root — every muxengine.New call
// in Decoder flows through here.
func New(ctx context.Context, providers []CustomProviderConfig) (*Gateway, error) {
	// Declare decoder's identity to the credential store FIRST, before
	// constructing the engine, so no credential read ever happens under
	// Mux's host-neutral default service name. The OnceFunc makes this
	// safe to call from every construction path.
	declareDecoderIdentity()

	gateways := customGatewaysFromSettings(providers)
	eng, err := muxengine.New(muxengine.Options{CustomGateways: gateways})
	if err != nil {
		return nil, err
	}
	// Gap-05: env-gated opt-in wiring of media/STT backends to the router
	// facade. No-op unless DECODER_MEDIA=1 / DECODER_STT=1.
	wireOptionalBackends(eng)
	p := newEngineProvider(eng)
	return &Gateway{
		Generator:          p,
		NativeCompactor:    p,
		ModelCatalog:       p,
		CredentialManager:  p,
		SelectionManager:   p,
		GatewayInspector:   p,
		CatalogMaintenance: p,
	}, nil
}

// BuildCustomGateways maps Decoder's OpenAI-compatible provider config onto
// Mux's CustomGateway spec. Shared by gateway.New, config.mux_engine, and
// engine.session_factory so a new CustomProviderConfig field only needs wiring
// in one place.
func BuildCustomGateways(providers []CustomProviderConfig) []muxengine.CustomGateway {
	gateways := make([]muxengine.CustomGateway, 0, len(providers))
	for _, provider := range providers {
		if provider.Name == "" && provider.BaseURL == "" {
			continue
		}
		gateways = append(gateways, muxengine.CustomGateway{
			ID: provider.Name, BaseURL: provider.BaseURL,
			CredentialEnv: provider.APIKeyEnv, DefaultModel: provider.Model,
		})
	}
	return gateways
}

// customGatewaysFromSettings is the internal alias kept for backward compat.
func customGatewaysFromSettings(providers []CustomProviderConfig) []muxengine.CustomGateway {
	return BuildCustomGateways(providers)
}

// CustomProviderConfig is Decoder's spec for a user-defined OpenAI-compatible
// provider. Kept here (rather than reusing config.CustomProviderConfig) so the
// gateway package does not import config and create an import cycle.
type CustomProviderConfig struct {
	Name      string
	BaseURL   string
	APIKeyEnv string
	Model     string
}

// ModelInfo is Decoder's product-facing view of Mux model metadata.
type ModelInfo struct {
	Name        string  `json:"name"`
	Provider    string  `json:"provider"`
	ContextSize int     `json:"context_size"`
	InputPrice  float64 `json:"input_price_per_million"`
	OutputPrice float64 `json:"output_price_per_million"`
	Description string  `json:"description,omitempty"`
	Recommended bool    `json:"recommended,omitempty"`
}

func fromEngineModel(model muxengine.Model) ModelInfo {
	return ModelInfo{
		Name: model.ID, Provider: model.ProviderID,
		ContextSize: model.ContextWindow,
		InputPrice:  model.InputPricePer1M, OutputPrice: model.OutputPricePer1M,
		Description: model.Description,
	}
}

// ChatClient returns a decoder ChatClient bound to this gateway's Provider.
func (g *Gateway) ChatClient() *translateProvider {
	return newChatClientProvider(g)
}

// MustSelectProvider returns the Provider, or nil if the gateway is unset.
func (g *Gateway) MustSelectProvider() Provider {
	if g == nil {
		return nil
	}
	return g
}

// NewFromEngine wraps an existing *muxengine.Engine as a Gateway. Tests that
// inject an Mux SecretStore (e.g. compaction-support detection) use it so the
// rest of Decoder still speaks the Gateway boundary.
func NewFromEngine(eng *muxengine.Engine) *Gateway {
	if eng == nil {
		return nil
	}
	p := newEngineProvider(eng)
	return &Gateway{
		Generator:          p,
		NativeCompactor:    p,
		ModelCatalog:       p,
		CredentialManager:  p,
		SelectionManager:   p,
		GatewayInspector:   p,
		CatalogMaintenance: p,
	}
}

// --- Stateless package-level lookups -------------------------------------
// These delegate Mux reads to one shared default gateway so decoder-owned
// policy packages (routing, config) never import Mux themselves. mux's
// Engine reloads its catalog and provider config from disk on every method
// call, so a single long-lived gateway returns identical freshness to
// constructing one per call — this just avoids redundant construction. It
// mirrors the sync.Once singleton the old routing code used.
//
// The ctx parameter is retained (though unused here) to keep this helper's
// signature stable across its 11 call sites; ctx still flows through to every
// data call via the helpers below.

var (
	defaultGatewayMu  sync.Mutex
	defaultGatewayVal *Gateway
	// newGatewayFn is the constructor used by defaultGateway. Indirect so
	// tests can inject failure and assert the retry behavior (H8).
	newGatewayFn = func(ctx context.Context) (*Gateway, error) {
		return New(ctx, nil)
	}
)

func defaultGateway(ctx context.Context) *Gateway {
	// Fast path: already constructed.
	defaultGatewayMu.Lock()
	defer defaultGatewayMu.Unlock()
	if defaultGatewayVal != nil {
		return defaultGatewayVal
	}
	// Construct on first successful use. If New fails, log the error and
	// return nil; a later call retries instead of being permanently nil
	// (the sync.Once footgun that discarded the error, H8).
	g, err := newGatewayFn(context.Background())
	if err != nil {
		slog.Error("gateway initialization failed", "error", err)
		return nil
	}
	defaultGatewayVal = g
	return defaultGatewayVal
}

// ModelInfoLookup returns a model by id or alias, or false if unknown.
func ModelInfoLookup(ctx context.Context, name string) (ModelInfo, bool) {
	g := defaultGateway(ctx)
	if g == nil {
		return ModelInfo{}, false
	}
	model, ok, err := g.ModelInfo(ctx, name)
	if err != nil || !ok {
		return ModelInfo{}, false
	}
	return fromEngineModel(model), true
}

// ModelsByProvider returns every model served by a provider/gateway.
func ModelsByProvider(ctx context.Context, provider string) ([]ModelInfo, error) {
	g := defaultGateway(ctx)
	if g == nil {
		return nil, nil
	}
	models, err := g.ListModels(ctx, provider, false)
	if err != nil {
		return nil, err
	}
	out := make([]ModelInfo, 0, len(models))
	for _, model := range models {
		out = append(out, fromEngineModel(model))
	}
	return out, nil
}

// RecommendedModel returns the catalog default for a provider, flagged as
// recommended.
func RecommendedModel(ctx context.Context, provider string) (ModelInfo, bool) {
	g := defaultGateway(ctx)
	if g == nil {
		return ModelInfo{}, false
	}
	name := g.DefaultModel(ctx, provider, "")
	if name == "" {
		return ModelInfo{}, false
	}
	info, ok := ModelInfoLookup(ctx, name)
	if ok {
		info.Recommended = true
	}
	return info, ok
}

// DefaultModel returns the catalog default model name for a provider.
func DefaultModel(ctx context.Context, provider string) string {
	g := defaultGateway(ctx)
	if g == nil {
		return ""
	}
	return g.DefaultModel(ctx, provider, "")
}

// AllProviders returns the distinct set of providers/gateways in the catalog.
func AllProviders(ctx context.Context) ([]string, error) {
	g := defaultGateway(ctx)
	if g == nil {
		return nil, nil
	}
	return g.ModelProviders(ctx)
}

// ProviderForModel resolves which provider owns a model name.
func ProviderForModel(ctx context.Context, modelName string) string {
	g := defaultGateway(ctx)
	if g == nil {
		return ""
	}
	return g.ProviderForModel(ctx, modelName)
}

// PreferredModel returns Mux's tier-preferred model for a provider.
func PreferredModel(ctx context.Context, provider string, class ModelClass, fallback string) string {
	g := defaultGateway(ctx)
	if g == nil {
		return fallback
	}
	return g.PreferredModel(ctx, provider, class, fallback)
}

// PreferredModels returns up to limit tier-preferred models for a provider.
func PreferredModels(ctx context.Context, primaryProvider string, class ModelClass, limit int) []string {
	g := defaultGateway(ctx)
	if g == nil {
		return nil
	}
	return g.PreferredModels(ctx, primaryProvider, class, limit)
}

// ModelClassOf returns the cost tier of a model.
func ModelClassOf(ctx context.Context, modelID string) ModelClass {
	g := defaultGateway(ctx)
	if g == nil {
		return ModelClassBalanced
	}
	return g.ModelClassOf(ctx, modelID)
}

// PrimaryModel returns the catalog-wide primary model.
func PrimaryModel(ctx context.Context) string {
	g := defaultGateway(ctx)
	if g == nil {
		return ""
	}
	return g.PrimaryModel(ctx)
}

// ModelNames returns all model names known to the catalog.
func ModelNames(ctx context.Context) []string {
	g := defaultGateway(ctx)
	if g == nil {
		return nil
	}
	return g.ModelNames(ctx)
}

// --- decoder-owned mirror of Mux's ModelClass tier enum -------------------
// Kept here (rather than importing neutral constants) so the boundary stays
// one-way; values match muxengine.ModelClass.
type ModelClass = muxengine.ModelClass

const (
	ModelClassEconomical = llm.ModelClassEconomical
	ModelClassBalanced   = llm.ModelClassBalanced
	ModelClassPremium    = llm.ModelClassPremium
	CheckFail            = muxengine.CheckFail
)

// NormalizeProviderID canonicalizes a host-facing provider/gateway id.
func NormalizeProviderID(id string) string {
	return muxengine.NormalizeProviderID(id)
}

// --- Mux report/type re-exports config internals consume ----------------
// These alias Mux types that a few config-only report paths return. They
// live in gateway (the single Mux importer) rather than config.

type (
	PreflightReport         = muxengine.PreflightReport
	PreflightOptions        = muxengine.PreflightOptions
	ProviderStateSecurity   = muxengine.ProviderStateSecurity
	DeploymentSummary       = muxengine.DeploymentSummary
	CredentialStorageReport = muxengine.CredentialStorageReport
	CredentialStatus        = muxengine.CredentialStatus
	CredentialResolution    = muxengine.CredentialResolution
	CredentialProvider      = muxengine.CredentialProvider
	GatewayDefs             = muxengine.Gateway
	CatalogSnapshot         = muxengine.CatalogSnapshot
	Model                   = muxengine.Model
	StatePaths              = muxengine.StatePaths
	SelectionOptions        = muxengine.SelectionOptions
	Selection               = muxengine.Selection
	NativeCompactionRequest = muxengine.NativeCompactionRequest
)

// Package-level Mux helpers that config delegates to (gateway stays the only importer).

func PreflightReportWithOptions(ctx context.Context, opts PreflightOptions) PreflightReport {
	return PreflightWithProviders(ctx, nil, opts)
}

// PreflightWithProviders runs preflight against a gateway built from an
// explicit provider list (a settings snapshot's custom gateways), so concurrent
// commands can isolate custom-provider state.
func PreflightWithProviders(ctx context.Context, providers []CustomProviderConfig, opts PreflightOptions) PreflightReport {
	g, err := New(ctx, providers)
	if err != nil {
		return PreflightReport{}
	}
	return g.PreflightWithOptions(ctx, opts)
}

func FormatPreflight(report PreflightReport) string {
	return muxengine.FormatPreflight(report)
}

func IsCatalogCacheRequired(err error) bool {
	return muxengine.IsCatalogCacheRequired(err)
}

// RegisteredProviderCount exposes Mux's first-class provider count through
// Decoder's single provider-runtime boundary. The count derives from Mux's
// provider registry, so adding a provider in Mux never requires a Decoder edit.
func RegisteredProviderCount() int {
	return muxengine.RegisteredGatewayCount()
}

func SecretStoreName() string { return muxengine.SecretStoreName() }

func CredentialStorage(ctx context.Context) CredentialStorageReport {
	return muxengine.CredentialStorage(ctx)
}

func CredentialGuidance(providerID, secret string) string {
	return muxengine.CredentialGuidance(providerID, secret)
}

func FormatSetupError(providerID string, err error) string {
	return muxengine.FormatSetupError(providerID, err)
}

// ParseInlineToolCalls extracts inline tool-call markup from model output.
func ParseInlineToolCalls(content string) (string, []muxengine.ToolCall) {
	return muxengine.ParseInlineToolCalls(content)
}

func DefaultThinkingDisabled(providerID string) bool {
	return muxengine.DefaultThinkingDisabled(providerID)
}

func ThinkingToggleSupported(providerID string) bool {
	return muxengine.ThinkingToggleSupported(providerID)
}

func (g *Gateway) DefaultThinkingDisabled(providerID string) bool {
	return muxengine.DefaultThinkingDisabled(providerID)
}

func (g *Gateway) ThinkingToggleSupported(providerID string) bool {
	return muxengine.ThinkingToggleSupported(providerID)
}

// --- Test fixtures -----------------------------------------------------
// Re-exported from engine so decoder tests inject credential fixtures through the
// single gateway+engine boundary. These are thin aliases only.

// SetDefaultStore replaces the process-wide credential store (for tests).
var SetDefaultStore = muxengine.SetDefaultStore

// DefaultStore returns the process-wide credential store (for tests).
var DefaultStore = muxengine.DefaultStore

// MapStore is the in-memory credential store for tests (alias).
type MapStore = muxengine.MapStore

// AccountForEnv returns the keychain account name for an env var.
func AccountForEnv(envVar string) string { return muxengine.AccountForEnv(envVar) }

// HasSecret reports whether a secret exists for an env var (for tests).
func HasSecret(ctx context.Context, envKey string) bool {
	return muxengine.HasSecret(ctx, envKey)
}
