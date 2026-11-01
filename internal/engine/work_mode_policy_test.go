package engine

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GrayCodeAI/reflect/internal/engine/safety"
	"github.com/GrayCodeAI/reflect/internal/storage"
	"github.com/GrayCodeAI/reflect/internal/tool"
	"github.com/GrayCodeAI/reflect/internal/types"
)

func TestWorkModeRestrictsExecutionAfterToolPromotion(t *testing.T) {
	storage.SetTestDirs(t, t.TempDir())
	project := t.TempDir()
	path := filepath.Join(project, "result.txt")
	reg := tool.NewRegistry(tool.FileWriteTool{}, tool.FileReadTool{}, tool.BashTool{}, tool.ToolSearchTool{})
	reg.EnableLazyModelSurface(EssentialModelTools)
	sess := NewSession("test", "test", "system", reg)
	sess.Tools().SetWorkingDir(project)
	sess.PermSvc().SetAllowedDirs([]string{project})
	sess.PermSvc().SetAutonomy(safety.AutonomyYOLO)
	write := types.ToolCall{ID: "write", Name: "Write", Arguments: map[string]interface{}{"file_path": path, "content": "built"}}
	for _, mode := range []WorkMode{WorkModePlan, WorkModeReview} {
		if err := sess.SetWorkMode(mode); err != nil {
			t.Fatal(err)
		}
		promote := sess.executeSingleTool(context.Background(), types.ToolCall{ID: "search", Name: "ToolSearch", Arguments: map[string]interface{}{"query": "select:Write"}}, nil, 0, "")
		if promote.isErr || !reg.IsModelVisible("Write") {
			t.Fatalf("promote Write: %s", promote.output)
		}
		res := sess.executeSingleTool(context.Background(), write, nil, 0, "")
		if !res.isErr || !strings.Contains(res.output, "read-only") {
			t.Fatalf("%s permitted promoted Write: %#v", mode, res)
		}
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("%s wrote file: %v", mode, err)
		}
		res = sess.executeSingleTool(context.Background(), types.ToolCall{ID: "bash", Name: "Bash", Arguments: map[string]interface{}{"command": "touch result.txt"}}, nil, 0, "")
		if !res.isErr {
			t.Fatalf("%s permitted a mutating shell command", mode)
		}
		if reason := sess.Tools().workModeRestriction("PowerShell"); reason == "" {
			t.Fatalf("%s allowed unguarded PowerShell", mode)
		}
	}
	if err := sess.SetWorkMode(WorkModeAct); err != nil {
		t.Fatal(err)
	}
	res := sess.executeSingleTool(context.Background(), write, nil, 0, "")
	if res.isErr {
		t.Fatalf("Build did not restore Write: %s", res.output)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "built" {
		t.Fatalf("Build output = %q, %v", got, err)
	}
	if sess.PermSvc().RuntimeState().Autonomy != safety.AutonomyYOLO {
		t.Fatal("mode switch changed permission tier")
	}
}

func TestWorkModeRequiresExplicitSpecWorkflow(t *testing.T) {
	storage.SetTestDirs(t, t.TempDir())
	sess := NewSession("test", "test", "system", tool.NewRegistry(tool.SpecifyTool{}))
	sess.Tools().SetWorkingDir(t.TempDir())
	sess.PermSvc().SetAutonomy(safety.AutonomyYOLO)
	call := types.ToolCall{ID: "spec", Name: "Specify", Arguments: map[string]interface{}{"specification": "An unwanted spec"}}
	res := sess.executeSingleTool(context.Background(), call, nil, 0, "")
	if !res.isErr || !strings.Contains(res.output, "/spec") {
		t.Fatalf("implicit spec was accepted: %#v", res)
	}
	if err := sess.SetWorkMode(WorkModePlan); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Proposal", "Design", "Specify", "Plan", "Tasks", "SpecEdit", "ApproveImplementation"} {
		if reason := sess.Tools().workModeRestriction(name); reason == "" {
			t.Fatalf("inactive spec allowed %s", name)
		}
	}
	sess.PermSvc().SetSpecStage(safety.SpecStageProposal)
	for _, name := range []string{"Proposal", "Design", "Specify", "Plan", "Tasks"} {
		if reason := sess.Tools().workModeRestriction(name); reason != "" {
			t.Fatalf("explicit spec blocked %s: %s", name, reason)
		}
	}
	if sess.workModeSystemAddon() != "" {
		t.Fatal("everyday Plan prompt overrides active advanced spec")
	}
}
