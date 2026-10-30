// Package permissions — RememberedRules is the unified rule store (proposal B).
//
// It collapses the four previously separate stores (AutoAllow settings,
// ExactRules stableid store, UnifiedGrants, PermissionMemory) into a single
// table keyed by (tool, identity) with a decision, source, scope, and optional
// expiry. The permission engine consults this one store instead of juggling
// ExactRules + UnifiedGrants + Memory + AutoAllow with implicit precedence.
package permissions

import (
	"sync"
	"time"
)

// RuleSource is an alias for GrantSource so the unified store reuses the
// existing source-priority vocabulary.
type RuleSource = GrantSource

// RuleScope records how broadly a rule applies.
type RuleScope string

const (
	ScopeSession RuleScope = "session"
	ScopeProject RuleScope = "project"
	ScopeGlobal  RuleScope = "global"
)

// RememberedRule is one row in the unified table.
type RememberedRule struct {
	Tool     string
	Identity string // full canonical identity (or "*" for tool-wide)
	Pattern  string // glob pattern for non-exact grants
	Exact    bool
	Allow    bool
	Source   RuleSource
	Scope    RuleScope
	Expires  time.Time // zero = never
	Label    string
}

// Active reports whether the rule is in effect at now.
func (r RememberedRule) Active(now time.Time) bool {
	if r.Expires.IsZero() {
		return true
	}
	return now.Before(r.Expires)
}

// RememberedRules is the single rule store consulted by the permission engine.
type RememberedRules struct {
	mu    sync.RWMutex
	rules []RememberedRule
}

// NewRememberedRules creates an empty store.
func NewRememberedRules() *RememberedRules {
	return &RememberedRules{}
}

// Add appends a rule.
func (r *RememberedRules) Add(rule RememberedRule) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rules = append(r.rules, rule)
}

// Remove deletes rules matching tool+identity.
func (r *RememberedRules) Remove(toolName, identity string) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	out := r.rules[:0]
	for _, g := range r.rules {
		if g.Tool == toolName && g.Identity == identity {
			continue
		}
		out = append(out, g)
	}
	r.rules = out
}

// Rules returns a snapshot copy.
func (r *RememberedRules) Rules() []RememberedRule {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]RememberedRule, len(r.rules))
	copy(out, r.rules)
	return out
}

// Check evaluates a tool call against the unified rule set. Returns (allowed, found).
// Precedence: deny > allow, then higher source, then higher specificity (exact > glob).
func (r *RememberedRules) Check(toolName, summary, identity string, now time.Time) (allowed bool, found bool) {
	if r == nil {
		return false, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	toolName = CanonicalToolName(toolName)
	var winner RememberedRule
	foundWinner := false
	for _, g := range r.rules {
		if !g.Active(now) {
			continue
		}
		if g.Tool != "*" && g.Tool != toolName {
			continue
		}
		if g.Exact && g.Identity != identity {
			continue
		}
		if !g.Exact && g.Pattern != "" && !grantMatchPattern(g.Pattern, summary) {
			continue
		}
		if !foundWinner || grantPrecedesRemembered(g, winner) {
			winner = g
			foundWinner = true
		}
	}
	if !foundWinner {
		return false, false
	}
	return winner.Allow, true
}

func grantPrecedesRemembered(candidate, current RememberedRule) bool {
	if candidate.Allow != current.Allow {
		return !candidate.Allow // deny first
	}
	if candidate.Source != current.Source {
		return sourceRank(candidate.Source) > sourceRank(current.Source)
	}
	// exact beats glob
	if candidate.Exact != current.Exact {
		return candidate.Exact
	}
	return false
}

func sourceRank(s RuleSource) int {
	switch s {
	case SourceGovernance:
		return 4
	case SourceUserAllow, SourceUserDeny:
		return 3
	case SourceHook:
		return 2
	default:
		return 1
	}
}

// Grants converts to the legacy Grant slice for compatibility.
func (r *RememberedRules) Grants() []Grant {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Grant, 0, len(r.rules))
	for _, g := range r.rules {
		out = append(out, Grant{
			Tool:    g.Tool,
			Pattern: g.Pattern,
			Exact:   g.Exact,
			Allow:   g.Allow,
			Source:  g.Source,
			Label:   g.Label,
		})
	}
	return out
}