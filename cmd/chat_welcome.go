package cmd

import (
	"context"
	"fmt"
	"image/color"
	"os"
	"sort"
	"strings"

	decoderconfig "github.com/GrayCodeAI/decoder/internal/config"
	"github.com/GrayCodeAI/decoder/internal/engine"
	"github.com/GrayCodeAI/decoder/internal/session"
	"github.com/GrayCodeAI/decoder/internal/tool"
)

type welcomeStatusSnapshot struct {
	setup    decoderconfig.SetupState
	agentsOK bool
}

func loadWelcomeStatusSnapshot() welcomeStatusSnapshot {
	ctx := context.Background()
	return welcomeStatusSnapshot{
		setup:    decoderconfig.EvaluateSetupCached(ctx),
		agentsOK: decoderconfig.LoadAgentsMD() != "",
	}
}

func (m *chatModel) refreshWelcomeStatusSnapshot() {
	snapshot := loadWelcomeStatusSnapshot()
	m.welcomeSetupState = snapshot.setup
	m.welcomeAgentsOK = snapshot.agentsOK
}

func (m chatModel) welcomeStatusSnapshot() welcomeStatusSnapshot {
	return welcomeStatusSnapshot{
		setup:    m.welcomeSetupState,
		agentsOK: m.welcomeAgentsOK,
	}
}

func (m *chatModel) rebuildWelcomeCache(opts ...any) {
	frame := m.eyeFrame
	if len(opts) > 0 {
		switch v := opts[0].(type) {
		case int:
			frame = v
		case bool:
			if v {
				frame = 2
			} else {
				frame = 0
			}
		}
	}
	width := m.width
	if width <= 0 {
		width = 80
	}
	height := m.height
	if height <= 0 {
		height = 24
	}
	skillsCount := 0
	if m.pluginRuntime != nil {
		skillsCount = len(m.pluginRuntime.SmartSkills)
	}
	m.welcomeCache = buildWelcomeMessageWithSnapshotAndMascot(m.session, m.sessionID, m.registry, nil, m.settings, skillsCount, connectedMCPCount(m.registry), frame, width, height, m.welcomeStatusSnapshot(), m.lastCommand, m.mascotEnabled)
}

// buildWelcomeMessage renders the branded inline DECODER welcome block.
func buildWelcomeMessage(sess *engine.Session, sessionID string, registry *tool.Registry, saved *session.Session, settings decoderconfig.Settings, skillsCount int, blinkClosed bool, width, height int) string {
	frame := 0
	if blinkClosed {
		frame = 2
	}
	return buildWelcomeMessageWithSnapshot(sess, sessionID, registry, saved, settings, skillsCount, connectedMCPCount(registry), frame, width, height, loadWelcomeStatusSnapshot(), "")
}

func buildWelcomeMessageWithSnapshot(sess *engine.Session, sessionID string, registry *tool.Registry, saved *session.Session, settings decoderconfig.Settings, skillsCount, mcpCount int, eyeFrame int, width, height int, snapshot welcomeStatusSnapshot, lastCommand string) string {
	return buildWelcomeMessageWithSnapshotAndMascot(sess, sessionID, registry, saved, settings, skillsCount, mcpCount, eyeFrame, width, height, snapshot, lastCommand, false)
}

