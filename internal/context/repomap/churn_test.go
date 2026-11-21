package repomap

import (
	"context"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// requireGit skips the test when git is not installed, so the suite stays
// hermetic on minimal CI images.
func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git not available: %v", err)
	}
}

// initGitRepo creates a git repo at root with a CI-safe identity config
// (same pattern as internal/multiagent tests: no global git identity on
// runners, GPG signing off).
func initGitRepo(t *testing.T, root string) {
	t.Helper()
	for _, args := range [][]string{
		{"git", "init", "--initial-branch", "main", root},
		{"git", "-C", root, "config", "user.email", "reflect-test@example.com"},
		{"git", "-C", root, "config", "user.name", "reflect test"},
		{"git", "-C", root, "config", "commit.gpgsign", "false"},
	} {
		if out, err := exec.CommandContext(context.Background(), args[0], args[1:]...).CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %v\n%s", args, err, out)
		}
	}
}

// gitCommitAll commits every change under root with the given message.
func gitCommitAll(t *testing.T, root, msg string) {
	t.Helper()
	for _, args := range [][]string{
		{"git", "-C", root, "add", "-A"},
		{"git", "-C", root, "commit", "-m", msg},
	} {
		if out, err := exec.CommandContext(context.Background(), args[0], args[1:]...).CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %v\n%s", args, err, out)
		}
	}
}

// writeGitFile overwrites (or creates) rel under root and returns its full path.
func writeGitFile(t *testing.T, root, rel, content string) string {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
	return full
}

func TestGitChurn(t *testing.T) {
	requireGit(t)

	tests := []struct {
		name        string
		files       map[string]string // initial tree
		touches     [][]string        // per commit: files to modify (after the initial commit)
		commitLimit int
		scanSubdir  string // empty = scan the repo root
		want        map[string]int
	}{
		{
			name: "counts commits per file",
			files: map[string]string{
				"hub/hub.go":   "package hub\nfunc Connect() {}\n",
				"a/a.go":       "package a\nfunc A() {}\n",
				"z/z.go":       "package z\nfunc Z() {}\n",
				"plain/notgo":  "not source",
				"plain/txt.md": "# doc",
			},
			touches: [][]string{
				{"hub/hub.go", "a/a.go"},
				{"hub/hub.go"},
				{"plain/txt.md"},
			},
			want: map[string]int{
				"hub/hub.go":   3, // initial commit + 2 touches
				"a/a.go":       2,
				"z/z.go":       1,
				"plain/notgo":  1,
				"plain/txt.md": 2,
			},
		},
		{
			name: "commit limit only counts recent history",
			files: map[string]string{
				"one.go": "package main\nfunc One() {}\n",
				"two.go": "package main\nfunc Two() {}\n",
			},
			touches: [][]string{
				{"one.go"},
				{"one.go"},
				{"two.go"}, // newest commit
			},
			commitLimit: 1,
			want: map[string]int{
				"two.go": 1, // only the newest commit is counted
			},
		},
		{
			name: "subdir scan root re-keys paths",
			files: map[string]string{
				"top.go":        "package main\nfunc Top() {}\n",
				"sub/x.go":      "package sub\nfunc X() {}\n",
				"sub/deep/y.go": "package deep\nfunc Y() {}\n",
			},
			touches: [][]string{
				{"sub/x.go"},
				{"top.go"},
			},
			scanSubdir: "sub",
			want: map[string]int{
				"x.go":      2, // initial commit + 1 touch
				"deep/y.go": 1,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := writeTree(t, tc.files)
			initGitRepo(t, root)
			gitCommitAll(t, root, "initial")
			for i, commitFiles := range tc.touches {
				for _, rel := range commitFiles {
					// Unique content per touch so every commit is non-empty.
					writeGitFile(t, root, rel, "package x\n// touch "+strconv.Itoa(i)+"\n")
				}
				gitCommitAll(t, root, "touch "+string(rune('A'+i)))
			}

			scanRoot := root
			if tc.scanSubdir != "" {
				scanRoot = filepath.Join(root, tc.scanSubdir)
			}

			limit := tc.commitLimit
			if limit == 0 {
				limit = DefaultChurnCommits
			}
			got := GitChurn(scanRoot, limit)
			if len(got) != len(tc.want) {
				t.Fatalf("GitChurn = %v, want %v", got, tc.want)
			}
			for path, wantCount := range tc.want {
				if got[path] != wantCount {
					t.Errorf("GitChurn[%q] = %d, want %d (full: %v)", path, got[path], wantCount, got)
				}
			}
		})
	}
}

