// Package gateway is Reflect's single boundary to Relay's provider runtime. It is
// the only package that imports Relay; everything else speaks the reflect-owned
// Provider interface and the internal/types DTOs.
//
// reflect = product face (UX/agent/sessions) · relay = provider engine
// One-way dependency only: relay never imports reflect. See README ecosystems.
package gateway

import (
	"context"
	"log/slog"
	"sync"

	relayengine "github.com/GrayCodeAI/relay/engine"
	"github.com/GrayCodeAI/relay/llm"
)

// Gateway is Reflect's single boundary to the Relay provider runtime. It embeds
// Provider so every engine method is forwarded, and it is the only type that
// constructs one (via New). All other Reflect packages hold a *Gateway or speak
// the Provider interface — never an *relayengine.Engine.
//
// Construction is centralized here: New is the only call to relayengine.New and
// the place Reflect declares its identity to the credential store.
// Gateway is Reflect's single boundary to the Relay provider runtime. It embeds the
// Provider roles so every engine method is forwarded, and it is the only type
// that constructs one (via New). All other Reflect packages hold a *Gateway or speak
// the Provider interface — never an *relayengine.Engine. *Gateway satisfies the
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

// declareReflectIdentity selects the "reflect" OS keychain namespace instead of
// Relay's host-neutral default. Credentials must be configured in this namespace.
// It is idempotent and runs exactly once. Called from New so the
// identity is always declared before any credential read, no matter which New
// path runs first.
var declareReflectIdentity = sync.OnceFunc(func() {
	relayengine.SetSecretStoreServiceName("reflect")
})

