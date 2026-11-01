package engine

import (
	"fmt"
	"strings"

	"github.com/GrayCodeAI/reflect/internal/engine/safety"
	"github.com/GrayCodeAI/reflect/internal/tool"
)

// WorkMode is the product-level Plan / Build / Review control plane.
// It is orthogonal to shellmode (auto|shell|agent) and SpecStage:
// WorkMode steers tool visibility and bash mutability for everyday use.
type WorkMode string

const (
	// WorkModeAct is full build mode: essential tools + writes + bash.
	WorkModeAct WorkMode = "act"
	// WorkModePlan is research/planning: read-oriented tools, read-only bash.
	WorkModePlan WorkMode = "plan"
	// WorkModeReview is inspect-only: no writes, no mutating shell.
	WorkModeReview WorkMode = "review"
)

// EssentialModelTools are always offered to the model in Act mode (lazy surface).
// Optional tools remain registered for execution/ToolSearch but stay hidden until promoted.
var EssentialModelTools = []string{
	"Bash", "Read", "Write", "Edit", "StructuredEdit", "MultiEdit",
	"LS", "Glob", "Grep",
	"WebFetch", "WebSearch", "ToolSearch", "Skill",
	"Agent", "AskUserQuestion", "TodoWrite",
	"TaskOutput", "TaskStop", "WaitTasks", "KillTask", "Monitor",
	"LSP", "Browser", "Screenshot",
}

// PlanModelTools are model-visible in Plan mode.
var PlanModelTools = []string{
	"Read", "LS", "Glob", "Grep", "Bash",
	"WebFetch", "WebSearch", "ToolSearch", "Skill",
	"AskUserQuestion",
	"CodeSearch", "CodeGraph", "Impact",
}

// ReviewModelTools are model-visible in Review mode (read-only).
var ReviewModelTools = []string{
	"Read", "LS", "Glob", "Grep", "Bash",
	"ToolSearch", "AskUserQuestion",
	"CodeSearch", "CodeGraph", "Impact",
	"WebFetch",
}

const workModePlanPrompt = `## Work mode: PLAN
Research the request and produce one concise Markdown implementation plan.
- Prefer Read, Grep, Glob, LS, and read-only Bash inspection.
- Do not modify files or run mutating commands.
- Include the goal, approach, files affected, ordered steps, and verification.
- Keep everything in this one plan; do not create proposal, specification, design, or tasks files.
- Do not invoke the advanced spec workflow unless the user explicitly starts /spec.
- The interactive host saves your final answer as the session's single plan document.
- End with a concrete step list the user can review before switching to Build mode.`

const workModeReviewPrompt = `## Work mode: REVIEW
You are in review mode. Inspect code and report findings with file:line evidence.
- Read-only tools only; do not modify files.
- Prefer concrete PASS/FAIL style verdicts with citations.`

// ParseWorkMode accepts plan|act|review (and aliases).
func ParseWorkMode(s string) (WorkMode, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "act", "build", "agent", "write":
		return WorkModeAct, nil
	case "plan", "planning", "research":
		return WorkModePlan, nil
	case "review", "inspect", "readonly", "read-only", "ro":
		return WorkModeReview, nil
	default:
		return "", fmt.Errorf("unknown work mode %q (want plan|build|review)", s)
	}
}

// SetWorkMode applies tool visibility, bash policy, and stores the mode.
// When a journal is wired, emits plan.mode events to track the transition,
// matching DSH's plan/mode event (last-write-wins on replay).
func (s *Session) SetWorkMode(mode WorkMode) error {
	if s == nil {
		return fmt.Errorf("session is nil")
	}
	mode, err := ParseWorkMode(string(mode))
	if err != nil {
		return err
	}
	s.mu.Lock()
	oldMode := s.workMode
	s.workMode = mode
	s.mu.Unlock()

	// Emit plan.mode journal event — only when the mode actually changes.
	active := mode == WorkModePlan
	if j := s.persist.Journal(); j != nil && active != (oldMode == WorkModePlan) {
		j.AppendPlanMode(active)
	}

	reg := (*tool.Registry)(nil)
	if s.Tools() != nil {
		s.Tools().setWorkMode(mode)
		reg = s.Tools().Registry()
	}
	switch mode {
	case WorkModePlan:
		if s.Tools() != nil {
			s.Tools().SetReadOnlyBash(true)
		}
		if reg != nil {
			reg.SetModelVisibility(PlanModelTools)
		}
	case WorkModeReview:
		if s.Tools() != nil {
			s.Tools().SetReadOnlyBash(true)
		}
		if reg != nil {
			reg.SetModelVisibility(ReviewModelTools)
		}
	default: // Act
		if s.Tools() != nil {
			s.Tools().SetReadOnlyBash(false)
		}
		if reg != nil {
			reg.SetModelVisibility(EssentialModelTools)
		}
	}
	return nil
}

// DisplayName keeps product labels independent of persisted mode identifiers.
// Build retains the wire value "act" for existing sessions and integrations.
func (mode WorkMode) DisplayName() string {
	switch mode {
	case WorkModePlan:
		return "Plan"
	case WorkModeReview:
		return "Review"
	default:
		return "Build"
	}
}

// NextWorkMode is the everyday keyboard cycle.
func NextWorkMode(mode WorkMode) WorkMode {
	switch mode {
	case WorkModePlan:
		return WorkModeAct
	case WorkModeAct, "":
		return WorkModeReview
	default:
		return WorkModePlan
	}
}

// WorkMode returns the active work mode (default act).
func (s *Session) WorkMode() WorkMode {
	if s == nil {
		return WorkModeAct
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.workMode == "" {
		return WorkModeAct
	}
	return s.workMode
}

// workModeSystemAddon returns ephemeral system prompt text for the active mode.
func (s *Session) workModeSystemAddon() string {
	if s.PermSvc() != nil && s.PermSvc().SpecStage() != safety.SpecStageNone {
		return ""
	}
	switch s.WorkMode() {
	case WorkModePlan:
		return workModePlanPrompt
	case WorkModeReview:
		return workModeReviewPrompt
	default:
		return ""
	}
}
