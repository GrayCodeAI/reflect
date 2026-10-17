// Package ext defines the single, run-scoped extension contract that Reflect (and
// sub-agents) use to plug in policy, redaction, verifiers, budgets and audit
// logging. It intentionally has no dependency on the LLM transport: callers
// translate a concrete tool call into the request shape below and feed back the
// decision it returns. A new hook instance is created per run so run-local
// state starts fresh and is never shared between two runs.
package ext

import (
	"context"
	"errors"
	"fmt"
)

// ToolCallRequest describes a tool invocation about to happen.
type ToolCallRequest struct {
	Agent     string
	ToolName  string
	Arguments map[string]any
	RawInput  string
}

// ToolDecision is the verdict returned by BeforeTool.
type ToolDecision string

const (
	// DecisionAllow lets the call run unchanged.
	DecisionAllow ToolDecision = "allow"
	// DecisionDeny blocks the call.
	DecisionDeny ToolDecision = "deny"
	// DecisionModify re-runs the call with ModifiedArgs.
	DecisionModify ToolDecision = "modify"
)

// ToolResult is the (possibly redacted) output of a tool.
type ToolResult struct {
	Content string
	IsError bool
}

// Extension is the single plug-in contract for a run.
//
// Every method is fail-closed: a returned error denies the corresponding
// step instead of being silently ignored. Methods that mutate their input
// (OnInput, AfterTool) should return the safe, possibly-redacted value.
type Extension interface {
	// Name identifies the extension for logs and audit.
	Name() string

	// OnInput may rewrite or reject the incoming user input. Returning an
	// error rejects the input.
	OnInput(ctx context.Context, input string) (string, error)

	// BeforeModel runs before a model call. Returning an error aborts the run.
	BeforeModel(ctx context.Context, messages []any) error

	// BeforeTool runs once per tool call and returns the verdict.
	BeforeTool(ctx context.Context, call *ToolCallRequest) (ToolDecision, map[string]any, string, error)

	// AfterTool may inspect or redact a tool result. Returning an error
	// withholds the result (fail-closed redaction).
	AfterTool(ctx context.Context, call *ToolCallRequest, result ToolResult) (ToolResult, error)

	// OnStop is invoked when the run ends for any reason.
	OnStop(ctx context.Context, reason string) error

	// Finish is the final cleanup hook.
	Finish(ctx context.Context) error
}

// Factory creates a fresh extension instance per run.
type Factory func() Extension

// Chain runs a set of extensions in registration order with fail-closed
// semantics. The zero value is usable (no-op).
type Chain struct {
	Extensions []Extension
}

// NewChain is a convenience constructor.
func NewChain(exts ...Extension) *Chain {
	return &Chain{Extensions: exts}
}

// RunOnInput chains OnInput calls; the first error short-circuits and is
// returned together with the last safe value.
func (c *Chain) RunOnInput(ctx context.Context, input string) (string, error) {
	current := input
	for _, e := range c.Extensions {
		next, err := e.OnInput(ctx, current)
		if err != nil {
			return current, fmt.Errorf("ext %s on_input: %w", e.Name(), err)
		}
		current = next
	}
	return current, nil
}

// RunBeforeModel chains BeforeModel calls.
func (c *Chain) RunBeforeModel(ctx context.Context, messages []any) error {
	for _, e := range c.Extensions {
		if err := e.BeforeModel(ctx, messages); err != nil {
			return fmt.Errorf("ext %s before_model: %w", e.Name(), err)
		}
	}
	return nil
}

// RunBeforeTool returns the most restrictive decision across extensions.
// Deny beats Modify beats Allow.
func (c *Chain) RunBeforeTool(ctx context.Context, call *ToolCallRequest) (ToolDecision, map[string]any, string, error) {
	args := call.Arguments
	decision := DecisionAllow
	for _, e := range c.Extensions {
		d, mod, reason, err := e.BeforeTool(ctx, call)
		if err != nil {
			return DecisionDeny, args, "", fmt.Errorf("ext %s before_tool: %w", e.Name(), err)
		}
		switch d {
		case DecisionDeny:
			return DecisionDeny, args, reason, nil
		case DecisionModify:
			if mod != nil {
				args = mod
				call.Arguments = mod
			}
			decision = DecisionModify
		case DecisionAllow:
		default:
			return DecisionDeny, args, reason, errors.New("invalid decision")
		}
	}
	return decision, args, "", nil
}

// RunAfterTool chains AfterTool; redaction failures withhold the result.
func (c *Chain) RunAfterTool(ctx context.Context, call *ToolCallRequest, result ToolResult) (ToolResult, error) {
	current := result
	for _, e := range c.Extensions {
		next, err := e.AfterTool(ctx, call, current)
		if err != nil {
			return ToolResult{}, fmt.Errorf("ext %s after_tool: %w", e.Name(), err)
		}
		current = next
	}
	return current, nil
}

// RunOnStop chains OnStop. Errors are collected and returned (fail-closed).
func (c *Chain) RunOnStop(ctx context.Context, reason string) error {
	for _, e := range c.Extensions {
		if err := e.OnStop(ctx, reason); err != nil {
			return fmt.Errorf("ext %s on_stop: %w", e.Name(), err)
		}
	}
	return nil
}

// RunFinish chains Finish.
func (c *Chain) RunFinish(ctx context.Context) error {
	for _, e := range c.Extensions {
		if err := e.Finish(ctx); err != nil {
			return fmt.Errorf("ext %s finish: %w", e.Name(), err)
		}
	}
	return nil
}

// Names returns the registered extension names.
func (c *Chain) Names() []string {
	out := make([]string, 0, len(c.Extensions))
	for _, e := range c.Extensions {
		out = append(out, e.Name())
	}
	return out
}
