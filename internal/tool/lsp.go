package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"

	"github.com/GrayCodeAI/reflect/internal/codegraph"
	"github.com/GrayCodeAI/reflect/internal/lsp"
)

type LSPTool struct {
	Manager *lsp.LSPManager
}

// LSPInput is the typed input for LSPTool.
type LSPInput struct {
	Action  string `json:"action"`
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Symbol  string `json:"symbol"`
	NewName string `json:"new_name"`
	Root    string `json:"root"`
}

func (LSPTool) Name() string      { return "LSP" }
func (LSPTool) Aliases() []string { return []string{"lsp"} }
func (LSPTool) Description() string {
	return "Get code intelligence through configured language servers, with codegraph and local-tool fallbacks. " +
		"The rename action renames a symbol workspace-wide through the language server; prepare_rename previews the change without writing."
}

// Schema returns the typed input schema. Parameters() delegates to it so the
// two cannot diverge.
func (LSPTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "object",
		Properties: map[string]SchemaProperty{
			"action":   {Type: "string", Enum: []interface{}{"diagnostics", "definition", "references", "implementations", "prepare_rename", "rename"}, Description: "LSP action"},
			"path":     {Type: "string", Description: "File path"},
			"line":     {Type: "integer", Description: "Line number (1-based)"},
			"column":   {Type: "integer", Description: "Column number (1-based)"},
			"symbol":   {Type: "string", Description: "Symbol name to look up (alternative to line/column); for rename/prepare_rename the current name, used to auto-locate and label the preview"},
			"new_name": {Type: "string", Description: "New symbol name (required for rename/prepare_rename)"},
			"root":     {Type: "string", Description: "Project root directory (default: current dir)"},
		},
		Required: []string{"action", "path"},
	}
}

func (LSPTool) Parameters() map[string]interface{} {
	return lspSchema.ToJSONSchema()
}

// lspSchema is the single source of truth for LSP's input schema.
var lspSchema = LSPTool{}.Schema()

func (t LSPTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	p, err := DecodeInput[LSPInput]("LSP", input)
	if err != nil {
		return "", err
	}

	root := p.Root
	if root == "" {
		root = "."
	}
	// rename/prepare_rename mutate files and codegraph cannot back them, so
	// they are dispatched before the language-server fallback below.
	switch p.Action {
	case "rename", "prepare_rename":
		return t.executeRename(ctx, p)
	}
	if t.Manager != nil {
		if result, ok := t.executeLanguageServer(ctx, p.Action, p.Path, p.Line, p.Column); ok {
			return result, nil
		}
	}

	switch p.Action {
	case "diagnostics":
		return lspDiagnostics(ctx, p.Path)
	case "definition":
		return lspDefinition(root, p.Path, p.Line, p.Symbol)
	case "references":
		return lspReferences(root, p.Path, p.Line, p.Symbol)
	case "implementations":
		return lspImplementations(root, p.Symbol)
	case "rename", "prepare_rename":
		// Unreachable (handled above), kept so reordering the dispatch cannot
		// silently turn a rename into a codegraph lookup.
		return "", fmt.Errorf("%s requires a language server; use the Refactor tool's rename_symbol action for a per-file regex fallback", p.Action)
	default:
		return "", fmt.Errorf("unknown LSP action: %s", p.Action)
	}
}

