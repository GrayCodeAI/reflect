package cmd

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"charm.land/bubbles/v2/textarea"

	reflectconfig "github.com/GrayCodeAI/reflect/internal/config"
	"github.com/GrayCodeAI/reflect/internal/tool"
	"github.com/GrayCodeAI/reflect/internal/ui/icons"
)

type welcomeMCPStub struct {
	name   string
	server string
}

// TestWelcomeScreenNerdIconsUnique renders the full welcome in Nerd mode
// for every execution state and asserts each PUA icon glyph appears at most
// once. Guards the "one icon per concept" rule on the welcome screen so the
// mode/iso/trust segments and the badge never reuse a glyph.
func TestWelcomeScreenNerdIconsUnique(t *testing.T) {
	icons.SetMode(icons.ModeNerd)
	defer icons.SetMode(icons.ModeASCII)

	out := buildWelcomeMessage(nil, "", nil, nil, reflectconfig.Settings{}, 0, false, 100, 24)
	seen := make(map[rune]struct{})
	for _, r := range out {
		if r < 0xE000 || r > 0xF8FF {
			continue
		}
		if _, dup := seen[r]; dup {
			t.Fatalf("PUA glyph %U reused on welcome screen:\n%s", r, out)
		}
		seen[r] = struct{}{}
	}
}

func (s welcomeMCPStub) Name() string                       { return s.name }
func (s welcomeMCPStub) Description() string                { return "test tool" }
func (s welcomeMCPStub) Parameters() map[string]interface{} { return nil }
func (s welcomeMCPStub) Execute(context.Context, json.RawMessage) (string, error) {
	return "", nil
}
func (s welcomeMCPStub) MCPServerName() string { return s.server }

func TestBuildWelcomeMessage_InlineShowsSetupGuidance(t *testing.T) {
	out := buildWelcomeMessage(nil, "", nil, nil, reflectconfig.Settings{}, 0, false, 100, 24)
	if !strings.Contains(out, "v") {
		t.Fatalf("inline welcome should show version, got:\n%s", out)
	}
	if strings.Contains(out, "WELCOME TO") {
		t.Fatalf("inline welcome should not render the old gate banner, got:\n%s", out)
	}
}

func TestBuildWelcomeMessage_InlineShowsGuidance(t *testing.T) {
	out := buildWelcomeMessageWithSnapshot(nil, "", nil, nil, reflectconfig.Settings{}, 1, 0, 0, 100, 24, welcomeStatusSnapshot{agentsOK: true}, "")
	for _, want := range []string{"Reflect", "Project instructions loaded", "1 skill", `Try "explain this repo"`, "/help"} {
		if !strings.Contains(out, want) {
			t.Errorf("welcome missing %q: %s", want, out)
		}
	}
	for _, hidden := range []string{"MCPs (0)", "Action Mode", "Trusted", "Skills (", "AGENTS.md", "____"} {
		if strings.Contains(out, hidden) {
			t.Errorf("welcome repeats clutter %q: %s", hidden, out)
		}
	}
}

func TestBuildWelcomeMessage_ShortTerminalUsesCompactCopy(t *testing.T) {
	out := buildWelcomeMessage(nil, "", nil, nil, reflectconfig.Settings{}, 0, false, 72, 20)
	if strings.Contains(out, "PgUp/Dn scroll chat") || strings.Contains(out, "for new session") {
		t.Fatalf("compact welcome should drop verbose descriptions, got:\n%s", out)
	}
	if !strings.Contains(out, "v") {
		t.Fatalf("compact welcome should keep version and execution mode, got:\n%s", out)
	}
}

func TestBuildWelcomeMessage_WideTerminalUsesReflectWordmark(t *testing.T) {
	for _, width := range []int{24, 40, 80, 120, 200} {
		out := buildWelcomeMessageWithSnapshot(nil, "", nil, nil, reflectconfig.Settings{}, 2, 1, 0, width, 40, welcomeStatusSnapshot{agentsOK: true}, "")
		if !strings.Contains(out, "Reflect") || !strings.Contains(out, "v"+DisplayVersion()) || !strings.Contains(out, "▀▀") || !strings.Contains(out, "▄▄") {
			t.Fatalf("width %d lost name/version: %s", width, out)
		}
		for _, line := range strings.Split(out, "\n") {
			if visibleWidth(line) > width {
				t.Fatalf("width %d overflow: %q", width, line)
			}
		}
	}
}

