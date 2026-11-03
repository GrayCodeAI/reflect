package tool

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	maxGrepMatches      = 200
	maxGrepOutputBytes  = 32 << 10
	maxGrepLineBytes    = 64 << 10
	maxGrepPreviewBytes = 4 << 10
)

type GrepTool struct{}

func (GrepTool) Name() string      { return "Grep" }
func (GrepTool) RiskLevel() string { return "low" }
func (GrepTool) Aliases() []string { return []string{"grep"} }
func (GrepTool) Description() string {
	return "Search for a regex pattern in text files. Skips sensitive files, binaries, and files larger than 10 MiB. Results are bounded; narrow the pattern or use include for more specific results."
}

// GrepInput is the typed input for GrepTool.
type GrepInput struct {
	Pattern string `json:"pattern"`
	Path    string `json:"path"`
	Include string `json:"include"`
}

// Schema returns the typed input schema. Parameters() delegates to it so the
// two cannot diverge.
func (GrepTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "object",
		Properties: map[string]SchemaProperty{
			"pattern": {Type: "string", Description: "Regex pattern to search for"},
			"path":    {Type: "string", Description: "Directory to search (default: current dir)"},
			"include": {Type: "string", Description: "File glob filter (e.g. *.go)"},
		},
		Required: []string{"pattern"},
	}
}

func (GrepTool) Parameters() map[string]interface{} {
	return grepSchema.ToJSONSchema()
}

// grepSchema is the single source of truth for Grep's input schema.
var grepSchema = GrepTool{}.Schema()

func (GrepTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	p, err := DecodeInput[GrepInput]("Grep", input)
	if err != nil {
		return "", err
	}
	re, err := regexp.Compile(p.Pattern)
	if err != nil {
		return "", fmt.Errorf("invalid regex: %w", err)
	}
	root := p.Path
	if root == "" {
		root = "."
	}
	root = resolveToolPath(ctx, root)
	if err := validatePathAllowed(ctx, root); err != nil {
		return "", err
	}
	rootAbs, err := guardedAbs(root)
	if err != nil {
		return "", err
	}
	if reason := IsSensitivePath(rootAbs); reason != "" {
		return "", fmt.Errorf("blocked: %s", reason)
	}
	rootHandle, err := os.OpenRoot(rootAbs)
	if err != nil {
		return "", fmt.Errorf("open search root: %w", err)
	}
	defer func() { _ = rootHandle.Close() }()

	results := grepResults{}
	skipped := map[string]int{}
	err = fs.WalkDir(rootHandle.FS(), ".", func(path string, d fs.DirEntry, err error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err != nil {
			skipped["unreadable"]++
			return nil
		}
		fullPath := filepath.Join(rootAbs, filepath.FromSlash(path))
		if IsSensitivePath(fullPath) != "" {
			skipped["sensitive"]++
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		// Do not follow links or open devices, sockets, or named pipes.
		if !d.Type().IsRegular() {
			return nil
		}
		if p.Include != "" {
			if matched, _ := filepath.Match(p.Include, d.Name()); !matched {
				return nil
			}
		}
		reason, err := grepFile(ctx, rootHandle, path, fullPath, re, &results)
		if err != nil {
			return err
		}
		if reason != "" {
			skipped[reason]++
		}
		if results.truncated {
			return fs.SkipAll
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	out := strings.Join(results.lines, "\n")
	if len(results.lines) == 0 {
		out = "No matches found"
	}
	if results.truncated {
		out += "\n[Results truncated; narrow the pattern or use include.]"
	}
	var notes []string
	for _, reason := range []string{"sensitive", "binary", "oversized", "unreadable", "with long lines"} {
		if count := skipped[reason]; count > 0 {
			notes = append(notes, fmt.Sprintf("%d %s", count, reason))
		}
	}
	if len(notes) > 0 {
		out += "\n[Skipped files: " + strings.Join(notes, ", ") + ".]"
	}
	return out, nil
}

type grepResults struct {
	lines     []string
	bytes     int
	truncated bool
}

func (r *grepResults) add(path string, line int, text string) {
	entry := fmt.Sprintf("%s:%d: %s", path, line, grepPreview(text))
	if r.bytes+len(entry)+1 > maxGrepOutputBytes {
		r.truncated = true
		return
	}
	r.lines = append(r.lines, entry)
	r.bytes += len(entry) + 1
	r.truncated = len(r.lines) >= maxGrepMatches
}

func grepPreview(text string) string {
	text = strings.ToValidUTF8(text, "�")
	if len(text) <= maxGrepPreviewBytes {
		return text
	}
	end := maxGrepPreviewBytes - len("…")
	for end > 0 && !utf8.RuneStart(text[end]) {
		end--
	}
	return text[:end] + "…"
}

func grepFile(ctx context.Context, root *os.Root, path, displayPath string, re *regexp.Regexp, results *grepResults) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	f, err := root.Open(path)
	if err != nil {
		return "unreadable", nil
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return "unreadable", nil
	}
	// Reject a link swapped in after the walk and sensitive-path checks.
	entry, err := root.Lstat(path)
	if err != nil || !os.SameFile(info, entry) {
		return "unreadable", nil
	}
	if info.Size() > maxFileSize {
		return "oversized", nil
	}
	// Bound the read even if the file grows after Stat.
	data, err := io.ReadAll(io.LimitReader(f, maxFileSize+1))
	if ctxErr := ctx.Err(); ctxErr != nil {
		return "", ctxErr
	}
	if err != nil {
		return "unreadable", nil
	}
	if len(data) > maxFileSize {
		return "oversized", nil
	}
	if IsBinaryContent(data) {
		return "binary", nil
	}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 4096), maxGrepLineBytes)
	for line := 1; scanner.Scan(); line++ {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		if re.Match(scanner.Bytes()) {
			results.add(displayPath, line, scanner.Text())
			if results.truncated {
				return "", nil
			}
		}
	}
	if scanner.Err() != nil {
		return "with long lines", nil
	}
	return "", nil
}
