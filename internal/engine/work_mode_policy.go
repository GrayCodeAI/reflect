package engine

import (
	"fmt"

	"github.com/GrayCodeAI/reflect/internal/engine/safety"
)

func (s *ToolService) setWorkMode(mode WorkMode) {
	s.executionConfigMu.Lock()
	s.workMode = mode
	s.readOnlyBash = mode != WorkModeAct
	s.executionConfigMu.Unlock()
}

// Model visibility helps discovery; execution restrictions remain authoritative
// even when ToolSearch promotes a tool or a provider calls it directly.
func (s *ToolService) workModeRestriction(name string) string {
	stage := safety.SpecStageNone
	if s.deps.permissions != nil {
		stage = s.deps.permissions.SpecStage()
	}
	canonical := canonicalToolName(name)
	policy := safety.ToolPolicyFor(name)
	for _, capability := range policy.Capabilities {
		if capability == safety.CapabilitySpecWrite && stage == safety.SpecStageNone {
			return "Start /spec explicitly to use advanced specification tools."
		}
	}
	switch canonical {
	case "Proposal", "Specify", "Design", "Plan", "Tasks", "ApproveImplementation":
		if stage == safety.SpecStageNone {
			return "The advanced spec workflow is inactive. The user must start /spec before using spec-generation tools."
		}
	}
	s.executionConfigMu.RLock()
	mode := s.workMode
	s.executionConfigMu.RUnlock()
	if mode != "" && mode != WorkModeAct {
		// Bash enforces a command allowlist through ReadOnlyBash. Other shell
		// tools have no equivalent guard and stay unavailable in read-only modes.
		if canonical == "Bash" {
			return ""
		}
		for _, capability := range policy.Capabilities {
			switch capability {
			case safety.CapabilityFilesystemWrite, safety.CapabilityFilesystemDelete, safety.CapabilityProcessExecute, safety.CapabilityConfigurationWrite, safety.CapabilityDestructive, safety.CapabilityUnknown:
				return fmt.Sprintf("%s mode is read-only. Switch to Build to use %s.", mode.DisplayName(), canonical)
			}
		}
	}
	return ""
}
