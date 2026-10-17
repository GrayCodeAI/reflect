package session

import "github.com/GrayCodeAI/reflect/internal/types"

// FromRuntimeMessages converts Reflect runtime messages into persisted session messages.
func FromRuntimeMessages(in []types.RelayMessage) []Message {
	if len(in) == 0 {
		return nil
	}
	out := make([]Message, len(in))
	for i, msg := range in {
		out[i] = Message{
			Role:         msg.Role,
			Content:      msg.Content,
			Thinking:     msg.Thinking,
			ContentParts: msg.ContentParts,
			Images:       msg.Images,
			ToolUse:      FromRuntimeToolCalls(msg.ToolUse),
			ToolResults:  FromRuntimeToolResults(msg.ToolResults),
		}
	}
	return out
}

// FromRuntimeToolCalls converts Reflect runtime tool calls into persisted contracts.
// types.ToolCall and session.ToolCall are identical (both alias tools.ToolCall).
func FromRuntimeToolCalls(in []types.ToolCall) []ToolCall {
	return in
}

// FromRuntimeToolResults converts Reflect runtime tool results into persisted contracts.
func FromRuntimeToolResults(in []types.ToolResult) []ToolResult {
	return in
}

// ToRuntimeMessages converts persisted session messages back into Reflect runtime messages.
func ToRuntimeMessages(in []Message) []types.RelayMessage {
	if len(in) == 0 {
		return nil
	}
	out := make([]types.RelayMessage, len(in))
	for i, msg := range in {
		out[i] = types.RelayMessage{
			Role:         msg.Role,
			Content:      msg.Content,
			Thinking:     msg.Thinking,
			ContentParts: msg.ContentParts,
			Images:       msg.Images,
			ToolUse:      ToRuntimeToolCalls(msg.ToolUse),
			ToolResults:  ToRuntimeToolResults(msg.ToolResults),
		}
	}
	return out
}

// ToRuntimeToolCalls converts persisted contracts back into Reflect runtime tool calls.
func ToRuntimeToolCalls(in []ToolCall) []types.ToolCall {
	return in
}

// ToRuntimeToolResults converts persisted contracts back into Reflect runtime tool results.
func ToRuntimeToolResults(in []ToolResult) []types.ToolResult {
	return in
}