func (t LSPTool) executeLanguageServer(ctx context.Context, action, path string, line, column int) (string, bool) {
	lang, _, ok := t.Manager.Config().ServerForFile(path)
	if !ok || path == "" {
		return "", false
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", false
	}
	text, err := os.ReadFile(absPath) // #nosec G304 -- workspace path supplied to the LSP tool
	if err != nil {
		return "", false
	}
	if line < 1 {
		line = 1
	}
	if column < 1 {
		column = 1
	}
	uri := (&url.URL{Scheme: "file", Path: filepath.ToSlash(absPath)}).String()
	var result string
	err = t.Manager.Execute(ctx, lang, action != "rename", func(client *lsp.LSPClient) error {
		if err := client.DidOpen(ctx, uri, lang, 1, string(text)); err != nil {
			return err
		}
		defer func() { _ = client.DidClose(context.Background(), uri) }()
		var err error
		switch action {
		case "diagnostics":
			var items []lsp.Diagnostic
			items, err = client.Diagnostics(ctx, uri)
			result = formatServerDiagnostics(items)
		case "definition":
			var locations []lsp.Location
			locations, err = client.GotoDefinition(ctx, uri, line-1, column-1)
			result = formatServerLocations("Definitions", locations)
		case "references":
			var locations []lsp.Location
			locations, err = client.FindReferences(ctx, uri, line-1, column-1)
			result = formatServerLocations("References", locations)
		case "symbols":
			var symbols []lsp.SymbolInformation
			symbols, err = client.DocumentSymbol(ctx, uri)
			result = formatServerSymbols(symbols)
		default:
			return fmt.Errorf("unsupported language-server action %q", action)
		}
		return err
	})
	if err != nil {
		return "", false
	}
	return result, true
}

func formatServerDiagnostics(items []lsp.Diagnostic) string {
	if len(items) == 0 {
		return "No diagnostics found."
	}
	var b strings.Builder
	b.WriteString("## Diagnostics\n\n")
	for _, item := range items {
		fmt.Fprintf(&b, "- %s:%d:%d [%d] %s\n", item.Source, item.Range.Start.Line+1, item.Range.Start.Character+1, item.Severity, item.Message)
	}
	return strings.TrimRight(b.String(), "\n")
}

func formatServerLocations(title string, locations []lsp.Location) string {
	if len(locations) == 0 {
		return "No " + strings.ToLower(title) + " found."
	}
	var b strings.Builder
	fmt.Fprintf(&b, "## %s\n\n", title)
	for _, location := range locations {
		fmt.Fprintf(&b, "- %s:%d:%d\n", strings.TrimPrefix(location.URI, "file://"), location.Range.Start.Line+1, location.Range.Start.Character+1)
	}
	return strings.TrimRight(b.String(), "\n")
}

func formatServerSymbols(symbols []lsp.SymbolInformation) string {
	if len(symbols) == 0 {
		return "No symbols found."
	}
	var b strings.Builder
	b.WriteString("## Symbols\n\n")
	for _, symbol := range symbols {
		fmt.Fprintf(&b, "- %s at %d:%d\n", symbol.Name, symbol.Location.Range.Start.Line+1, symbol.Location.Range.Start.Character+1)
	}
	return strings.TrimRight(b.String(), "\n")
}

// ─── rename (workspace-wide, language-server backed) ───────────────────────────

// lspRenameFileEdits counts the edits a rename applies to one file.
type lspRenameFileEdits struct {
	Path  string
	Edits int
}

// lspRenameOutcome describes a language-server rename before it is applied.
type lspRenameOutcome struct {
	OldName string
	NewName string
	Files   int
	Edits   int
	PerFile []lspRenameFileEdits
	Edit    *lsp.WorkspaceEdit
}

// executeRename implements the rename and prepare_rename actions. Both go
// through the language server: PrepareRename validates that the position is
// renamable (and yields the old name), Rename resolves the WorkspaceEdit.
// prepare_rename is the dry-run form: it stops at the preview and never
// writes. rename applies the edit through the same guarded, transactional
// pipeline as the other edit tools, so the normal permission gates apply.
func (t LSPTool) executeRename(ctx context.Context, p LSPInput) (string, error) {
	if p.Path == "" {
		return "", fmt.Errorf("path is required")
	}
	path := resolveToolPath(ctx, p.Path)
	if err := validatePathAllowed(ctx, path); err != nil {
		return "", err
	}
	if reason := IsSensitivePath(path); reason != "" {
		return "", fmt.Errorf("blocked: %s", reason)
	}
	if p.NewName == "" {
		return "", fmt.Errorf("new_name is required for %s", p.Action)
	}
	if (p.Line < 1 || p.Column < 1) && p.Symbol == "" {
		return "", fmt.Errorf("line and column (1-based) are required for %s (or pass symbol to auto-locate its first occurrence)", p.Action)
	}

	// readOnly marks the request as safe to retry on server crash; it mirrors
	// the action != "rename" convention used by executeLanguageServer.
	readOnly := p.Action == "prepare_rename"
	outcome, err := lspRenameEdits(ctx, t.Manager, path, p.Line, p.Column, p.Symbol, p.NewName, readOnly)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteString(formatRenamePreview(outcome))
	if !readOnly {
		summary, err := applyWorkspaceEdit(ctx, outcome.Edit)
		if err != nil {
			return "", err
		}
		b.WriteString("\n\n" + summary)
		return b.String(), nil
	}
	b.WriteString("\n\nDry run only — no files were modified. Re-run with action \"rename\" to apply.")
	return b.String(), nil
}

