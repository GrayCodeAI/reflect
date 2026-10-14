package ext

import (
	"context"
	"errors"
	"testing"
)

type allowExt struct{ name string }

func (a *allowExt) Name() string { return a.name }
func (a *allowExt) OnInput(ctx context.Context, input string) (string, error) {
	return input + "!", nil
}

func (a *allowExt) BeforeModel(ctx context.Context, messages []any) error { return nil }

func (a *allowExt) BeforeTool(ctx context.Context, call *ToolCallRequest) (ToolDecision, map[string]any, string, error) {
	return DecisionAllow, nil, "", nil
}

func (a *allowExt) AfterTool(ctx context.Context, call *ToolCallRequest, result ToolResult) (ToolResult, error) {
	return result, nil
}
func (a *allowExt) OnStop(ctx context.Context, reason string) error { return nil }
func (a *allowExt) Finish(ctx context.Context) error                { return nil }

type denyExt struct{ name string }

func (d *denyExt) Name() string { return d.name }
func (d *denyExt) OnInput(ctx context.Context, input string) (string, error) {
	return input, errors.New("no")
}

func (d *denyExt) BeforeModel(ctx context.Context, messages []any) error { return errors.New("boom") }

func (d *denyExt) BeforeTool(ctx context.Context, call *ToolCallRequest) (ToolDecision, map[string]any, string, error) {
	return DecisionDeny, nil, "denied by denyExt", nil
}

func (d *denyExt) AfterTool(ctx context.Context, call *ToolCallRequest, result ToolResult) (ToolResult, error) {
	return ToolResult{}, errors.New("redact failed")
}

func (d *denyExt) OnStop(ctx context.Context, reason string) error { return errors.New("stop fail") }

func (d *denyExt) Finish(ctx context.Context) error { return errors.New("finish fail") }

func TestChainOnInputFailClosed(t *testing.T) {
	c := NewChain(&allowExt{"a"}, &denyExt{"b"})
	got, err := c.RunOnInput(context.Background(), "hi")
	if err == nil {
		t.Fatal("expected error")
	}
	if got != "hi!" {
		t.Fatalf("got %q want %q", got, "hi!")
	}
}

func TestChainBeforeModelFailClosed(t *testing.T) {
	c := NewChain(&allowExt{"a"}, &denyExt{"b"})
	if err := c.RunBeforeModel(context.Background(), nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestChainBeforeToolDenyWins(t *testing.T) {
	c := NewChain(&allowExt{"a"}, &denyExt{"b"})
	decision, _, _, err := c.RunBeforeTool(context.Background(), &ToolCallRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if decision != DecisionDeny {
		t.Fatalf("decision=%s", decision)
	}
}

func TestChainAfterToolWithholds(t *testing.T) {
	c := NewChain(&denyExt{"b"})
	_, err := c.RunAfterTool(context.Background(), &ToolCallRequest{}, ToolResult{Content: "secret"})
	if err == nil {
		t.Fatal("expected withhold")
	}
}

func TestChainOnStopFinish(t *testing.T) {
	c := NewChain(&denyExt{"b"})
	if err := c.RunOnStop(context.Background(), "done"); err == nil {
		t.Fatal("expected on_stop error")
	}
	if err := c.RunFinish(context.Background()); err == nil {
		t.Fatal("expected finish error")
	}
}

func TestChainNames(t *testing.T) {
	c := NewChain(&allowExt{"a"}, &denyExt{"b"})
	if len(c.Names()) != 2 {
		t.Fatalf("names=%v", c.Names())
	}
}
