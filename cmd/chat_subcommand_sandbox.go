package cmd

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/GrayCodeAI/reflect/internal/config"
	"github.com/GrayCodeAI/reflect/internal/sandbox"
)

// sandboxSubcommand reports and toggles sandboxed tool execution.
type sandboxSubcommand struct{}

func (c *sandboxSubcommand) Name() string      { return "sandbox" }
func (c *sandboxSubcommand) Aliases() []string { return nil }
func (c *sandboxSubcommand) Description() string {
	return "show sandbox status (Mirror-style isolated execution)"
}
func (c *sandboxSubcommand) Usage() string { return "/sandbox" }

func (c *sandboxSubcommand) Handle(m *chatModel, args []string, text string) (tea.Model, tea.Cmd) {
	s := config.LoadSettings()
	mode := s.Sandbox
	if envMode := os.Getenv("REFLECT_SANDBOX"); envMode != "" {
		mode = envMode
	}
	cfg := sandbox.Config{Mode: sandbox.NormalizeMode(mode), Image: os.Getenv("REFLECT_SANDBOX_IMAGE")}
	status := "off (host execution)"
	if sandbox.Enabled(cfg) {
		status = "on (" + string(cfg.Mode) + ")"
	}
	docker := "not found"
	if sandbox.DockerAvailable() {
		docker = "available"
	}
	msb := "not found"
	if sandbox.MicrosandboxAvailable() {
		msb = "available"
	}
	m.messages = append(m.messages, displayMsg{role: "system", content: fmt.Sprintf(
		"Sandbox: %s\nDocker: %s\nMicrosandbox (msb): %s\n\nTo enable, set \"sandbox\": \"docker\" or \"microsandbox\" in ~/.reflect/settings.json\nor export REFLECT_SANDBOX=docker|microsandbox (and optionally REFLECT_SANDBOX_IMAGE).",
		status, docker, msb,
	)})
	return m, nil
}

func init() {
	subcommandRegistry.Register(&sandboxSubcommand{})
}
