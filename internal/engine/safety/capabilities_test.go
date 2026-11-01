package safety

import "testing"

func TestLocalSkillLoadingIsReadOnly(t *testing.T) {
	p := ToolPolicyFor("Skill")
	if p.DefaultRisk != RiskLow || len(p.Capabilities) != 1 || p.Capabilities[0] != CapabilityFilesystemRead {
		t.Fatalf("local instruction loading declares executable effects: %#v", p)
	}
	builder := ProfileFromLevel(AutonomySemi)
	if builder.NeedsPermission("Skill", true) {
		t.Fatal("Builder asks before reading local skill instructions")
	}
	if !builder.NeedsPermission("Bash", false) || !builder.NeedsPermission("Write", false) {
		t.Fatal("skill loading relaxed subsequent command or edit checks")
	}
	if !ProfileFromLevel(AutonomySupervised).NeedsPermission("Skill", true) {
		t.Fatal("Always Ask no longer asks for skill loading")
	}
}

func TestToolPolicyForReturnsDefensiveCapabilities(t *testing.T) {
	policy := ToolPolicyFor("Write")
	if policy.Name != "Write" || policy.DefaultRisk != RiskMedium {
		t.Fatalf("unexpected Write policy: %#v", policy)
	}
	policy.Capabilities[0] = CapabilityUnknown
	if ToolPolicyFor("Write").Capabilities[0] != CapabilityFilesystemWrite {
		t.Fatal("ToolPolicyFor returned mutable registry state")
	}
}

func TestToolPolicyForUnknownFailsClosed(t *testing.T) {
	policy := ToolPolicyFor("plugin_future_tool")
	if policy.DefaultRisk != RiskHigh || len(policy.Capabilities) != 1 || policy.Capabilities[0] != CapabilityUnknown {
		t.Fatalf("unexpected unknown policy: %#v", policy)
	}
}

func TestToolPolicyForCanonicalAliases(t *testing.T) {
	if got := ToolPolicyFor("bash"); got.Name != "Bash" || got.Capabilities[0] != CapabilityProcessExecute {
		t.Fatalf("bash alias did not resolve: %#v", got)
	}
}

func TestToolPolicyForBuiltInSurfaceIsDeclared(t *testing.T) {
	for _, name := range []string{
		"ToolSearch", "SessionQuery", "Skill", "Agent", "TodoWrite",
		"TerminalCreate", "TerminalRead", "RequestCredential",
	} {
		policy := ToolPolicyFor(name)
		if !isKnownTool(name) {
			t.Fatalf("built-in tool %q has no declared capability policy: %#v", name, policy)
		}
	}
}
