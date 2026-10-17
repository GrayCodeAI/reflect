// Package hosttools provides a lightweight host-tool registry inspired by
// Mirror's Python extensions. External (e.g. Python) scripts can be registered
// as Reflect tools via JSON manifests in ~/.reflect/host_tools/*.json.
//
// Each manifest describes one tool: name, description, command, and args
// (with {input} placeholder replaced by the tool's JSON input). Reflect invokes
// the command and returns its stdout as the tool result.
package hosttools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/GrayCodeAI/reflect/internal/storage"
)

// Manifest describes a host tool.
type Manifest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Command     string   `json:"command"`
	Args        []string `json:"args,omitempty"`
}

// Dir returns the host-tools directory.
func Dir() string {
	return filepath.Join(storage.ConfigDir(), "host_tools")
}

// Load reads all manifests from Dir().
func Load() ([]Manifest, error) {
	dir := Dir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []Manifest
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var m Manifest
		if err := json.Unmarshal(data, &m); err != nil {
			continue
		}
		if m.Name == "" || m.Command == "" {
			continue
		}
		out = append(out, m)
	}
	return out, nil
}

// Tool adapts a Manifest into a tool.Tool-compatible wrapper. The wrapper is
// defined in the tool package to avoid an import cycle; here we expose a
// constructor returning an opaque value. Callers should use hosttools.AsTool.
//
// Run executes a host tool with the given JSON input and returns stdout.
func Run(ctx context.Context, m Manifest, input string) (string, error) {
	args := make([]string, len(m.Args))
	for i, a := range m.Args {
		args[i] = strings.ReplaceAll(a, "{input}", input)
	}
	cmd := exec.CommandContext(ctx, m.Command, args...) // #nosec G204 -- command comes from a trusted local manifest
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("host tool %s failed: %w: %s", m.Name, err, string(out))
	}
	return string(out), nil
}
