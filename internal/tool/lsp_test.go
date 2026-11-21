package tool

import (
	"context"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GrayCodeAI/reflect/internal/lsp"
)

// rng builds an LSP range from 0-based line/character positions.
func rng(startLine, startChar, endLine, endChar int) lsp.Range {
	return lsp.Range{
		Start: lsp.Position{Line: startLine, Character: startChar},
		End:   lsp.Position{Line: endLine, Character: endChar},
	}
}

// lspFileURI builds the file:// URI form the production code uses.
func lspFileURI(path string) string {
	return (&url.URL{Scheme: "file", Path: filepath.ToSlash(path)}).String()
}

// mustWriteFile writes content or fails the test.
func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// fakeProtector implements PathProtector for tests.
type fakeProtector struct {
	protected map[string]bool
}

func (f fakeProtector) IsProtected(path string) bool { return f.protected[path] }

func TestLSPToolSchemaIncludesRenameActions(t *testing.T) {
	props := schemaProps(t, LSPTool{}.Parameters())
	enum, ok := props["action"].(map[string]interface{})["enum"].([]interface{})
	if !ok {
		t.Fatalf("action enum missing: %v", props["action"])
	}
	seen := map[string]bool{}
	for _, v := range enum {
		name, _ := v.(string)
		seen[name] = true
	}
	for _, want := range []string{"diagnostics", "definition", "references", "implementations", "prepare_rename", "rename"} {
		if !seen[want] {
			t.Errorf("action %q missing from enum %v", want, enum)
		}
	}
	if _, ok := props["new_name"]; !ok {
		t.Error("new_name property missing from schema")
	}
}

func TestLSPToolExecuteRenameArgValidation(t *testing.T) {
	cases := []struct {
		name    string
		input   map[string]interface{}
		wantErr string
	}{
		{
			name:    "missing path",
			input:   map[string]interface{}{"action": "rename", "new_name": "total"},
			wantErr: "path is required",
		},
		{
			name:    "missing new_name",
			input:   map[string]interface{}{"action": "rename", "path": "main.go", "line": 1, "column": 1},
			wantErr: "new_name is required for rename",
		},
		{
			name:    "prepare_rename missing new_name",
			input:   map[string]interface{}{"action": "prepare_rename", "path": "main.go", "line": 1, "column": 1},
			wantErr: "new_name is required for prepare_rename",
		},
		{
			name:    "missing position and symbol",
			input:   map[string]interface{}{"action": "rename", "path": "main.go", "new_name": "total"},
			wantErr: "line and column",
		},
		{
			name:    "unknown action",
			input:   map[string]interface{}{"action": "explode", "path": "main.go"},
			wantErr: "unknown LSP action",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input, err := json.Marshal(tc.input)
			if err != nil {
				t.Fatal(err)
			}
			_, err = LSPTool{}.Execute(testCtx(), input)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error = %q, want substring %q", err.Error(), tc.wantErr)
			}
		})
	}
}

