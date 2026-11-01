package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestConnectSurvivesDiscoveryCancellation(t *testing.T) {
	t.Setenv("REFLECT_MCP_TEST_CHILD", "1")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	server, err := Connect(ctx, "fixture", os.Args[0], "-test.run=^TestMCPStdioProcessHelper$")
	cancel()
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	if _, err := server.ListTools(); err != nil {
		t.Fatalf("server died after discovery context ended: %v", err)
	}
	value, err := server.CallTool(context.Background(), "echo", map[string]interface{}{})
	if err != nil || value != "alive" {
		t.Fatalf("call after cancellation: %q %v", value, err)
	}
}

func TestMCPStdioProcessHelper(t *testing.T) {
	if os.Getenv("REFLECT_MCP_TEST_CHILD") != "1" {
		return
	}
	scanner := bufio.NewScanner(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)
	for scanner.Scan() {
		var req struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
		}
		if json.Unmarshal(scanner.Bytes(), &req) != nil || len(req.ID) == 0 {
			continue
		}
		var result any = map[string]any{}
		switch req.Method {
		case "initialize":
			result = map[string]any{"protocolVersion": "2025-03-26", "capabilities": map[string]any{}, "serverInfo": map[string]string{"name": "fixture", "version": "1"}}
		case "tools/list":
			result = map[string]any{"tools": []any{}}
		case "tools/call":
			result = map[string]any{"content": []map[string]string{{"type": "text", "text": "alive"}}}
		}
		if err := encoder.Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result}); err != nil {
			os.Exit(2)
		}
	}
	os.Exit(0)
}