func TestGitChurn_Fallbacks(t *testing.T) {
	tests := []struct {
		name    string
		run     func(t *testing.T) map[string]int
		wantNil bool
	}{
		{
			name: "non-git directory yields nil",
			run: func(t *testing.T) map[string]int {
				root := writeTree(t, map[string]string{"a.go": "package a\n"})
				return GitChurn(root, DefaultChurnCommits)
			},
			wantNil: true,
		},
		{
			name: "git repo without commits yields nil",
			run: func(t *testing.T) map[string]int {
				requireGit(t)
				root := writeTree(t, map[string]string{"a.go": "package a\n"})
				initGitRepo(t, root)
				return GitChurn(root, DefaultChurnCommits)
			},
			wantNil: true,
		},
		{
			name: "non-positive commit limit falls back to default",
			run: func(t *testing.T) map[string]int {
				requireGit(t)
				root := writeTree(t, map[string]string{"a.go": "package a\n"})
				initGitRepo(t, root)
				gitCommitAll(t, root, "initial")
				got := GitChurn(root, -1)
				if got == nil {
					t.Fatal("expected counts for valid repo with default limit")
				}
				if got["a.go"] != 1 {
					t.Errorf("a.go = %d, want 1", got["a.go"])
				}
				return got
			},
			wantNil: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.run(t)
			if tc.wantNil && got != nil {
				t.Errorf("expected nil churn, got %v", got)
			}
		})
	}
}

func TestParseChurnPaths(t *testing.T) {
	tests := []struct {
		name   string
		out    string
		prefix string
		want   map[string]int
	}{
		{
			name: "counts each mention",
			out:  "a.go\nb.go\n\na.go\n",
			want: map[string]int{"a.go": 2, "b.go": 1},
		},
		{
			name:   "strips scan-root prefix",
			out:    "sub/x.go\nother/o.go\nsub/deep/y.go\n",
			prefix: "sub",
			want:   map[string]int{"x.go": 1, "deep/y.go": 1},
		},
		{
			name: "rename lines are skipped",
			out:  "old.go => new.go\nsub/{a.go => b.go}\nnew.go\n",
			want: map[string]int{"new.go": 1},
		},
		{
			name: "backslashes normalize",
			out:  "win\\sub\\a.go\n",
			want: map[string]int{"win/sub/a.go": 1},
		},
		{
			name: "empty transcript",
			out:  "\n\n",
			want: map[string]int{},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseChurnPaths(tc.out, tc.prefix)
			if len(got) != len(tc.want) {
				t.Fatalf("parseChurnPaths = %v, want %v", got, tc.want)
			}
			for path, wantCount := range tc.want {
				if got[path] != wantCount {
					t.Errorf("parseChurnPaths[%q] = %d, want %d", path, got[path], wantCount)
				}
			}
		})
	}
}

func TestChurnMultipliers(t *testing.T) {
	tests := []struct {
		name   string
		counts map[string]int
		want   map[string]float64
	}{
		{
			name:   "no signal yields nil",
			counts: nil,
			want:   nil,
		},
		{
			name:   "all-zero counts yield nil",
			counts: map[string]int{"a.go": 0},
			want:   nil,
		},
		{
			name:   "single file gets max boost",
			counts: map[string]int{"a.go": 7},
			want:   map[string]float64{"a.go": 2.0},
		},
		{
			name: "scaled between 1 and max",
			counts: map[string]int{
				"hot.go":  8,
				"warm.go": 4,
				"zero.go": 0,
			},
			want: map[string]float64{
				"hot.go":  2.0,
				"warm.go": 1.5,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ChurnMultipliers(tc.counts)
			if tc.want == nil {
				if got != nil {
					t.Fatalf("ChurnMultipliers = %v, want nil", got)
				}
				return
			}
			if len(got) != len(tc.want) {
				t.Fatalf("ChurnMultipliers = %v, want %v", got, tc.want)
			}
			for path, wantMult := range tc.want {
				if abs(got[path]-wantMult) > 1e-9 {
					t.Errorf("ChurnMultipliers[%q] = %v, want %v", path, got[path], wantMult)
				}
			}
		})
	}
}

func TestApplyChurn(t *testing.T) {
	g := &Graph{Nodes: map[string]*FileNode{
		"a.go": {Path: "a.go", Rank: 0.5},
		"b.go": {Path: "b.go", Rank: 0.5},
		"c.go": {Path: "c.go", Rank: 0.5},
		"d.go": {Path: "d.go", Rank: 0.5},
	}}

	g.ApplyChurn(map[string]float64{
		"a.go":    2.0,
		"b.go":    1.25,
		"c.go":    0,   // non-positive: ignored
		"nope.go": 9.9, // not a node: ignored
	})

	if abs(g.Nodes["a.go"].Rank-1.0) > 1e-9 {
		t.Errorf("a.go rank = %v, want 1.0 (0.5 * 2.0)", g.Nodes["a.go"].Rank)
	}
	if want := 0.625; abs(g.Nodes["b.go"].Rank-want) > 1e-9 {
		t.Errorf("b.go rank = %v, want %v", g.Nodes["b.go"].Rank, want)
	}
	for _, unchanged := range []string{"c.go", "d.go"} {
		if abs(g.Nodes[unchanged].Rank-0.5) > 1e-9 {
			t.Errorf("%s rank = %v, want 0.5", unchanged, g.Nodes[unchanged].Rank)
		}
	}

	// Non-finite multipliers must be ignored too.
	g2 := &Graph{Nodes: map[string]*FileNode{"a.go": {Path: "a.go", Rank: 1}}}
	g2.ApplyChurn(map[string]float64{"a.go": math.NaN(), "b.go": math.Inf(1)})
	if g2.Nodes["a.go"].Rank != 1 {
		t.Errorf("NaN multiplier changed rank: %v", g2.Nodes["a.go"].Rank)
	}
}

