// churn.go derives a per-file churn score from git history — how many of the
// most recent commits touched each file — and exposes it as a bounded rank
// multiplier applied on top of the import-graph PageRank. Files that change
// often are where work actually happens; the map should surface them.
//
// Everything here degrades gracefully: outside a git repo, without a git
// binary, or on any git error the churn signal is simply empty and every
// multiplier stays at 1.0, leaving the pure PageRank ordering untouched.
package repomap

import (
	"math"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// DefaultChurnCommits caps the git history scan used for churn weighting.
// It bounds both the subprocess cost and the amount of history considered:
// recent activity matters, ancient history does not.
const DefaultChurnCommits = 500

// churnBoost bounds how much churn can amplify a file's PageRank. The
// most-changed file in the window gets a multiplier of 1+churnBoost; files
// with no history keep 1.0.
const churnBoost = 1.0

// GitChurn returns, for each file under root, how many of the last
// commitLimit commits touched it. Paths are slash-separated and relative to
// root. It returns nil when root is not a git repo, git is unavailable, or
// git fails for any other reason (callers treat that as "no churn signal").
func GitChurn(root string, commitLimit int) map[string]int {
	if commitLimit <= 0 {
		commitLimit = DefaultChurnCommits
	}
	prefix, ok := gitScanPrefix(root)
	if !ok {
		return nil
	}
	cmd := exec.Command("git", "log", "--name-only", "--pretty=format:", "-n", strconv.Itoa(commitLimit)) // #nosec G204 -- fixed git binary and flags; commitLimit is a locally derived int rendered via strconv
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		// Non-git dir, no commits yet, or git unavailable: churn stays off.
		return nil
	}
	return parseChurnPaths(string(out), prefix)
}

// gitScanPrefix returns the slash-separated path from the git repo root down
// to root ("" when root is the repo root), so that repo-root-relative git
// output can be re-keyed to the scanner's root-relative paths. ok is false
// when root is outside any git repo.
func gitScanPrefix(root string) (prefix string, ok bool) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel") // #nosec G204 -- fixed git binary and flags
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}
	toplevel := strings.TrimSpace(string(out))
	if toplevel == "" {
		return "", false
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", false
	}
	rel, relErr := filepath.Rel(toplevel, absRoot)
	if relErr != nil || strings.HasPrefix(rel, "..") {
		// Retry with symlinks resolved: git reports the real path while the
		// caller may hold a symlinked one (macOS temp dirs, /tmp links).
		if realRoot, rerr := filepath.EvalSymlinks(absRoot); rerr == nil {
			rel, relErr = filepath.Rel(toplevel, realRoot)
		}
		if relErr != nil || strings.HasPrefix(rel, "..") {
			return "", false
		}
	}
	rel = normalizePath(rel)
	if rel == "." || rel == "" {
		return "", true
	}
	return rel, true
}

// parseChurnPaths counts occurrences of each file path in a
// `git log --name-only --pretty=format:` transcript. When prefix is not
// empty, paths outside it are dropped and the remainder is re-keyed relative
// to the scan root. Rename lines ("old => new" or brace forms) are skipped
// rather than guessed at; a renamed file simply earns no churn credit in
// that commit.
func parseChurnPaths(out, prefix string) map[string]int {
	counts := make(map[string]int)
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(line, "=>") {
			continue
		}
		if prefix != "" {
			if !strings.HasPrefix(line, prefix+"/") {
				continue
			}
			line = line[len(prefix)+1:]
		}
		counts[normalizePath(line)]++
	}
	return counts
}

// ChurnMultipliers converts raw per-file commit counts into rank multipliers
// in [1, 1+churnBoost]: the most-changed file earns the largest boost and
// files absent from counts implicitly stay at 1.0. It returns nil when there
// is no usable signal.
func ChurnMultipliers(counts map[string]int) map[string]float64 {
	if len(counts) == 0 {
		return nil
	}
	maxCount := 0
	for _, c := range counts {
		if c > maxCount {
			maxCount = c
		}
	}
	if maxCount <= 0 {
		return nil
	}
	multipliers := make(map[string]float64, len(counts))
	for p, c := range counts {
		if c <= 0 {
			continue
		}
		multipliers[p] = 1 + churnBoost*(float64(c)/float64(maxCount))
	}
	return multipliers
}

// ApplyChurn multiplies each node's PageRank by its churn multiplier, so
// frequently-edited files float up in rankedFiles and Render. Nodes without
// an entry — or with a non-positive or non-finite one — keep their rank
// unchanged. Call it after Rank.
func (g *Graph) ApplyChurn(multipliers map[string]float64) {
	for p, mult := range multipliers {
		if !usableMultiplier(mult) {
			continue
		}
		if n := g.Nodes[p]; n != nil {
			n.Rank *= mult
		}
	}
}

func usableMultiplier(m float64) bool {
	return m > 0 && !math.IsNaN(m) && !math.IsInf(m, 0)
}
