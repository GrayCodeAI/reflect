package prompt

import "strings"

// IsSmallTalk recognizes complete conversational messages that need no tools
// or workspace context. Match whole phrases so "hi, fix this" remains a task.
func IsSmallTalk(message string) bool {
	text := strings.ToLower(strings.TrimSpace(message))
	text = strings.Trim(text, " \t\r\n.,!?;:")
	text = strings.Join(strings.Fields(text), " ")
	switch text {
	case "hi", "hi there", "hi reflect", "hello", "hello there", "hello reflect", "hey", "hey there", "hey reflect",
		"how are you", "how are you doing", "how are you doing today", "how's it going", "how's it going today",
		"what's up", "who are you", "what can you do",
		"thanks", "thank you", "thanks a lot", "thanks for that",
		"good morning", "good afternoon", "good evening", "nice to meet you",
		"goodbye", "bye":
		return true
	default:
		return false
	}
}