func TestLSPToolExecuteRename_NoManager(t *testing.T) {
	input, err := json.Marshal(map[string]interface{}{
		"action": "rename", "path": "main.go", "line": 1, "column": 1, "new_name": "total",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = LSPTool{}.Execute(testCtx(), input)
	if err == nil {
		t.Fatal("expected error when no language-server manager is set")
	}
	for _, want := range []string{"no language-server manager", "rename_symbol"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error = %q, want substring %q", err.Error(), want)
		}
	}
}

func TestLSPToolExecuteRename_NoServerForExtension(t *testing.T) {
	mgr := lsp.NewManager(&lsp.LSPConfig{Servers: map[string]lsp.ServerConfig{}})
	defer func() { _ = mgr.Close() }()

	file := filepath.Join(t.TempDir(), "thing.xyz")
	input, err := json.Marshal(map[string]interface{}{
		"action": "rename", "path": file, "line": 1, "column": 1, "new_name": "total",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = LSPTool{Manager: mgr}.Execute(testCtx(), input)
	if err == nil {
		t.Fatal("expected error when no server is configured for the file's language")
	}
	for _, want := range []string{"no language server configured for .xyz files", "rename_symbol"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error = %q, want substring %q", err.Error(), want)
		}
	}
}

func TestLSPToolExecuteRename_ServerStartFailure(t *testing.T) {
	mgr := lsp.NewManager(&lsp.LSPConfig{Servers: map[string]lsp.ServerConfig{
		"go": {Command: "nonexistent-lsp-command-xyz", Extensions: []string{".go"}},
	}})
	defer func() { _ = mgr.Close() }()

	file := filepath.Join(t.TempDir(), "main.go")
	mustWriteFile(t, file, "package main\n")
	input, err := json.Marshal(map[string]interface{}{
		"action": "rename", "path": file, "line": 1, "column": 9, "new_name": "other",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = LSPTool{Manager: mgr}.Execute(testCtx(), input)
	if err == nil {
		t.Fatal("expected error when the language server fails to start")
	}
	for _, want := range []string{"lsp rename via go server", "rename_symbol"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error = %q, want substring %q", err.Error(), want)
		}
	}
}

func TestFindSymbolPosition(t *testing.T) {
	cases := []struct {
		name       string
		text       string
		symbol     string
		wantLine   int
		wantColumn int
		wantOK     bool
	}{
		{"middle of file", "package main\n\nvar count = 0\n", "count", 3, 5, true},
		{"second line", "a := 1\nname := 2\n", "name", 2, 1, true},
		{"word boundary respected", "namePrefix := 1\n", "name", 0, 0, false},
		{"absent symbol", "a := 1\n", "missing", 0, 0, false},
		{"empty symbol", "a := 1\n", "", 0, 0, false},
		{"regex metachars quoted", "x := a.b\n", "a.b", 1, 6, true},
		{"utf16 column", "x := \"𝄞\"; alpha = 1\n", "alpha", 1, 12, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			line, column, ok := findSymbolPosition(tc.text, tc.symbol)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if line != tc.wantLine || column != tc.wantColumn {
				t.Errorf("position = %d:%d, want %d:%d", line, column, tc.wantLine, tc.wantColumn)
			}
		})
	}
}

func TestPathFromFileURI(t *testing.T) {
	cases := []struct {
		name    string
		uri     string
		want    string
		wantErr bool
	}{
		{"plain", "file:///tmp/x.go", filepath.FromSlash("/tmp/x.go"), false},
		{
			"percent-encoded",
			(&url.URL{Scheme: "file", Path: filepath.ToSlash("/tmp/x y.go")}).String(),
			filepath.FromSlash("/tmp/x y.go"),
			false,
		},
		{"non-file scheme", "untitled:Untitled-1", "", true},
		{"no path", "file://", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := pathFromFileURI(tc.uri)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("path = %q, want error", got)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("path = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFormatRenamePreview(t *testing.T) {
	outcome := &lspRenameOutcome{
		OldName: "calculate",
		NewName: "compute",
		Files:   2,
		Edits:   3,
		PerFile: []lspRenameFileEdits{
			{Path: filepath.FromSlash("/proj/a.go"), Edits: 2},
			{Path: filepath.FromSlash("/proj/b.go"), Edits: 1},
		},
	}
	out := formatRenamePreview(outcome)
	for _, want := range []string{
		`Old name: "calculate"`, `New name: "compute"`,
		"Affected files: 2", "Total edits: 3",
		filepath.FromSlash("/proj/a.go") + ": 2 edit(s)",
		filepath.FromSlash("/proj/b.go") + ": 1 edit(s)",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("preview missing %q:\n%s", want, out)
		}
	}
}

func TestFormatRenamePreview_UnknownOldName(t *testing.T) {
	out := formatRenamePreview(&lspRenameOutcome{NewName: "compute"})
	if !strings.Contains(out, "(unknown)") {
		t.Errorf("preview should mark the old name as unknown:\n%s", out)
	}
}

func TestApplyTextEdits(t *testing.T) {
	cases := []struct {
		name    string
		content string
		edits   []lsp.TextEdit
		want    string
		wantErr string
	}{
		{
			name:    "single edit",
			content: "count := 1\n",
			edits:   []lsp.TextEdit{{Range: rng(0, 0, 0, 5), NewText: "total"}},
			want:    "total := 1\n",
		},
		{
			name:    "two edits applied back to front",
			content: "count := 1\ncount++\n",
			edits: []lsp.TextEdit{
				{Range: rng(0, 0, 0, 5), NewText: "total"},
				{Range: rng(1, 0, 1, 5), NewText: "total"},
			},
			want: "total := 1\ntotal++\n",
		},
		{
			name:    "utf16 positions",
			content: "x := \"𝄞\"; alpha = 1\n",
			edits:   []lsp.TextEdit{{Range: rng(0, 11, 0, 16), NewText: "beta"}},
			want:    "x := \"𝄞\"; beta = 1\n",
		},
		{
			name:    "no edits",
			content: "count := 1\n",
			edits:   nil,
			want:    "count := 1\n",
		},
		{
			name:    "line out of range",
			content: "count := 1\n",
			edits:   []lsp.TextEdit{{Range: rng(99, 0, 99, 5), NewText: "total"}},
			wantErr: "out of range",
		},
		{
			name:    "column beyond line",
			content: "count := 1\n",
			edits:   []lsp.TextEdit{{Range: rng(0, 0, 0, 99), NewText: "total"}},
			wantErr: "beyond end of line",
		},
		{
			name:    "inverted range",
			content: "count := 1\n",
			edits:   []lsp.TextEdit{{Range: rng(0, 5, 0, 0), NewText: "total"}},
			wantErr: "inverted range",
		},
		{
			name:    "overlapping edits",
			content: "package main\n",
			edits: []lsp.TextEdit{
				{Range: rng(0, 0, 0, 7), NewText: "mod"},
				{Range: rng(0, 4, 0, 10), NewText: "xxx"},
			},
			wantErr: "overlapping",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := applyTextEdits(tc.content, tc.edits)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error = %v, want substring %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("result = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestApplyWorkspaceEdit_Success(t *testing.T) {
	dir := t.TempDir()
	aPath := filepath.Join(dir, "a.go")
	bPath := filepath.Join(dir, "b.go")
	mustWriteFile(t, aPath, "package main\n\nfunc main() {\n\tcount := 1\n\tcount++\n}\n")
	mustWriteFile(t, bPath, "package main\n\nvar count = 2\n")

	edit := &lsp.WorkspaceEdit{Changes: map[string][]lsp.TextEdit{
		lspFileURI(aPath): {
			{Range: rng(3, 1, 3, 6), NewText: "total"},
			{Range: rng(4, 1, 4, 6), NewText: "total"},
		},
		lspFileURI(bPath): {
			{Range: rng(2, 4, 2, 9), NewText: "total"},
		},
	}}

	summary, err := applyWorkspaceEdit(testCtx(), edit)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(summary, "2 file(s)") || !strings.Contains(summary, "3 edit(s)") {
		t.Errorf("summary = %q, want 2 file(s) and 3 edit(s)", summary)
	}

	aData, err := os.ReadFile(aPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(aData); !strings.Contains(got, "total := 1") || !strings.Contains(got, "total++") {
		t.Errorf("a.go = %q, want count renamed on both lines", got)
	}

	bData, err := os.ReadFile(bPath)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(bData), "package main\n\nvar total = 2\n"; got != want {
		t.Errorf("b.go = %q, want %q", got, want)
	}
}

func TestApplyWorkspaceEdit_RejectedByGuards(t *testing.T) {
	cases := []struct {
		name    string
		file    string
		content string
		newText string
		ctx     func(path string) context.Context
		wantErr string
	}{
		{
			name:    "no tool context",
			file:    "a.go",
			content: "package main\n",
			newText: "xyz",
			ctx:     func(string) context.Context { return context.Background() },
			wantErr: "no tool context attached",
		},
		{
			name:    "protected path",
			file:    "a.go",
			content: "package main\n",
			newText: "xyz",
			ctx: func(path string) context.Context {
				return WithToolContext(context.Background(), &ToolContext{
					AllowedDirectories: []string{string(filepath.Separator)},
					Protected:          fakeProtector{protected: map[string]bool{path: true}},
				})
			},
			wantErr: "protected (read-only)",
		},
		{
			name:    "sensitive path",
			file:    ".env",
			content: "KEY=value\n",
			newText: "value2",
			ctx:     func(string) context.Context { return testCtx() },
			wantErr: "blocked",
		},
		{
			name:    "credential in new content",
			file:    "a.go",
			content: "var alpha = 1\n",
			newText: "sk-ant-api00-abcdefghijklmnopqrst",
			ctx:     func(string) context.Context { return testCtx() },
			wantErr: "credential",
		},
		{
			name:    "outside allowed directories",
			file:    "a.go",
			content: "package main\n",
			newText: "xyz",
			ctx: func(path string) context.Context {
				return WithToolContext(context.Background(), &ToolContext{
					AllowedDirectories: []string{filepath.Join(filepath.Dir(path), "not_target")},
				})
			},
			wantErr: "outside the working directory",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), tc.file)
			mustWriteFile(t, path, tc.content)
			edit := &lsp.WorkspaceEdit{Changes: map[string][]lsp.TextEdit{
				lspFileURI(path): {{Range: rng(0, 0, 0, 3), NewText: tc.newText}},
			}}
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			_, err = applyWorkspaceEdit(tc.ctx(path), edit)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error = %v, want substring %q", err, tc.wantErr)
			}
			after, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(before) != string(after) {
				t.Errorf("file was modified despite rejection: %q", string(after))
			}
		})
	}
}

func TestApplyWorkspaceEdit_BadEditLeavesFilesUnmodified(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "good.go")
	bad := filepath.Join(dir, "bad.go")
	mustWriteFile(t, good, "package good\n")
	mustWriteFile(t, bad, "package bad\n")

	edit := &lsp.WorkspaceEdit{Changes: map[string][]lsp.TextEdit{
		lspFileURI(good): {{Range: rng(0, 8, 0, 12), NewText: "excellent"}},
		lspFileURI(bad):  {{Range: rng(99, 0, 99, 4), NewText: "nope"}},
	}}
	beforeGood, err := os.ReadFile(good)
	if err != nil {
		t.Fatal(err)
	}
	beforeBad, err := os.ReadFile(bad)
	if err != nil {
		t.Fatal(err)
	}

	_, err = applyWorkspaceEdit(testCtx(), edit)
	if err == nil || !strings.Contains(err.Error(), "out of range") {
		t.Fatalf("error = %v, want out-of-range failure", err)
	}
	afterGood, err := os.ReadFile(good)
	if err != nil {
		t.Fatal(err)
	}
	afterBad, err := os.ReadFile(bad)
	if err != nil {
		t.Fatal(err)
	}
	if string(beforeGood) != string(afterGood) {
		t.Errorf("good.go was modified despite the failed edit: %q", string(afterGood))
	}
	if string(beforeBad) != string(afterBad) {
		t.Errorf("bad.go was modified despite the failed edit: %q", string(afterBad))
	}
}

func TestApplyWorkspaceEdit_OverlappingEditsRejected(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a.go")
	mustWriteFile(t, path, "package main\n")
	edit := &lsp.WorkspaceEdit{Changes: map[string][]lsp.TextEdit{
		lspFileURI(path): {
			{Range: rng(0, 0, 0, 7), NewText: "mod"},
			{Range: rng(0, 4, 0, 10), NewText: "xxx"},
		},
	}}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = applyWorkspaceEdit(testCtx(), edit)
	if err == nil || !strings.Contains(err.Error(), "overlapping") {
		t.Fatalf("error = %v, want overlapping-edit failure", err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Errorf("file was modified despite rejection: %q", string(after))
	}
}

func TestApplyWorkspaceEdit_UTF16Positions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a.go")
	mustWriteFile(t, path, "x := \"𝄞\"; alpha = 1\n")
	edit := &lsp.WorkspaceEdit{Changes: map[string][]lsp.TextEdit{
		lspFileURI(path): {{Range: rng(0, 11, 0, 16), NewText: "beta"}},
	}}
	if _, err := applyWorkspaceEdit(testCtx(), edit); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(data), "x := \"𝄞\"; beta = 1\n"; got != want {
		t.Errorf("content = %q, want %q", got, want)
	}
}

func TestApplyWorkspaceEdit_Empty(t *testing.T) {
	cases := []struct {
		name    string
		edit    *lsp.WorkspaceEdit
		wantErr string
	}{
		{"nil edit", nil, "workspace edit is empty"},
		{"no changes", &lsp.WorkspaceEdit{}, "workspace edit is empty"},
		{"empty edit lists", &lsp.WorkspaceEdit{Changes: map[string][]lsp.TextEdit{"file:///x.go": {}}}, "no file changes"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := applyWorkspaceEdit(testCtx(), tc.edit)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error = %v, want substring %q", err, tc.wantErr)
			}
		})
	}
}

func TestApplyWorkspaceEdit_UnsupportedURI(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a.go")
	mustWriteFile(t, path, "package main\n")
	edit := &lsp.WorkspaceEdit{Changes: map[string][]lsp.TextEdit{
		"untitled:Untitled-1": {{Range: rng(0, 0, 0, 7), NewText: "xyz"}},
	}}
	_, err := applyWorkspaceEdit(testCtx(), edit)
	if err == nil || !strings.Contains(err.Error(), "only file:// URIs are applied") {
		t.Fatalf("error = %v, want unsupported-URI failure", err)
	}
}