// lspRenameEdits resolves a rename through the language server configured for
// path and computes the WorkspaceEdit without writing anything; the caller
// applies (or only previews) it via applyWorkspaceEdit. line/column are
// 1-based and optional when oldHint is set: the first word-boundary
// occurrence of oldHint in the file is used as the position.
//
// Capability gating: the LSP client does not retain the server's initialize
// capabilities (RenameCapabilities is never populated by internal/lsp), so the
// gate is behavioral — a server without rename support (or a position that is
// not renamable) answers textDocument/rename with a null/error payload that
// decodes to an empty WorkspaceEdit, which is turned into a clear error
// recommending the Refactor tool's per-file regex fallback.
func lspRenameEdits(ctx context.Context, mgr *lsp.LSPManager, path string, line, column int, oldHint, newName string, readOnly bool) (*lspRenameOutcome, error) {
	const fallbackHint = "use the Refactor tool's rename_symbol action for a per-file regex fallback"
	if newName == "" {
		return nil, fmt.Errorf("lsp rename: new_name is required")
	}
	if mgr == nil {
		return nil, fmt.Errorf("lsp rename: no language-server manager available; %s", fallbackHint)
	}
	cfg := mgr.Config()
	if cfg == nil {
		return nil, fmt.Errorf("lsp rename: language-server manager has no configuration; %s", fallbackHint)
	}
	lang, _, ok := cfg.ServerForFile(path)
	if !ok {
		return nil, fmt.Errorf("lsp rename: no language server configured for %s files; %s", filepath.Ext(path), fallbackHint)
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("lsp rename: resolve path: %w", err)
	}
	text, err := os.ReadFile(absPath) // #nosec G304 -- workspace path supplied to the LSP tool
	if err != nil {
		return nil, fmt.Errorf("lsp rename: read %s: %w", absPath, err)
	}
	if line < 1 || column < 1 {
		if oldHint == "" {
			return nil, fmt.Errorf("lsp rename: line and column (1-based) are required when symbol is not provided")
		}
		l, c, found := findSymbolPosition(string(text), oldHint)
		if !found {
			return nil, fmt.Errorf("lsp rename: symbol %q not found in %s; %s", oldHint, path, fallbackHint)
		}
		line, column = l, c
	}
	uri := (&url.URL{Scheme: "file", Path: filepath.ToSlash(absPath)}).String()
	outcome := &lspRenameOutcome{OldName: oldHint, NewName: newName}
	err = mgr.Execute(ctx, lang, readOnly, func(client *lsp.LSPClient) error {
		if err := client.DidOpen(ctx, uri, lang, 1, string(text)); err != nil {
			return fmt.Errorf("didOpen: %w", err)
		}
		defer func() { _ = client.DidClose(context.Background(), uri) }()
		// PrepareRename is the rename gate. Servers that support rename but
		// not prepareRename answer with a null/error payload that decodes to
		// a zero range; treat that as "no preview info" instead of aborting.
		prepare, err := client.PrepareRename(ctx, uri, line-1, column-1)
		if err != nil {
			return fmt.Errorf("prepare rename: %w", err)
		}
		if old, serr := oldNameFromRange(string(text), prepare); serr == nil && old != "" {
			outcome.OldName = old
		}
		edit, err := client.Rename(ctx, uri, line-1, column-1, newName)
		if err != nil {
			return fmt.Errorf("rename: %w", err)
		}
		outcome.Edit = edit
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("lsp rename via %s server: %w; %s", lang, err, fallbackHint)
	}
	if outcome.Edit == nil || len(outcome.Edit.Changes) == 0 {
		return nil, fmt.Errorf("the %s language server returned no rename edits for %s:%d:%d — the symbol may not be renamable at this position, or the server lacks rename support; %s", lang, path, line, column, fallbackHint)
	}
	if err := outcome.summarize(); err != nil {
		return nil, err
	}
	return outcome, nil
}

// summarize fills the Files/Edits/PerFile fields from the resolved edit.
func (o *lspRenameOutcome) summarize() error {
	if o.Edit == nil {
		return nil
	}
	uris := make([]string, 0, len(o.Edit.Changes))
	for uri := range o.Edit.Changes {
		uris = append(uris, uri)
	}
	sort.Strings(uris)
	for _, uri := range uris {
		edits := o.Edit.Changes[uri]
		if len(edits) == 0 {
			continue
		}
		path, err := pathFromFileURI(uri)
		if err != nil {
			return fmt.Errorf("lsp rename: %w", err)
		}
		o.PerFile = append(o.PerFile, lspRenameFileEdits{Path: path, Edits: len(edits)})
		o.Files++
		o.Edits += len(edits)
	}
	return nil
}

// formatRenamePreview renders the preview of a language-server rename: old
// name, new name, affected-file count and per-file edit counts.
func formatRenamePreview(o *lspRenameOutcome) string {
	var b strings.Builder
	b.WriteString("## Rename Preview\n\n")
	old := o.OldName
	if old == "" {
		old = "(unknown)"
	}
	fmt.Fprintf(&b, "- Old name: %q\n", old)
	fmt.Fprintf(&b, "- New name: %q\n", o.NewName)
	fmt.Fprintf(&b, "- Affected files: %d\n", o.Files)
	fmt.Fprintf(&b, "- Total edits: %d\n", o.Edits)
	if len(o.PerFile) > 0 {
		b.WriteString("\nEdits per file:\n")
		for _, f := range o.PerFile {
			fmt.Fprintf(&b, "- %s: %d edit(s)\n", f.Path, f.Edits)
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// applyWorkspaceEdit applies a language-server WorkspaceEdit through the same
// guarded pipeline as the other edit tools: every target path is resolved and
// permission-checked (allowed directories, sensitive paths, protected paths),
// the computed content is scanned for credentials, and the writes are staged
// into an atomic Transaction whose Commit rolls everything back if any file
// fails. It never writes files directly.
func applyWorkspaceEdit(ctx context.Context, edit *lsp.WorkspaceEdit) (string, error) {
	if edit == nil || len(edit.Changes) == 0 {
		return "", fmt.Errorf("workspace edit is empty; nothing to apply")
	}
	uris := make([]string, 0, len(edit.Changes))
	for uri := range edit.Changes {
		uris = append(uris, uri)
	}
	sort.Strings(uris)

	tx := NewTransaction()
	staged := 0
	totalEdits := 0
	for _, uri := range uris {
		edits := edit.Changes[uri]
		if len(edits) == 0 {
			continue
		}
		path, err := pathFromFileURI(uri)
		if err != nil {
			return "", err
		}
		if err := validatePathAllowed(ctx, path); err != nil {
			return "", err
		}
		if reason := IsSensitivePath(path); reason != "" {
			return "", fmt.Errorf("blocked: %s", reason)
		}
		if tc := GetToolContext(ctx); tc != nil && tc.Protected != nil && tc.Protected.IsProtected(path) {
			return "", fmt.Errorf("path %s is protected (read-only)", path)
		}
		data, err := readGuardedFile(ctx, path)
		if err != nil {
			return "", fmt.Errorf("read %s: %w", path, err)
		}
		newContent, err := applyTextEdits(string(data), edits)
		if err != nil {
			return "", fmt.Errorf("apply rename edits to %s: %w", path, err)
		}
		if cred := DetectCredentials(newContent); cred != "" {
			return "", fmt.Errorf("rename would write a credential (%s) into %s — refusing", cred, path)
		}
		if err := tx.AddModify(path, []byte(newContent)); err != nil {
			return "", fmt.Errorf("stage modify of %s: %w", path, err)
		}
		staged++
		totalEdits += len(edits)
	}
	if staged == 0 {
		return "", fmt.Errorf("workspace edit contains no file changes; nothing to apply")
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return fmt.Sprintf("Applied rename to %d file(s) (%d edit(s)); transaction %s committed.", staged, totalEdits, tx.ID), nil
}

// applyTextEdits applies LSP text edits to content. Positions are 0-based line
// and UTF-16 character offsets per the LSP specification. Edits are applied
// from the last position to the first so earlier offsets stay valid;
// overlapping edits are rejected.
func applyTextEdits(content string, edits []lsp.TextEdit) (string, error) {
	if len(edits) == 0 {
		return content, nil
	}
	starts := lineStartOffsets(content)
	type span struct {
		start, end int
		text       string
	}
	spans := make([]span, 0, len(edits))
	for i, e := range edits {
		start, err := positionOffset(starts, content, e.Range.Start)
		if err != nil {
			return "", fmt.Errorf("edit %d: %w", i, err)
		}
		end, err := positionOffset(starts, content, e.Range.End)
		if err != nil {
			return "", fmt.Errorf("edit %d: %w", i, err)
		}
		if end < start {
			return "", fmt.Errorf("edit %d: inverted range (start %v after end %v)", i, e.Range.Start, e.Range.End)
		}
		spans = append(spans, span{start: start, end: end, text: e.NewText})
	}
	sort.Slice(spans, func(i, j int) bool {
		if spans[i].start != spans[j].start {
			return spans[i].start > spans[j].start
		}
		return spans[i].end > spans[j].end
	})
	for i := 1; i < len(spans); i++ {
		if spans[i-1].start < spans[i].end {
			return "", fmt.Errorf("overlapping edits at byte offsets %d and %d", spans[i].start, spans[i-1].start)
		}
	}
	result := content
	for _, s := range spans {
		result = result[:s.start] + s.text + result[s.end:]
	}
	return result, nil
}

// lineStartOffsets returns the byte offset of the start of every 0-based line
// in content. The first line always starts at 0.
func lineStartOffsets(content string) []int {
	starts := []int{0}
	for i := 0; i < len(content); i++ {
		if content[i] == '\n' {
			starts = append(starts, i+1)
		}
	}
	return starts
}

// positionOffset converts a 0-based LSP position (UTF-16 character units)
// into a byte offset in content.
func positionOffset(lineStarts []int, content string, pos lsp.Position) (int, error) {
	if pos.Line < 0 {
		return 0, fmt.Errorf("negative line %d", pos.Line)
	}
	if pos.Line >= len(lineStarts) {
		return 0, fmt.Errorf("line %d out of range (content has %d lines)", pos.Line+1, len(lineStarts))
	}
	lineEnd := len(content)
	if pos.Line+1 < len(lineStarts) {
		lineEnd = lineStarts[pos.Line+1] - 1
	}
	// A CRLF line terminator is not part of the LSP line, so trim it before
	// mapping the character column.
	line := strings.TrimSuffix(content[lineStarts[pos.Line]:lineEnd], "\r")
	byteOff, err := utf16ColumnToByteOffset(line, pos.Character)
	if err != nil {
		return 0, fmt.Errorf("line %d: %w", pos.Line+1, err)
	}
	return lineStarts[pos.Line] + byteOff, nil
}

// utf16ColumnToByteOffset maps a 0-based UTF-16 character column to the byte
// offset of that column in line. Runes outside the BMP count as two units, per
// the LSP specification's default UTF-16 position encoding.
func utf16ColumnToByteOffset(line string, column int) (int, error) {
	if column < 0 {
		return 0, fmt.Errorf("negative column %d", column)
	}
	units := 0
	for i, r := range line {
		if units >= column {
			return i, nil
		}
		if r >= 0x10000 {
			units += 2 // surrogate pair
		} else {
			units++
		}
	}
	if units >= column {
		return len(line), nil
	}
	return 0, fmt.Errorf("column %d beyond end of line (line is %d UTF-16 units)", column+1, units)
}

// oldNameFromRange extracts the symbol text covered by a PrepareRename range.
// A nil range or an empty range carries no name: some servers answer
// prepareRename with a null/error payload that decodes to a zero range.
func oldNameFromRange(content string, r *lsp.Range) (string, error) {
	if r == nil {
		return "", nil
	}
	if r.Start.Line == r.End.Line && r.Start.Character == r.End.Character {
		return "", nil
	}
	starts := lineStartOffsets(content)
	start, err := positionOffset(starts, content, r.Start)
	if err != nil {
		return "", fmt.Errorf("prepare-rename range: %w", err)
	}
	end, err := positionOffset(starts, content, r.End)
	if err != nil {
		return "", fmt.Errorf("prepare-rename range: %w", err)
	}
	if end < start || end > len(content) {
		return "", fmt.Errorf("prepare-rename range outside file content")
	}
	return content[start:end], nil
}

// findSymbolPosition locates the first word-boundary occurrence of symbol in
// text and returns its 1-based line and 1-based UTF-16 column. Identifiers are
// ASCII in practice, but the column walk is UTF-16-aware for correctness.
func findSymbolPosition(text, symbol string) (int, int, bool) {
	if symbol == "" {
		return 0, 0, false
	}
	re := regexp.MustCompile(`\b` + regexp.QuoteMeta(symbol) + `\b`)
	loc := re.FindStringIndex(text)
	if loc == nil {
		return 0, 0, false
	}
	prefix := text[:loc[0]]
	line := strings.Count(prefix, "\n") + 1
	lineStart := 0
	if idx := strings.LastIndex(prefix, "\n"); idx >= 0 {
		lineStart = idx + 1
	}
	column := utf16Length(text[lineStart:loc[0]]) + 1
	return line, column, true
}

// utf16Length returns the length of s in UTF-16 code units.
func utf16Length(s string) int {
	n := 0
	for _, r := range s {
		if r >= 0x10000 {
			n += 2
		} else {
			n++
		}
	}
	return n
}

// pathFromFileURI converts a file:// URI from a language server into a
// filesystem path. It mirrors the URI form produced above and also tolerates
// opaque forms like file:C:/x.go.
func pathFromFileURI(uri string) (string, error) {
	if !strings.HasPrefix(uri, "file:") {
		return "", fmt.Errorf("unsupported workspace edit URI %q: only file:// URIs are applied", uri)
	}
	u, err := url.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("parse workspace edit URI %q: %w", uri, err)
	}
	p := u.Path
	if p == "" && u.Opaque != "" {
		p = u.Opaque
	}
	if p == "" {
		return "", fmt.Errorf("workspace edit URI %q has no path", uri)
	}
	if runtime.GOOS == "windows" {
		// file:///C:/x.go parses with a leading slash before the drive letter.
		p = strings.TrimPrefix(p, "/")
	}
	return filepath.FromSlash(p), nil
}

// lspDefinition finds where a symbol is defined using codegraph.
func lspDefinition(root, filePath string, line int, symbol string) (string, error) {
	cg, err := codegraph.Open(root)
	if err != nil {
		return "", fmt.Errorf("codegraph not available: %w", err)
	}
	defer func() { _ = cg.Close() }()

	// If symbol is provided directly, search for it
	if symbol != "" {
		nodes, err := cg.Search(symbol, 10)
		if err != nil || len(nodes) == 0 {
			return fmt.Sprintf("No definition found for %q", symbol), nil
		}

		var result strings.Builder
		result.WriteString(fmt.Sprintf("## Definition: %s\n\n", symbol))
		for _, n := range nodes {
			sig := n.QualifiedName
			if n.Signature != "" {
				sig = n.Signature
			}
			result.WriteString(fmt.Sprintf("- **%s** `%s`\n  in %s:%d\n", n.Kind, sig, n.FilePath, n.StartLine))
		}
		return result.String(), nil
	}

	// If line is provided, try to extract symbol from that line
	if line > 0 {
		absPath, err := filepath.Abs(filePath)
		if err != nil {
			absPath = filePath
		}

		source, err := os.ReadFile(absPath) // #nosec G304 -- path provided by caller via tool/task parameters, inherent to this dev CLI's file operations
		if err != nil {
			return fmt.Sprintf("Cannot read file: %s", filePath), nil
		}

		lines := strings.Split(string(source), "\n")
		if line <= len(lines) {
			symbolFromLine := extractSymbolFromLine(lines[line-1])
			if symbolFromLine != "" {
				nodes, err := cg.Search(symbolFromLine, 10)
				if err != nil || len(nodes) == 0 {
					return fmt.Sprintf("No definition found for %q (from line %d)", symbolFromLine, line), nil
				}

				var result strings.Builder
				result.WriteString(fmt.Sprintf("## Definition: %s (from %s:%d)\n\n", symbolFromLine, filePath, line))
				for _, n := range nodes {
					sig := n.QualifiedName
					if n.Signature != "" {
						sig = n.Signature
					}
					result.WriteString(fmt.Sprintf("- **%s** `%s`\n  in %s:%d\n", n.Kind, sig, n.FilePath, n.StartLine))
				}
				return result.String(), nil
			}
		}
	}

	return "Provide 'symbol' or 'line' to find definition", nil
}

// lspReferences finds all references to a symbol using codegraph.
func lspReferences(root, filePath string, line int, symbol string) (string, error) {
	cg, err := codegraph.Open(root)
	if err != nil {
		return "", fmt.Errorf("codegraph not available: %w", err)
	}
	defer func() { _ = cg.Close() }()

	// Get symbol name
	sym := symbol
	if sym == "" && line > 0 {
		absPath, absErr := filepath.Abs(filePath)
		if absErr != nil {
			absPath = filePath
		}
		source, readErr := os.ReadFile(absPath) // #nosec G304 -- path provided by caller via tool/task parameters, inherent to this dev CLI's file operations
		if readErr == nil {
			lines := strings.Split(string(source), "\n")
			if line <= len(lines) {
				sym = extractSymbolFromLine(lines[line-1])
			}
		}
	}

	if sym == "" {
		return "Provide 'symbol' or 'line' to find references", nil
	}

	// Search for the symbol
	nodes, err := cg.Search(sym, 5)
	if err != nil || len(nodes) == 0 {
		return fmt.Sprintf("No references found for %q", sym), nil
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("## References: %s\n\n", sym))

	totalRefs := 0
	for _, n := range nodes {
		// Get callers of this symbol
		callers, err := cg.GetCallers(n.ID, 1)
		if err != nil {
			continue
		}

		if len(callers) > 0 {
			result.WriteString(fmt.Sprintf("### %s `%s` in %s:%d\n", n.Kind, n.Name, n.FilePath, n.StartLine))
			result.WriteString(fmt.Sprintf("Called by %d:\n", len(callers)))
			for _, c := range callers {
				result.WriteString(fmt.Sprintf("- %s `%s` in %s:%d\n", c.Kind, c.Name, c.FilePath, c.StartLine))
				totalRefs++
			}
			result.WriteString("\n")
		}
	}

	if totalRefs == 0 {
		return fmt.Sprintf("No references found for %q", sym), nil
	}

	return result.String(), nil
}

// lspImplementations finds types that implement an interface using codegraph.
func lspImplementations(root, symbol string) (string, error) {
	if symbol == "" {
		return "Provide 'symbol' (interface name) to find implementations", nil
	}

	cg, err := codegraph.Open(root)
	if err != nil {
		return "", fmt.Errorf("codegraph not available: %w", err)
	}
	defer func() { _ = cg.Close() }()

	// Search for the interface
	nodes, err := cg.Search(symbol, 5)
	if err != nil || len(nodes) == 0 {
		return fmt.Sprintf("No interface found for %q", symbol), nil
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("## Implementations: %s\n\n", symbol))

	for _, n := range nodes {
		if n.Kind != "interface" && n.Kind != "trait" {
			continue
		}

		result.WriteString(fmt.Sprintf("### Interface: `%s` in %s:%d\n\n", n.QualifiedName, n.FilePath, n.StartLine))

		// Find nodes that implement this interface (via 'implements' edges)
		impact, err := cg.GetImpactRadius(n.ID, 2)
		if err != nil {
			continue
		}

		implementors := 0
		for _, imp := range impact {
			if imp.Kind == "struct" || imp.Kind == "class" {
				result.WriteString(fmt.Sprintf("- **%s** `%s` in %s:%d\n", imp.Kind, imp.Name, imp.FilePath, imp.StartLine))
				implementors++
			}
		}

		if implementors == 0 {
			result.WriteString("No implementations found.\n")
		}
	}

	return result.String(), nil
}

// extractSymbolFromLine extracts a likely symbol name from a line of code.
func extractSymbolFromLine(line string) string {
	line = strings.TrimSpace(line)

	// Common patterns to extract symbol names
	patterns := []struct {
		prefix string
		suffix string
	}{
		{"func ", "("},
		{"def ", "("},
		{"function ", "("},
		{"class ", "{"},
		{"class ", ":"},
		{"interface ", "{"},
		{"type ", " struct"},
		{"type ", " interface"},
		{"import {", "}"},
		{"from ", " import"},
		{"new ", "("},
		{"", "("}, // generic function call
	}

	for _, p := range patterns {
		if p.prefix != "" && !strings.HasPrefix(line, p.prefix) {
			continue
		}

		text := line
		if p.prefix != "" {
			text = strings.TrimPrefix(text, p.prefix)
		}

		if idx := strings.Index(text, p.suffix); idx > 0 {
			symbol := strings.TrimSpace(text[:idx])
			// Clean up common noise
			symbol = strings.TrimLeft(symbol, "*&") // pointer/receiver
			if strings.Contains(symbol, ".") {
				parts := strings.Split(symbol, ".")
				symbol = parts[len(parts)-1] // last part
			}
			if len(symbol) > 2 && len(symbol) < 100 {
				return symbol
			}
		}
	}

	// Fallback: look for camelCase or snake_case identifiers
	words := strings.FieldsFunc(line, func(r rune) bool {
		return !isAlphaNum(r) && r != '_' && r != '.'
	})
	for _, w := range words {
		if len(w) > 2 && (isCamelCase(w) || isSnakeCase(w)) {
			// Skip common keywords
			lower := strings.ToLower(w)
			if lower != "func" && lower != "return" && lower != "if" && lower != "else" &&
				lower != "for" && lower != "range" && lower != "var" && lower != "const" &&
				lower != "type" && lower != "struct" && lower != "interface" && lower != "package" {
				return w
			}
		}
	}

	return ""
}

func isAlphaNum(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

func isCamelCase(s string) bool {
	hasLower := false
	hasUpper := false
	for _, r := range s {
		if r >= 'a' && r <= 'z' {
			hasLower = true
		}
		if r >= 'A' && r <= 'Z' {
			hasUpper = true
		}
	}
	return hasLower && hasUpper
}

func isSnakeCase(s string) bool {
	return strings.Contains(s, "_") && len(s) > 3
}

func lspDiagnostics(ctx context.Context, path string) (string, error) {
	ext := ""
	if i := strings.LastIndex(path, "."); i >= 0 {
		ext = path[i:]
	}

	var cmd *exec.Cmd
	switch ext {
	case ".go":
		cmd = exec.CommandContext(ctx, "go", "vet", "./...")
	case ".ts", ".tsx", ".js", ".jsx":
		cmd = exec.CommandContext(ctx, "npx", "tsc", "--noEmit", "--pretty")
	case ".py":
		cmd = exec.CommandContext(ctx, "python3", "-m", "py_compile", path) // #nosec G204 -- fixed interpreter and separate path argument
	case ".rs":
		cmd = exec.CommandContext(ctx, "cargo", "check", "--message-format=short")
	default:
		return fmt.Sprintf("No linter configured for %s files", ext), nil
	}

	out, err := cmd.CombinedOutput()
	result := strings.TrimSpace(string(out))
	if err != nil && result == "" {
		result = err.Error()
	}
	if result == "" {
		return "No diagnostics found.", nil
	}
	if len(result) > 10000 {
		result = result[:10000] + "\n... (truncated)"
	}
	return result, nil
}
