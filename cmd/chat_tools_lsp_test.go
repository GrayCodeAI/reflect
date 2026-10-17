package cmd

import (
	"testing"

	reflectconfig "github.com/GrayCodeAI/reflect/internal/config"
	"github.com/GrayCodeAI/reflect/internal/tool"
)

func TestDefaultRegistryWiresLanguageServerManager(t *testing.T) {
	registry, err := defaultRegistry(reflectconfig.Settings{})
	if err != nil {
		t.Fatal(err)
	}
	registered, ok := registry.Get("LSP")
	if !ok {
		t.Fatal("default registry must include LSP")
	}
	lspTool, ok := registered.(tool.LSPTool)
	if !ok {
		t.Fatalf("LSP tool type = %T, want tool.LSPTool", registered)
	}
	if lspTool.Manager == nil {
		t.Fatal("LSP tool must have a language-server manager")
	}
	if len(lspTool.Manager.Status()) == 0 {
		t.Fatal("language-server manager should expose built-in server configurations")
	}
	_ = lspTool.Manager.Close()
}
