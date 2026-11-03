package tool

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

func runGrepFixture(t *testing.T, ctx context.Context, dir, pattern string) (string, error) {
	t.Helper()
	input, err := json.Marshal(GrepInput{Path: dir, Pattern: pattern})
	if err != nil {
		t.Fatal(err)
	}
	return (GrepTool{}).Execute(WithToolContext(ctx, &ToolContext{WorkingDir: dir}), input)
}

func TestGrepExcludesSensitiveFilesAndLinks(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{".env", ".env.production", "credentials.json", "id_rsa"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("needle-secret"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "source.go"), []byte("package main\n// needle-source\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dir, ".env"), filepath.Join(dir, "readme.txt")); err != nil {
		t.Fatal(err)
	}
	out, err := runGrepFixture(t, t.Context(), dir, "needle")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "needle-secret") || !strings.Contains(out, "source.go:2: // needle-source") {
		t.Fatalf("search leaked sensitive data or lost source matches: %q", out)
	}
	if !strings.Contains(out, "sensitive") {
		t.Fatalf("missing skipped-file notice: %q", out)
	}
}

func TestGrepSkipsBinaryOversizedAndLongLineFiles(t *testing.T) {
	dir := t.TempDir()
	for name, data := range map[string]string{"binary": "\x7fELF\x00needle-binary", "long.txt": strings.Repeat("x", maxGrepLineBytes) + "needle-long", "source.go": "// needle-source\n"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	f, err := os.Create(filepath.Join(dir, "large.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("needle-large"); err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(maxFileSize + 1); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	out, err := runGrepFixture(t, t.Context(), dir, "needle")
	if err != nil {
		t.Fatal(err)
	}
	for _, unexpected := range []string{"needle-binary", "needle-large", "needle-long"} {
		if strings.Contains(out, unexpected) {
			t.Errorf("returned skipped file contents: %q", out)
		}
	}
	for _, want := range []string{"needle-source", "1 binary", "1 oversized", "1 with long lines"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in %q", want, out)
		}
	}
}

func TestGrepCancellation(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "source.go"), []byte(strings.Repeat("needle\n", 100)), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if out, err := runGrepFixture(t, ctx, dir, "needle"); !errors.Is(err, context.Canceled) || out != "" {
		t.Fatalf("cancelled search returned %q, %v", out, err)
	}
	// Cancel once a scan is underway, rather than only before Execute.
	ctx, cancel = context.WithCancel(t.Context())
	defer cancel()
	counted := &grepCancelContext{Context: ctx, cancel: cancel, remaining: 8}
	if out, err := runGrepFixture(t, counted, dir, "needle"); !errors.Is(err, context.Canceled) || out != "" {
		t.Fatalf("search ignored cancellation during scan: %q, %v", out, err)
	}
}

type grepCancelContext struct {
	context.Context
	cancel    context.CancelFunc
	remaining int
}

func (c *grepCancelContext) Err() error {
	c.remaining--
	if c.remaining == 0 {
		c.cancel()
	}
	return c.Context.Err()
}

func TestGrepBoundsMatchCountAndOutput(t *testing.T) {
	for _, tc := range []struct {
		name, text string
		want       int
	}{
		{name: "match count", text: strings.Repeat("needle\n", maxGrepMatches+10), want: maxGrepMatches},
		{name: "output bytes", text: strings.Repeat("needle "+strings.Repeat("界", 2000)+"\n", 20)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "source.go"), []byte(tc.text), 0o600); err != nil {
				t.Fatal(err)
			}
			out, err := runGrepFixture(t, t.Context(), dir, "needle")
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out, "Results truncated") || len(out) > maxGrepOutputBytes+256 || !utf8.ValidString(out) {
				t.Fatalf("unbounded or invalid output: bytes=%d, tail=%q", len(out), out[max(0, len(out)-100):])
			}
			if tc.want > 0 && strings.Count(out, "needle") != tc.want {
				t.Fatalf("matches=%d, want %d", strings.Count(out, "needle"), tc.want)
			}
		})
	}
}