func buildWelcomeMessageWithSnapshotAndMascot(sess *engine.Session, sessionID string, registry *tool.Registry, saved *session.Session, settings decoderconfig.Settings, skillsCount, mcpCount int, eyeFrame int, width, height int, snapshot welcomeStatusSnapshot, lastCommand string, useMascot bool) string {
	if width <= 0 {
		width = 80
	}
	// A small, left-aligned block keeps the empty state readable at every size.
	indent := 2
	if width < 40 {
		indent = 0
	}
	line := func(text string) string {
		return strings.Repeat(" ", indent) + clipFooterLine(text, width-indent) + "\n"
	}
	version := DisplayVersion()
	if version != "" && !strings.HasPrefix(strings.ToLower(version), "v") {
		version = "v" + version
	}
	var b strings.Builder
	b.WriteString("\n")
	heading := ansiBold + "Decoder" + ansiReset + ansiDim + "  " + version + ansiReset
	selection := "Choose a model with /config"
	if sess != nil && strings.TrimSpace(sess.Model()) != "" {
		selection = shortModelID(sess.Model())
		if provider := strings.TrimSpace(sess.Provider()); provider != "" {
			selection = provider + " · " + selection
		}
	}
	workspace, err := os.Getwd()
	if err != nil {
		workspace = "."
	}
	details := []string{heading, ansiDim + selection + ansiReset, ansiDim + shortenHomePath(workspace) + ansiReset, ""}
	mascot := decoderRobotArtLines()
	if width-indent >= 50 {
		for i, art := range mascot {
			b.WriteString(line(ansiOrange + art + ansiReset + "  " + details[i]))
		}
	} else {
		for _, art := range mascot {
			b.WriteString(line(ansiOrange + art + ansiReset))
		}
		for _, detail := range details[:3] {
			b.WriteString(line(detail))
		}
	}
	b.WriteString("\n")
	if indicators := welcomeIndicatorRow(skillsCount, snapshot.agentsOK, mcpCount, ansiDim, ansiGrayDim, ansiReset, "", ""); indicators != "" {
		b.WriteString(line(indicators))
		b.WriteString("\n")
	}
	if snapshot.setup.NeedsSetup {
		b.WriteString(line("Choose a provider and model with /config."))
	} else {
		b.WriteString(line(`Try "explain this repo" or describe a task.`))
	}
	b.WriteString(line(ansiDim + "/help for commands" + ansiReset))
	if resume := actLine(saved, sessionID); resume != "" {
		b.WriteString("\n")
		b.WriteString(line(ansiDim + resume + ansiReset))
	}
	return b.String()
}

type mcpServerNamed interface {
	MCPServerName() string
}

func connectedMCPCount(registry *tool.Registry) int {
	if registry == nil {
		return 0
	}
	servers := make(map[string]struct{})
	for _, candidate := range registry.PrimaryTools() {
		mcpTool, ok := candidate.(mcpServerNamed)
		if !ok || mcpTool.MCPServerName() == "" {
			continue
		}
		servers[mcpTool.MCPServerName()] = struct{}{}
	}
	return len(servers)
}

func welcomeIndicatorRow(skillsCount int, agentsOK bool, mcpCount int, activeC, idleC, rst, markPresent, markNone string) string {
	var parts []string
	if agentsOK {
		parts = append(parts, "Project instructions loaded")
	}
	if skillsCount > 0 {
		label := "skill"
		if skillsCount != 1 {
			label = "skills"
		}
		parts = append(parts, fmt.Sprintf("%d %s", skillsCount, label))
	}
	if mcpCount > 0 {
		label := "MCP server"
		if mcpCount != 1 {
			label = "MCP servers"
		}
		parts = append(parts, fmt.Sprintf("%d %s", mcpCount, label))
	}
	if len(parts) == 0 {
		return ""
	}
	return activeC + strings.Join(parts, " · ") + rst
}

// welcomeModeBadge is intentionally absent; decoder uses host permission tiers.

func actLine(saved *session.Session, sessionID string) string {
	if saved != nil && len(sessionID) >= 8 {
		return "Resumed session " + sessionID[:8]
	}
	return ""
}

func toolListSummary(registry *tool.Registry) string {
	if registry == nil {
		return "No tools enabled."
	}
	tools := registry.MuxTools()
	registered := len(registry.PrimaryTools())
	if len(tools) == 0 {
		return "No tools enabled."
	}
	var b strings.Builder
	if registered > len(tools) {
		b.WriteString(fmt.Sprintf("Model-visible tools (%d of %d registered — lazy surface):\n", len(tools), registered))
	} else {
		b.WriteString(fmt.Sprintf("Enabled tools (%d):\n", len(tools)))
	}
	for _, t := range tools {
		desc := t.Description
		if runes := []rune(desc); len(runes) > 96 {
			desc = string(runes[:96]) + "..."
		}
		b.WriteString(fmt.Sprintf("  %s — %s\n", t.Name, desc))
	}
	if registered > len(tools) {
		b.WriteString("\nUnlock more: ToolSearch with query select:<ToolName>")
	}
	return strings.TrimRight(b.String(), "\n")
}

