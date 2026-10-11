package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/GrayCodeAI/decoder/internal/intelligence/memory"
	"github.com/GrayCodeAI/decoder/internal/token"
)

// ToolExecutor is a function that executes a named tool with JSON input.
// This avoids importing the tool package (which already imports mcp).
type ToolExecutor func(ctx context.Context, name string, input json.RawMessage) (string, error)

// boolPtr returns a pointer to b, for the pointer-typed MCP annotation hints.
func boolPtr(b bool) *bool { return &b }

// readOnlyAnnotations marks a tool that only reads/inspects and never mutates
// the workspace, so a client can run it without prompting.
func readOnlyAnnotations(title string) *ToolAnnotations {
	return &ToolAnnotations{Title: title, ReadOnlyHint: boolPtr(true), DestructiveHint: boolPtr(false)}
}

// RegisterDefaultTools registers decoder's standard capabilities as MCP tools.
// If executor is non-nil, tools that delegate to decoder's tool registry will
// use it for execution; otherwise those tools return a not-configured error.
func RegisterDefaultTools(server *MCPServer, executor ToolExecutor) {
	server.RegisterTool(decoderChatTool(executor))
	server.RegisterTool(decoderSearchTool(executor))
	server.RegisterTool(decoderMemoryRecallTool(executor))
	server.RegisterTool(decoderMemoryStoreTool(executor))
	server.RegisterTool(decoderCompressTool(executor))
}

// decoderChatTool sends a prompt to decoder and returns the response.
func decoderChatTool(executor ToolExecutor) MCPToolHandler {
	return MCPToolHandler{
		Name: "decoder_chat",
		Description: "Send a prompt to the decoder AI coding agent and receive a response. " +
			"WARNING: this runs an autonomous agent that may execute shell commands and modify files.",
		Annotations: &ToolAnnotations{
			Title:           "Run decoder agent",
			ReadOnlyHint:    boolPtr(false),
			DestructiveHint: boolPtr(true),
			OpenWorldHint:   boolPtr(true),
		},
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"prompt": map[string]interface{}{
					"type":        "string",
					"description": "The prompt or question to send to decoder.",
				},
			},
			"required": []string{"prompt"},
		},
		Handler: func(ctx context.Context, params json.RawMessage) (string, error) {
			var input struct {
				Prompt string `json:"prompt"`
			}
			if err := json.Unmarshal(params, &input); err != nil {
				return "", fmt.Errorf("invalid input: %w", err)
			}
			if input.Prompt == "" {
				return "", fmt.Errorf("prompt is required")
			}
			return delegateToExecutor(ctx, executor, "agent", params)
		},
	}
}

// decoderSearchTool searches across decoder sessions.
func decoderSearchTool(executor ToolExecutor) MCPToolHandler {
	return MCPToolHandler{
		Name:        "decoder_search",
		Description: "Search across decoder sessions and conversation history.",
		Annotations: readOnlyAnnotations("Search decoder sessions"),
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "The search query.",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of results to return.",
				},
			},
			"required": []string{"query"},
		},
		Handler: func(ctx context.Context, params json.RawMessage) (string, error) {
			var input struct {
				Query string `json:"query"`
				Limit int    `json:"limit"`
			}
			if err := json.Unmarshal(params, &input); err != nil {
				return "", fmt.Errorf("invalid input: %w", err)
			}
			if input.Query == "" {
				return "", fmt.Errorf("query is required")
			}
			return delegateToExecutor(ctx, executor, "code_search", params)
		},
	}
}

// decoderMemoryRecallTool recalls information from decoder's local memory store.
func decoderMemoryRecallTool(executor ToolExecutor) MCPToolHandler {
	return MCPToolHandler{
		Name:        "decoder_memory_recall",
		Description: "Recall stored information from decoder's local persistent memory.",
		Annotations: readOnlyAnnotations("Recall from decoder memory"),
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "The memory recall query.",
				},
				"namespace": map[string]interface{}{
					"type":        "string",
					"description": "Optional namespace to search within.",
				},
			},
			"required": []string{"query"},
		},
		Handler: func(ctx context.Context, params json.RawMessage) (string, error) {
			var input struct {
				Query     string `json:"query"`
				Namespace string `json:"namespace"`
			}
			if err := json.Unmarshal(params, &input); err != nil {
				return "", fmt.Errorf("invalid input: %w", err)
			}
			if input.Query == "" {
				return "", fmt.Errorf("query is required")
			}
			matches, err := memory.Search(input.Query)
			if err != nil {
				return "", fmt.Errorf("memory recall failed: %w", err)
			}
			if len(matches) == 0 {
				return fmt.Sprintf("No memories matched %q.", input.Query), nil
			}
			var b strings.Builder
			for i, m := range matches {
				if i > 0 {
					b.WriteString("\n")
				}
				fmt.Fprintf(&b, "- %s", m.Content)
			}
			return b.String(), nil
		},
	}
}

