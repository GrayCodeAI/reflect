package config

import (
	"context"
	"fmt"
	"strings"

	"github.com/GrayCodeAI/decoder/internal/theme"
	"github.com/GrayCodeAI/decoder/internal/token"
)

// EcosystemReport is the structured view of the ecosystem panel.
type EcosystemReport struct {
	Mux   EcosystemMux `json:"mux"`
	Token EngineToken  `json:"token"`
}

type EcosystemMux struct {
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

	// mux
	cat := CatalogHealthReport(ctx)
	r.Mux.CatalogExists = cat.Exists
	r.Mux.ModelCount = cat.Models
	pre := EnginePreflightReport(ctx)
	r.Mux.Ready = pre.Ready
	if strings.TrimSpace(provider) != "" && provider != "auto" {
		r.Mux.Provider = provider
	}
	if dep, err := EngineDeploymentSummary(ctx, model); err == nil {
		r.Mux.RoutingSource = dep.RoutingSource
		r.Mux.RoutingStages = dep.RoutingStages
	}

	// embedded token engine
	r.Token.Embedded = token.ShrikeAvailable()
	r.Token.SampleTokens = token.CountTokensFast("decoder context compression pipeline")

	return r
}

// FormatEcosystemPanel summarizes mux and token-engine integration for doctor and status output.
func FormatEcosystemPanel(ctx context.Context, provider, model string) string {
	var b strings.Builder
	b.WriteString(theme.Tint("Ecosystem (mux · token engine):", theme.ReportInfo) + "\n")

	// mux — LLM provider layer
	cat := CatalogHealthReport(ctx)
	muxLine := "  " + theme.Tint("mux:", theme.ReportMuted) + " "
	if cat.Exists {
		muxLine += theme.Tint(fmt.Sprintf("catalog %d models", cat.Models), theme.ReportInfo)
	} else {
		muxLine += theme.Tint("catalog missing (run decoder models refresh)", theme.ReportWarn)
	}
	pre := EnginePreflightReport(ctx)
	if pre.Ready {
		muxLine += " · " + theme.Tint("locally ready", theme.ReportSuccess)
	} else {
		muxLine += " · " + theme.Tint("setup incomplete", theme.ReportWarn)
	}
	if strings.TrimSpace(provider) != "" && provider != "auto" {
		muxLine += " · " + theme.Tint("provider "+provider, theme.ReportInfo)
	}
	if dep, err := EngineDeploymentSummary(ctx, model); err == nil {
		if dep.RoutingStages > 0 {
			muxLine += " · " + theme.Tint(fmt.Sprintf("routing %s (%d stages)", dep.RoutingSource, dep.RoutingStages), theme.ReportInfo)
		} else {
			muxLine += " · " + theme.Tint("routing "+dep.RoutingSource, theme.ReportInfo)
		}
	}
	b.WriteString(muxLine + "\n")

	// Embedded token engine — token counting and context compression.
	sample := token.CountTokensFast("decoder context compression pipeline")
	if token.ShrikeAvailable() {
		b.WriteString("  " + theme.Tint("token:", theme.ReportMuted) + " " + theme.Tint("embedded", theme.ReportInfo) + " · " + theme.Tint("token/compress pipeline OK", theme.ReportSuccess) + fmt.Sprintf(" (sample=%d tokens)", sample) + "\n")
	} else {
		b.WriteString("  " + theme.Tint("token:", theme.ReportMuted) + " " + theme.Tint("unavailable", theme.ReportWarn) + " · " + theme.Tint("token/compress pipeline not linked", theme.ReportWarn) + fmt.Sprintf(" (sample=%d tokens)", sample) + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}