func envSummary(provider, model string) string {
	return envSummaryWithSelection(provider, model, true)
}

func envSummaryWithSelection(provider, model string, includeSelection bool) string {
	var providers []string
	for _, gateway := range decoderconfig.GatewayStatuses(context.Background(), provider, model) {
		providers = append(providers, gateway.ID)
	}
	sort.Strings(providers)
	var b strings.Builder
	if includeSelection {
		b.WriteString(fmt.Sprintf("Provider: %s\nModel: %s\n\n", provider, model))
	}
	b.WriteString(fmt.Sprintf("Credentials (%s):\n", decoderconfig.CredentialStoreName()))
	for _, providerID := range providers {
		b.WriteString(fmt.Sprintf("  %s: %s\n", providerID, decoderconfig.EnvKeyStatus(providerID)))
	}
	return strings.TrimRight(b.String(), "\n")
}

func configCommandSummary(settings decoderconfig.Settings) string {
	_ = settings
	providerName := displayConfigValue(decoderconfig.ActiveProvider(context.Background()))
	modelName := displayConfigValue(decoderconfig.ActiveModel(context.Background()))
	keys := configuredKeyList()
	keysColor := infoSky
	if keys == "(none)" {
		keysColor = textMuted
	}
	return fmt.Sprintf(`%s

  /config  → paste API key (OS keychain) + pick model
  /path    → verify readiness in TUI
  decoder path (CLI)

%s:
  %s %s
  %s %s
  %s %s

Model catalog and routing live in mux — decoder is the UI only.`,
		auditTint("Setup (mux)", textPrimary),
		auditTint("Current", textPrimary),
		auditTint("provider:", textMuted), auditTint(providerName, infoSky),
		auditTint("model:", textMuted), auditTint(modelName, infoSky),
		auditTint("keys:", textMuted), auditTint(keys, keysColor))
}

func apiKeyConfigSummary() string {
	return auditTint("API keys ("+decoderconfig.CredentialStoreName()+")", textPrimary) + "\n" + indentedAPIKeyLines()
}

func configuredKeyList() string {
	var providers []string
	for _, line := range apiKeyStatusLines() {
		name, status, ok := strings.Cut(line, ": ")
		if ok && status == "set" {
			providers = append(providers, name)
		}
	}
	if len(providers) == 0 {
		return "(none)"
	}
	return strings.Join(providers, ", ")
}

func indentedAPIKeyLines() string {
	lines := apiKeyStatusLines()
	if len(lines) == 0 {
		return "  " + auditTint("(empty)", textMuted)
	}
	var b strings.Builder
	for _, line := range lines {
		name, status, ok := strings.Cut(line, ": ")
		if !ok {
			b.WriteString("  " + line + "\n")
			continue
		}
		b.WriteString("  " + auditTint(name, textPrimary) + ": " + auditTint(status, apiKeyStatusColor(status)) + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func apiKeyStatusColor(status string) color.Color {
	switch status {
	case "set":
		return doneGreen
	case "local":
		return infoSky
	default:
		return textMuted
	}
}

func apiKeyStatusLines() []string {
	providers := decoderconfig.AllSetupGateways()
	sort.Strings(providers)
	var lines []string
	for _, provider := range providers {
		lines = append(lines, fmt.Sprintf("%s: %s", provider, decoderconfig.EnvKeyStatus(provider)))
	}
	return lines
}

func displayConfigValue(value string) string {
	if strings.TrimSpace(value) == "" {
		return "(empty)"
	}
	return value
}
