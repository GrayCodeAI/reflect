package tool

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

func TestBashPipelineFailurePreservesDiagnostics(t *testing.T) {
	input, _ := json.Marshal(map[string]string{"command": "printf 'compiler diagnostic\\n'; false | tail -5"})
	out, err := RetryExecutor(testCtx(), BashTool{}, input, DefaultRetryPolicy())
	if err == nil || !strings.Contains(err.Error(), "exit code:") || !strings.Contains(out, "compiler diagnostic") {
		t.Fatalf("pipeline failure lost its status or diagnostic: output=%q error=%v", out, err)
	}
}

func TestSandboxBashQuotingAndPipelineStatus(t *testing.T) {
	command := bashSandboxCommand("printf '%s\\n' 'quoted value'; false | cat")
	out, err := exec.CommandContext(t.Context(), "bash", "-c", command).CombinedOutput()
	if err == nil || !strings.Contains(string(out), "quoted value") {
		t.Fatalf("sandbox wrapper changed quoting or hid failure: %q, %v", out, err)
	}
}

func TestBashPipelineSuccess(t *testing.T) {
	input := json.RawMessage(`{"command":"printf 'checked\\n' | tail -1"}`)
	out, err := (BashTool{}).Execute(testCtx(), input)
	if err != nil || out != "checked" {
		t.Fatalf("successful pipeline failed: %q, %v", out, err)
	}
}

func TestBashBackgroundPipelineFails(t *testing.T) {
	input := json.RawMessage(`{"command":"false | cat","run_in_background":true}`)
	out, err := (BashTool{}).Execute(testCtx(), input)
	if err != nil {
		t.Fatal(err)
	}
	id := extractTaskID(out)
	if id == "" {
		t.Fatalf("missing background task: %q", out)
	}
	task, ok := getBackgroundTask(id)
	if !ok {
		t.Fatal("task was not registered")
	}
	select {
	case <-task.done:
	case <-t.Context().Done():
		t.Fatal("task did not finish")
	}
	status, _, _, _, err := resolveTaskOutput(t.Context(), id)
	if err != nil || status != "failed" {
		t.Fatalf("pipeline was not marked failed: status=%s error=%v", status, err)
	}
}
