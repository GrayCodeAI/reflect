package tool

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/GrayCodeAI/reflect/internal/storage"
)

func TestReadOnlyBashCannotWriteThroughSed(t *testing.T) {
	dir := t.TempDir()
	source, marker := filepath.Join(dir, "source.txt"), filepath.Join(dir, "marker.txt")
	if err := os.WriteFile(source, []byte("fixture\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx := WithToolContext(t.Context(), &ToolContext{ReadOnlyBash: true, WorkingDir: dir})
	input, _ := json.Marshal(map[string]interface{}{"command": "sed 'w " + marker + "' " + source})
	if _, err := (BashTool{}).Execute(ctx, input); err == nil {
		t.Fatal("read-only Bash accepted a writing sed program")
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("read-only command created marker: %v", err)
	}
}

func TestSandboxRejectsBackgroundExecution(t *testing.T) {
	storage.SetTestDirs(t, t.TempDir())
	t.Setenv("REFLECT_SANDBOX", "bwrap")
	ctx := WithToolContext(t.Context(), &ToolContext{WorkingDir: t.TempDir()})
	_, err := (BashTool{}).Execute(ctx, json.RawMessage(`{"command":"echo host","run_in_background":true}`))
	if err == nil || !strings.Contains(err.Error(), "sandbox") {
		t.Fatalf("background execution must not bypass sandboxing: %v", err)
	}
}

func TestReadOnlyGitDisablesRepositoryPrograms(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX helper fixture")
	}
	storage.SetTestDirs(t, t.TempDir())
	t.Setenv("REFLECT_SANDBOX", "off")
	dir := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		command := exec.CommandContext(t.Context(), "git", args...)
		command.Dir = dir
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s, %v", args, output, err)
		}
	}
	git("init")
	for name, content := range map[string]string{
		".gitattributes": "*.txt diff=audit\n",
		"project.txt":    "before\n",
		"helper.sh":      "#!/bin/sh\necho invoked > helper-ran\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	git("add", ".")
	git("config", "diff.audit.textconv", "./helper.sh")
	git("config", "core.fsmonitor", "./helper.sh")
	if err := os.WriteFile(filepath.Join(dir, "project.txt"), []byte("after\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx := WithToolContext(t.Context(), &ToolContext{ReadOnlyBash: true, WorkingDir: dir})
	for _, command := range []string{"git diff", "git status"} {
		input, _ := json.Marshal(map[string]interface{}{"command": command})
		if output, err := (BashTool{}).Execute(ctx, input); err != nil || strings.Contains(output, "exit code:") {
			t.Fatalf("%s: %s, %v", command, output, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "helper-ran")); !os.IsNotExist(err) {
		t.Fatalf("read-only git invoked a repository program: %v", err)
	}
}

func TestBashDeadlineTerminatesDescendants(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell fixture; Windows uses taskkill /T")
	}
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash unavailable")
	}
	storage.SetTestDirs(t, t.TempDir())
	t.Setenv("REFLECT_SANDBOX", "off")
	dir := t.TempDir()
	marker := filepath.Join(dir, "survived.txt")
	ctx, cancel := context.WithTimeout(t.Context(), 150*time.Millisecond)
	defer cancel()
	ctx = WithToolContext(ctx, &ToolContext{WorkingDir: dir})
	input := json.RawMessage(`{"command":"sleep 1; echo survived > survived.txt"}`)
	started := time.Now()
	output, err := (BashTool{}).Execute(ctx, input)
	if err != nil || !strings.Contains(output, "timed out") {
		t.Fatalf("expected deadline result, got %q, %v", output, err)
	}
	if elapsed := time.Since(started); elapsed > 750*time.Millisecond {
		t.Fatalf("deadline waited for descendants: %v", elapsed)
	}
	// A child left running would create this marker after the tool returned.
	time.Sleep(time.Second)
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("descendant survived cancellation: %v", err)
	}
}

func TestFileToolsUseSessionWorkingDir(t *testing.T) {
	parent, child := t.TempDir(), t.TempDir()
	t.Chdir(parent)
	for _, dir := range []string{parent, child} {
		if err := os.WriteFile(filepath.Join(dir, "project.txt"), []byte("original"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	ctx := WithToolContext(t.Context(), &ToolContext{WorkingDir: child, AllowedDirectories: []string{child}})
	cases := []struct {
		tool  Tool
		input string
	}{
		{FileWriteTool{}, `{"path":"project.txt","content":"child change"}`},
		{FileEditTool{}, `{"file_path":"project.txt","old_str":"original","new_str":"child change"}`},
		{MultiEditTool{}, `{"file_path":"project.txt","edits":[{"old_string":"original","new_string":"child change"}]}`},
		{StructuredEditTool{}, `{"path":"project.txt","blocks":[{"search":"original","replace":"child change"}]}`},
	}
	for _, tc := range cases {
		t.Run(tc.tool.Name(), func(t *testing.T) {
			if err := os.WriteFile(filepath.Join(child, "project.txt"), []byte("original"), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := tc.tool.Execute(ctx, json.RawMessage(tc.input)); err != nil {
				t.Fatal(err)
			}
			parentData, _ := os.ReadFile(filepath.Join(parent, "project.txt"))
			childData, _ := os.ReadFile(filepath.Join(child, "project.txt"))
			if string(parentData) != "original" || string(childData) != "child change" {
				t.Fatalf("workspace isolation failed: parent=%q child=%q", parentData, childData)
			}
		})
	}
	if err := validatePathAllowed(ctx, filepath.Join(parent, "project.txt")); err == nil {
		t.Fatal("parent process cwd must not be implicitly allowed in a child workspace")
	}
	output, err := (FileReadTool{}).Execute(ctx, json.RawMessage(`{"path":"project.txt"}`))
	if err != nil || !strings.Contains(output, "child change") {
		t.Fatalf("Read used the wrong workspace: %q, %v", output, err)
	}
}

func TestWorkspaceArgumentsPreserveCallerAndRemotePaths(t *testing.T) {
	dir := t.TempDir()
	args := map[string]interface{}{"path": "project.txt", "content": "unchanged"}
	got := WorkspaceArguments("Write", args, dir)
	if got["path"] != filepath.Join(dir, "project.txt") || args["path"] != "project.txt" {
		t.Fatalf("path normalization mutated caller arguments: got=%v caller=%v", got, args)
	}
	if got := WorkspaceArguments("mcp__remote__read", args, dir); got["path"] != "project.txt" {
		t.Fatalf("remote path rewritten: %v", got)
	}
}
