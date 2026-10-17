package config

import (
	"context"
	"fmt"
	"strings"

	"github.com/GrayCodeAI/reflect/internal/theme"
	"github.com/GrayCodeAI/reflect/internal/token"
)

// EcosystemReport is the structured view of the ecosystem panel.
type EcosystemReport struct {
	Relay EcosystemRelay `json:"relay"`
	Token EngineToken    `json:"token"`
}

type EcosystemRelay struct {
	CatalogExists bool   `json:"catalog_exists"`
	ModelCount    int    `json:"model_count,omitempty"`
	Ready         bool   `json:"ready"`
	Provider      string `json:"provider,omitempty"`
	RoutingSource string `json:"routing_source,omitempty"`
	RoutingStages int    `json:"routing_stages,omitempty"`
}

type EngineToken struct {
	Embedded     bool `json:"embedded"`
	SampleTokens int  `json:"sample_tokens"`
}

// BuildEcosystemReport returns a structured ecosystem report.
func BuildEcosystemReport(ctx context.Context, provider, model string) EcosystemReport {
	var r EcosystemReport

	// relay
	cat := CatalogHealthReport(ctx)
	r.Relay.CatalogExists = cat.Exists
	r.Relay.ModelCount = cat.Models
	pre := EnginePreflightReport(ctx)
	r.Relay.Ready = pre.Ready
	if strings.TrimSpace(provider) != "" && provider != "auto" {
		r.Relay.Provider = provider
	}
	if dep, err := EngineDeploymentSummary(ctx, model); err == nil {
		r.Relay.RoutingSource = dep.RoutingSource
		r.Relay.RoutingStages = dep.RoutingStages
	}

	// embedded token engine
	r.Token.Embedded = token.ShrikeAvailable()
	r.Token.SampleTokens = token.CountTokensFast("reflect context compression pipeline")

	return r
}

// FormatEcosystemPanel summarizes relay and token-engine integration for doctor and status output.
func FormatEcosystemPanel(ctx context.Context, provider, model string) string {
	var b strings.Builder
	b.WriteString(theme.Tint("Ecosystem (relay · token engine):", theme.ReportInfo) + "\n")

	// relay — LLM provider layer
	cat := CatalogHealthReport(ctx)
	relayLine := "  " + theme.Tint("relay:", theme.ReportMuted) + " "
	if cat.Exists {
		relayLine += theme.Tint(fmt.Sprintf("catalog %d models", cat.Models), theme.ReportInfo)
	} else {
		relayLine += theme.Tint("catalog missing (run reflect models refresh)", theme.ReportWarn)
	}
	pre := EnginePreflightReport(ctx)
	if pre.Ready {
		relayLine += " · " + theme.Tint("locally ready", theme.ReportSuccess)
	} else {
		relayLine += " · " + theme.Tint("setup incomplete", theme.ReportWarn)
	}
	if strings.TrimSpace(provider) != "" && provider != "auto" {
		relayLine += " · " + theme.Tint("provider "+provider, theme.ReportInfo)
	}
	if dep, err := EngineDeploymentSummary(ctx, model); err == nil {
		if dep.RoutingStages > 0 {
			relayLine += " · " + theme.Tint(fmt.Sprintf("routing %s (%d stages)", dep.RoutingSource, dep.RoutingStages), theme.ReportInfo)
		} else {
			relayLine += " · " + theme.Tint("routing "+dep.RoutingSource, theme.ReportInfo)
		}
	}
	b.WriteString(relayLine + "\n")

	// Embedded token engine — token counting and context compression.
	sample := token.CountTokensFast("reflect context compression pipeline")
	if token.ShrikeAvailable() {
		b.WriteString("  " + theme.Tint("token:", theme.ReportMuted) + " " + theme.Tint("embedded", theme.ReportInfo) + " · " + theme.Tint("token/compress pipeline OK", theme.ReportSuccess) + fmt.Sprintf(" (sample=%d tokens)", sample) + "\n")
	} else {
		b.WriteString("  " + theme.Tint("token:", theme.ReportMuted) + " " + theme.Tint("unavailable", theme.ReportWarn) + " · " + theme.Tint("token/compress pipeline not linked", theme.ReportWarn) + fmt.Sprintf(" (sample=%d tokens)", sample) + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}
