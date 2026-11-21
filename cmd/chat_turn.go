package cmd

import chatfeature "github.com/GrayCodeAI/reflect/internal/features/chat"

// turnHadThinkingOnly reports whether the latest user turn ended with internal
// reasoning visible but no assistant reply or tool activity. This is the TUI
// symptom of relay's ResponseErrorOnlyReasoning health check.
func turnHadThinkingOnly(messages []displayMsg) bool {
	return chatfeature.HadThinkingOnly(toFeatureMessages(messages))
}

// stripCurrentTurnThinking removes thinking messages from the latest user turn.
// Used when the engine retries after a reasoning-only response.
func stripCurrentTurnThinking(messages []displayMsg) []displayMsg {
	clean := chatfeature.StripCurrentTurnThinking(toFeatureMessages(messages))
	out := make([]displayMsg, 0, len(clean))
	// The policy only removes entries. Select its ordered subsequence from
	// the originals so call IDs, outcomes, and interactive metadata survive.
	for _, original := range messages {
		if len(out) < len(clean) && original.role == clean[len(out)].Role && original.content == clean[len(out)].Content {
			out = append(out, original)
		}
	}
	return out
}

func toFeatureMessages(messages []displayMsg) []chatfeature.Message {
	projected := make([]chatfeature.Message, 0, len(messages))
	for _, message := range messages {
		projected = append(projected, chatfeature.Message{Role: message.role, Content: message.content})
	}
	return projected
}