// decoderMemoryStoreTool stores information to decoder's local memory store.
func decoderMemoryStoreTool(executor ToolExecutor) MCPToolHandler {
	return MCPToolHandler{
		Name:        "decoder_memory_store",
		Description: "Store information in decoder's local persistent memory for future recall.",
		Annotations: &ToolAnnotations{
			Title:           "Store in decoder memory",
			ReadOnlyHint:    boolPtr(false),
			DestructiveHint: boolPtr(false), // additive write, does not destroy existing data
		},
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"key": map[string]interface{}{
					"type":        "string",
					"description": "A key or label for the memory entry.",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "The content to store.",
				},
				"namespace": map[string]interface{}{
					"type":        "string",
					"description": "Optional namespace for organization.",
				},
			},
			"required": []string{"key", "content"},
		},
		Handler: func(ctx context.Context, params json.RawMessage) (string, error) {
			var input struct {
				Key       string `json:"key"`
				Content   string `json:"content"`
				Namespace string `json:"namespace"`
			}
			if err := json.Unmarshal(params, &input); err != nil {
				return "", fmt.Errorf("invalid input: %w", err)
			}
			if input.Key == "" || input.Content == "" {
				return "", fmt.Errorf("key and content are required")
			}
			if err := memory.Save(&memory.Memory{
				Content: input.Content,
				Tags:    []string{input.Key},
			}); err != nil {
				return "", fmt.Errorf("memory store failed: %w", err)
			}
			return fmt.Sprintf("Stored memory under %q.", input.Key), nil
		},
	}
}

// decoderCompressTool compresses text via the embedded token engine.
func decoderCompressTool(executor ToolExecutor) MCPToolHandler {
	return MCPToolHandler{
		Name:        "decoder_compress",
		Description: "Compress text using decoder's embedded token engine to reduce token usage while preserving meaning.",
		Annotations: readOnlyAnnotations("Compress text (token engine)"),
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"text": map[string]interface{}{
					"type":        "string",
					"description": "The text to compress.",
				},
				"ratio": map[string]interface{}{
					"type":        "number",
					"description": "Target compression ratio (0.0-1.0). Lower means more compression.",
				},
			},
			"required": []string{"text"},
		},
		Handler: func(ctx context.Context, params json.RawMessage) (string, error) {
			var input struct {
				Text  string  `json:"text"`
				Ratio float64 `json:"ratio"`
			}
			if err := json.Unmarshal(params, &input); err != nil {
				return "", fmt.Errorf("invalid input: %w", err)
			}
			if input.Text == "" {
				return "", fmt.Errorf("text is required")
			}
			budget := token.EstimateTokens(input.Text)
			if input.Ratio > 0 && input.Ratio < 1 {
				budget = int(float64(budget) * input.Ratio)
			}
			if budget < 1 {
				budget = 1
			}
			compressed, stats := token.Compress(input.Text, budget)
			result, err := json.Marshal(map[string]any{
				"compressed":        compressed,
				"original_tokens":   stats.OriginalTokens,
				"final_tokens":      stats.FinalTokens,
				"tokens_saved":      stats.TokensSaved,
				"reduction_percent": stats.ReductionPercent,
			})
			if err != nil {
				return "", err
			}
			return string(result), nil
		},
	}
}

// delegateToExecutor executes a tool through the provided executor function.
func delegateToExecutor(ctx context.Context, executor ToolExecutor, name string, params json.RawMessage) (string, error) {
	if executor == nil {
		return "", fmt.Errorf("tool executor not configured")
	}
	return executor(ctx, name, params)
}