// churnGraph builds the classic fixture: hub imported by a and b, plus two
// lonely packages, one of which is a churn magnet.
func churnGraph() map[string]string {
	return map[string]string{
		"hub/hub.go": `package hub

func Connect() {}
`,
		"a/a.go": `package a

import "example.com/m/hub"

func UseA() { hub.Connect() }
`,
		"b/b.go": `package b

import "example.com/m/hub"

func UseB() { hub.Connect() }
`,
		"lonely/a.go": `package lonely

func Cold() {}
`,
		"lonely/z.go": `package lonely

func Hot() {}
`,
	}
}

func TestRank_WithChurnBoostsHotFiles(t *testing.T) {
	requireGit(t)

	root := writeTree(t, churnGraph())
	initGitRepo(t, root)
	gitCommitAll(t, root, "initial")
	for i := 0; i < 3; i++ {
		writeGitFile(t, root, "lonely/z.go", "package lonely\n// touch "+strconv.Itoa(i)+"\n")
		gitCommitAll(t, root, "touch z")
	}

	g, err := Build(root)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	g.Rank()

	// Sanity: without churn the two lonely files tie, and the deterministic
	// path tiebreak puts a.go first.
	ranked := g.rankedFiles()
	var lonelyA, lonelyZ int
	for i, n := range ranked {
		switch n.Path {
		case "lonely/a.go":
			lonelyA = i
		case "lonely/z.go":
			lonelyZ = i
		}
	}
	if lonelyA > lonelyZ {
		t.Fatalf("precondition failed: lonely/a.go should outrank lonely/z.go before churn (a=%d z=%d)", lonelyA, lonelyZ)
	}

	g.ApplyChurn(ChurnMultipliers(GitChurn(root, DefaultChurnCommits)))
	ranked = g.rankedFiles()

	// hub stays on top (structure still dominates), but the churned lonely
	// file must now outrank its untouched sibling.
	if ranked[0].Path != "hub/hub.go" {
		t.Errorf("hub/hub.go should stay on top, got %s", ranked[0].Path)
	}
	for _, n := range ranked {
		if n.Path == "lonely/z.go" {
			break
		}
		if n.Path == "lonely/a.go" {
			t.Errorf("lonely/z.go should outrank lonely/a.go after churn:\n%v", rankedPaths(ranked))
		}
	}
}

func TestRepoMap_ChurnChangesTightBudgetOrdering(t *testing.T) {
	// Two isolated files with identical PageRank: z.go is a churn magnet,
	// a.go is untouched after the initial commit. Under a budget that fits
	// only one file, the git repo's map leads with the churned file while
	// the non-git fallback keeps the deterministic path ordering.
	files := map[string]string{
		"a.go": "package main\nfunc Cold() {}\n",
		"z.go": "package main\nfunc Hot() {}\n",
	}

	t.Run("git repo prefers churned file", func(t *testing.T) {
		requireGit(t)
		root := writeTree(t, files)
		initGitRepo(t, root)
		gitCommitAll(t, root, "initial")
		for i := 0; i < 2; i++ {
			writeGitFile(t, root, "z.go", "package main\n// touch "+strconv.Itoa(i)+"\n")
			gitCommitAll(t, root, "touch z")
		}

		out, err := RepoMap(root, 6)
		if err != nil {
			t.Fatalf("RepoMap: %v", err)
		}
		if !strings.HasPrefix(out, "z.go") {
			t.Errorf("tight-budget map should lead with churned z.go, got:\n%s", out)
		}
	})

	t.Run("non-git fallback keeps path ordering", func(t *testing.T) {
		root := writeTree(t, files)
		out, err := RepoMap(root, 6)
		if err != nil {
			t.Fatalf("RepoMap: %v", err)
		}
		if !strings.HasPrefix(out, "a.go") {
			t.Errorf("non-git tight-budget map should lead with a.go, got:\n%s", out)
		}
	})
}

func rankedPaths(nodes []*FileNode) []string {
	paths := make([]string, len(nodes))
	for i, n := range nodes {
		paths[i] = n.Path
	}
	return paths
}
