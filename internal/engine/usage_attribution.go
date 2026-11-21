package engine

import "strings"

// UsageAttribution identifies the helper that owns a session's generations
// so the usage ledger can report per-helper spend (subagents, mission
// workers). The zero value means the main interactive session.
type UsageAttribution struct {
	// SubagentID names the spawned helper: a subagent type ("explore",
	// "plan", "general-purpose"), a mission worker, or a validator.
	SubagentID string
	// MissionID links the helper to the multi-agent mission run that
	// spawned it; empty for helpers outside a mission.
	MissionID string
}

func (a UsageAttribution) normalized() UsageAttribution {
	return UsageAttribution{
		SubagentID: strings.TrimSpace(a.SubagentID),
		MissionID:  strings.TrimSpace(a.MissionID),
	}
}

// SetUsageAttribution tags every usage-ledger record this session appends
// with the given helper identity (zero value = main session).
func (s *Session) SetUsageAttribution(a UsageAttribution) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.usageAttrib = a.normalized()
	s.mu.Unlock()
}

// usageAttribution returns the session's current helper attribution.
func (s *Session) usageAttribution() UsageAttribution {
	if s == nil {
		return UsageAttribution{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.usageAttrib
}
