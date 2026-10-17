package safety

import (
	"context"
	"errors"
	"fmt"

	"github.com/GrayCodeAI/reflect/internal/ext"
)

// PolicyExtension adapts a PermissionEngine to the unified ext.Extension
// contract. It is opt-in: callers install the returned extension in an
// ext.Chain and install that chain on a PermissionService via SetExtChain,
// or execute it independently in previews/tests.
type PolicyExtension struct {
	Engine *PermissionEngine
}

// NewPolicyExtension wraps a PermissionEngine as an ext.Extension.
func NewPolicyExtension(pe *PermissionEngine) *PolicyExtension {
	return &PolicyExtension{Engine: pe}
}

// AsExtension returns the engine as an ext.Extension.
func (pe *PermissionEngine) AsExtension() ext.Extension {
	return NewPolicyExtension(pe)
}

func (p *PolicyExtension) Name() string { return "permission-engine" }

func (p *PolicyExtension) OnInput(ctx context.Context, input string) (string, error) {
	return input, nil
}

func (p *PolicyExtension) BeforeModel(ctx context.Context, messages []any) error {
	return nil
}

func (p *PolicyExtension) BeforeTool(ctx context.Context, call *ext.ToolCallRequest) (ext.ToolDecision, map[string]any, string, error) {
	if p.Engine == nil {
		return ext.DecisionDeny, nil, "permission engine unavailable", errors.New("permission engine unavailable")
	}
	info := ToolCallInfo{
		Name: call.ToolName,
		ID:   call.RawInput,
		Args: call.Arguments,
	}
	granted, denyMsg := p.Engine.CheckTool(ctx, info)
	if granted {
		return ext.DecisionAllow, call.Arguments, "", nil
	}
	reason := "permission denied"
	if denyMsg != "" {
		reason = denyMsg
	}
	return ext.DecisionDeny, nil, reason, fmt.Errorf("%s", reason)
}

func (p *PolicyExtension) AfterTool(ctx context.Context, call *ext.ToolCallRequest, result ext.ToolResult) (ext.ToolResult, error) {
	return result, nil
}

func (p *PolicyExtension) OnStop(ctx context.Context, reason string) error {
	return nil
}

func (p *PolicyExtension) Finish(ctx context.Context) error {
	return nil
}
