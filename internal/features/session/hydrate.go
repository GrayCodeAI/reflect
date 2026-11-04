package session

import (
	store "github.com/GrayCodeAI/reflect/internal/session"
	"github.com/GrayCodeAI/reflect/internal/types"
)

// Hydrated contains both representations needed when resuming a session.
type Hydrated struct {
	Runtime []types.RelayMessage
	Display []HydratedDisplayMessage
}

// HydratedDisplayMessage retains tool identity and observed outcomes without
// coupling durable messages to terminal layout or expansion state.
type HydratedDisplayMessage struct {
	Role, Content string
	Call          *types.ToolCall
	Result        *types.ToolResult
}

// Hydrate converts one durable session once, keeping runtime and display
// projections consistent across TUI resume paths.
func Hydrate(saved *store.Session) Hydrated {
	if saved == nil {
		return Hydrated{}
	}
	result := Hydrated{Runtime: store.ToRuntimeMessages(saved.Messages)}
	calls := make(map[string]int)
	for _, message := range saved.Messages {
		if message.Role == "user" && len(message.ToolResults) == 0 {
			clear(calls) // Providers may reuse an ID on a subsequent user turn.
		}
		if (message.Role == "user" || message.Role == "assistant") && (len(message.ToolUse) == 0 && len(message.ToolResults) == 0 || message.Content != "") {
			result.Display = append(result.Display, HydratedDisplayMessage{Role: message.Role, Content: message.Content})
		}
		for _, call := range message.ToolUse {
			if call.ID != "" {
				calls[call.ID] = len(result.Display)
			}
			result.Display = append(result.Display, HydratedDisplayMessage{Role: "tool_use", Call: &call})
		}
		for _, output := range message.ToolResults {
			var call *types.ToolCall
			if index, ok := calls[output.ToolUseID]; ok && output.ToolUseID != "" {
				result.Display[index].Result = &output
				call = result.Display[index].Call
			}
			result.Display = append(result.Display, HydratedDisplayMessage{Role: "tool_result", Content: output.Content, Call: call, Result: &output})
		}
		if message.Role == "tool_result" && len(message.ToolResults) == 0 {
			result.Display = append(result.Display, HydratedDisplayMessage{Role: "tool_result", Content: message.Content})
		}
	}
	return result
}