func TestBuildWelcomeMessage_ReflectWordmarkIsStatic(t *testing.T) {
	first := buildWelcomeMessageWithSnapshot(nil, "", nil, nil, reflectconfig.Settings{}, 0, 0, 0, 120, 40, welcomeStatusSnapshot{}, "")
	blink := buildWelcomeMessageWithSnapshot(nil, "", nil, nil, reflectconfig.Settings{}, 0, 0, 2, 120, 40, welcomeStatusSnapshot{}, "")
	if first != blink {
		t.Fatal("welcome heading should not animate")
	}
}

func TestBuildWelcomeMessage_GraphicalMascotKeepsTextFallback(t *testing.T) {
	out := buildWelcomeMessageWithSnapshotAndMascot(nil, "", nil, nil, reflectconfig.Settings{}, 0, 0, 0, 120, 40, welcomeStatusSnapshot{setup: reflectconfig.SetupState{NeedsSetup: true}}, "", true)
	if !strings.Contains(out, "Reflect") || !strings.Contains(out, "/config") || !strings.Contains(out, "▀▀") {
		t.Fatalf("welcome must retain heading and actionable setup guidance: %s", out)
	}
}

func TestEyeBlinkTick_CyclesEyeFrameStates(t *testing.T) {
	m := chatModel{input: textarea.New(), width: 100, height: 40}
	m.rebuildWelcomeCache()
	next, cmd := m.Update(eyeBlinkTickMsg{})
	nextModel := next.(chatModel)
	if nextModel.eyeFrame != 1 {
		t.Fatalf("eyeBlinkTickMsg eyeFrame = %d, want 1", nextModel.eyeFrame)
	}
	if cmd == nil {
		t.Fatal("eyeBlinkTickMsg should return next commands")
	}

	next2, _ := nextModel.Update(eyeFrameNextMsg{frame: 2})
	nextModel2 := next2.(chatModel)
	if nextModel2.eyeFrame != 2 {
		t.Fatalf("eyeFrameNextMsg frame 2 eyeFrame = %d, want 2", nextModel2.eyeFrame)
	}

	next3, _ := nextModel2.Update(eyeFrameNextMsg{frame: 3})
	nextModel3 := next3.(chatModel)
	if nextModel3.eyeFrame != 3 {
		t.Fatalf("eyeFrameNextMsg frame 3 eyeFrame = %d, want 3", nextModel3.eyeFrame)
	}

	next4, _ := nextModel3.Update(eyeFrameNextMsg{frame: 0})
	nextModel4 := next4.(chatModel)
	if nextModel4.eyeFrame != 0 {
		t.Fatalf("eyeFrameNextMsg frame 0 eyeFrame = %d, want 0", nextModel4.eyeFrame)
	}
}

func TestWelcomeIndicatorRow_UsesSemanticStatesAndCounts(t *testing.T) {
	for _, tc := range []struct {
		skills int
		agents bool
		mcp    int
		want   string
	}{
		{0, false, 0, ""},
		{1, true, 1, "Project instructions loaded · 1 skill · 1 MCP server"},
		{2, false, 3, "2 skills · 3 MCP servers"},
	} {
		got := welcomeIndicatorRow(tc.skills, tc.agents, tc.mcp, "", "", "", "", "")
		if got != tc.want {
			t.Errorf("status = %q, want %q", got, tc.want)
		}
	}
}

func TestConnectedMCPCount_CountsDistinctUsableServers(t *testing.T) {
	registry := tool.NewRegistry(
		welcomeMCPStub{name: "alpha-one", server: "alpha"},
		welcomeMCPStub{name: "alpha-two", server: "alpha"},
		welcomeMCPStub{name: "beta-one", server: "beta"},
		welcomeMCPStub{name: "not-mcp"},
	)

	if got := connectedMCPCount(registry); got != 2 {
		t.Fatalf("connectedMCPCount() = %d, want 2 distinct connected servers", got)
	}
	if got := connectedMCPCount(nil); got != 0 {
		t.Fatalf("connectedMCPCount(nil) = %d, want 0", got)
	}
}
