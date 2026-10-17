package tool

import (
	"context"
	"encoding/json"

	"github.com/GrayCodeAI/reflect/internal/hosttools"
)

// HostTool wraps a hosttools.Manifest as a Reflect tool.
type HostTool struct {
	Manifest hosttools.Manifest
}

func (h HostTool) Name() string { return h.Manifest.Name }

func (h HostTool) Description() string {
	if h.Manifest.Description != "" {
		return h.Manifest.Description
	}
	return "Host-provided tool: " + h.Manifest.Name
}

func (h HostTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"input": map[string]interface{}{
				"type":        "string",
				"description": "JSON input passed to the host tool",
			},
		},
	}
}

func (h HostTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	return hosttools.Run(ctx, h.Manifest, string(input))
}

// LoadHostTools returns registered host tools.
func LoadHostTools() []Tool {
	manifests, err := hosttools.Load()
	if err != nil {
		return nil
	}
	out := make([]Tool, 0, len(manifests))
	for _, m := range manifests {
		out = append(out, HostTool{Manifest: m})
	}
	return out
}