// New composes the Relay engine for one effective settings snapshot and wraps it
// as a Provider. It is the single composition root — every relayengine.New call
// in Reflect flows through here.
func New(ctx context.Context, providers []CustomProviderConfig) (*Gateway, error) {
	// Declare reflect's identity to the credential store FIRST, before
	// constructing the engine, so no credential read ever happens under
	// Relay's host-neutral default service name. The OnceFunc makes this
	// safe to call from every construction path.
	declareReflectIdentity()

	gateways := customGatewaysFromSettings(providers)
	eng, err := relayengine.New(relayengine.Options{CustomGateways: gateways})
	if err != nil {
		return nil, err
	}
	// Gap-05: env-gated opt-in wiring of media/STT backends to the router
	// facade. No-op unless REFLECT_MEDIA=1 / REFLECT_STT=1.
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

// BuildCustomGateways maps Reflect's OpenAI-compatible provider config onto
// Relay's CustomGateway spec. Shared by gateway.New, config.relay_engine, and
// engine.session_factory so a new CustomProviderConfig field only needs wiring
// in one place.
func BuildCustomGateways(providers []CustomProviderConfig) []relayengine.CustomGateway {
	gateways := make([]relayengine.CustomGateway, 0, len(providers))
	for _, provider := range providers {
		if provider.Name == "" && provider.BaseURL == "" {
			continue
		}
		gateways = append(gateways, relayengine.CustomGateway{
			ID: provider.Name, BaseURL: provider.BaseURL,
			CredentialEnv: provider.APIKeyEnv, DefaultModel: provider.Model,
		})
	}
	return gateways
}

// customGatewaysFromSettings is the internal alias kept for backward compat.
func customGatewaysFromSettings(providers []CustomProviderConfig) []relayengine.CustomGateway {
	return BuildCustomGateways(providers)
}

// CustomProviderConfig is Reflect's spec for a user-defined OpenAI-compatible
// provider. Kept here (rather than reusing config.CustomProviderConfig) so the
// gateway package does not import config and create an import cycle.
type CustomProviderConfig struct {
	Name      string
	BaseURL   string
	APIKeyEnv string
	Model     string
}

// ModelInfo is Reflect's product-facing view of Relay model metadata.
type ModelInfo struct {
	Name        string  `json:"name"`
	Provider    string  `json:"provider"`
	ContextSize int     `json:"context_size"`
	InputPrice  float64 `json:"input_price_per_million"`
	OutputPrice float64 `json:"output_price_per_million"`
	Description string  `json:"description,omitempty"`
	Recommended bool    `json:"recommended,omitempty"`
}

func fromEngineModel(model relayengine.Model) ModelInfo {
	return ModelInfo{
		Name: model.ID, Provider: model.ProviderID,
		ContextSize: model.ContextWindow,
		InputPrice:  model.InputPricePer1M, OutputPrice: model.OutputPricePer1M,
		Description: model.Description,
	}
}

// ChatClient returns a reflect ChatClient bound to this gateway's Provider.
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

// NewFromEngine wraps an existing *relayengine.Engine as a Gateway. Tests that
// inject an Relay SecretStore (e.g. compaction-support detection) use it so the
// rest of Reflect still speaks the Gateway boundary.
func NewFromEngine(eng *relayengine.Engine) *Gateway {
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
// These delegate Relay reads to one shared default gateway so reflect-owned
// policy packages (routing, config) never import Relay themselves. relay's
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

// PreferredModel returns Relay's tier-preferred model for a provider.
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

// --- reflect-owned mirror of Relay's ModelClass tier enum -------------------
// Kept here (rather than importing neutral constants) so the boundary stays
// one-way; values match relayengine.ModelClass.
type ModelClass = relayengine.ModelClass

const (
	ModelClassEconomical = llm.ModelClassEconomical
	ModelClassBalanced   = llm.ModelClassBalanced
	ModelClassPremium    = llm.ModelClassPremium
	CheckFail            = relayengine.CheckFail
)

// NormalizeProviderID canonicalizes a host-facing provider/gateway id.
func NormalizeProviderID(id string) string {
	return relayengine.NormalizeProviderID(id)
}

// --- Relay report/type re-exports config internals consume ----------------
// These alias Relay types that a few config-only report paths return. They
// live in gateway (the single Relay importer) rather than config.

type (
	PreflightReport         = relayengine.PreflightReport
	PreflightOptions        = relayengine.PreflightOptions
	ProviderStateSecurity   = relayengine.ProviderStateSecurity
	DeploymentSummary       = relayengine.DeploymentSummary
	CredentialStorageReport = relayengine.CredentialStorageReport
	CredentialStatus        = relayengine.CredentialStatus
	CredentialResolution    = relayengine.CredentialResolution
	CredentialProvider      = relayengine.CredentialProvider
	GatewayDefs             = relayengine.Gateway
	CatalogSnapshot         = relayengine.CatalogSnapshot
	Model                   = relayengine.Model
	StatePaths              = relayengine.StatePaths
	SelectionOptions        = relayengine.SelectionOptions
	Selection               = relayengine.Selection
	NativeCompactionRequest = relayengine.NativeCompactionRequest
)

// Package-level Relay helpers that config delegates to (gateway stays the only importer).

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
	return relayengine.FormatPreflight(report)
}

func IsCatalogCacheRequired(err error) bool {
	return relayengine.IsCatalogCacheRequired(err)
}

// RegisteredProviderCount exposes Relay's first-class provider count through
// Reflect's single provider-runtime boundary. The count derives from Relay's
// provider registry, so adding a provider in Relay never requires a Reflect edit.
func RegisteredProviderCount() int {
	return relayengine.RegisteredGatewayCount()
}

func SecretStoreName() string { return relayengine.SecretStoreName() }

func CredentialStorage(ctx context.Context) CredentialStorageReport {
	return relayengine.CredentialStorage(ctx)
}

func CredentialGuidance(providerID, secret string) string {
	return relayengine.CredentialGuidance(providerID, secret)
}

func FormatSetupError(providerID string, err error) string {
	return relayengine.FormatSetupError(providerID, err)
}

// ParseInlineToolCalls extracts inline tool-call markup from model output.
func ParseInlineToolCalls(content string) (string, []relayengine.ToolCall) {
	return relayengine.ParseInlineToolCalls(content)
}

func DefaultThinkingDisabled(providerID string) bool {
	return relayengine.DefaultThinkingDisabled(providerID)
}

func ThinkingToggleSupported(providerID string) bool {
	return relayengine.ThinkingToggleSupported(providerID)
}

func (g *Gateway) DefaultThinkingDisabled(providerID string) bool {
	return relayengine.DefaultThinkingDisabled(providerID)
}

func (g *Gateway) ThinkingToggleSupported(providerID string) bool {
	return relayengine.ThinkingToggleSupported(providerID)
}

// --- Test fixtures -----------------------------------------------------
// Re-exported from engine so reflect tests inject credential fixtures through the
// single gateway+engine boundary. These are thin aliases only.

// SetDefaultStore replaces the process-wide credential store (for tests).
var SetDefaultStore = relayengine.SetDefaultStore

// DefaultStore returns the process-wide credential store (for tests).
var DefaultStore = relayengine.DefaultStore

// MapStore is the in-memory credential store for tests (alias).
type MapStore = relayengine.MapStore

// AccountForEnv returns the keychain account name for an env var.
func AccountForEnv(envVar string) string { return relayengine.AccountForEnv(envVar) }

// HasSecret reports whether a secret exists for an env var (for tests).
func HasSecret(ctx context.Context, envKey string) bool {
	return relayengine.HasSecret(ctx, envKey)
}
