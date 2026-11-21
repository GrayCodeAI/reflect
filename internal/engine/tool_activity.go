package engine

import "strings"

// ToolActivitySummary is presentation-only. Permission identities and matching
// retain their existing contract. Only action fields, never file contents or
// arbitrary argument maps, belong in the activity heading.
func ToolActivitySummary(name string, args map[string]interface{}) string {
	var keys []string
	switch canonicalToolName(name) {
	case "Bash", "PowerShell":
		keys = []string{"command"}
	case "Read", "Write", "Edit", "MultiEdit", "NotebookEdit", "StructuredEdit":
		keys = filePathArgKeys
	case "Grep", "Glob":
		keys = []string{"pattern"}
	case "Skill":
		keys = []string{"skill"}
	case "WebSearch":
		keys = []string{"query"}
	case "WebFetch":
		keys = []string{"url"}
	}
	for _, key := range keys {
		if value, ok := args[key].(string); ok && strings.TrimSpace(value) != "" {
			runes := []rune(value)
			if len(runes) > 240 {
				value = string(runes[:237]) + "..."
			}
			return value
		}
	}
	return ""
}
