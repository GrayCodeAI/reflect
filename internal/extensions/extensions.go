// Package extensions provides a lightweight extension system inspired by
// Mirror's Python extensions. Extensions live in ~/.reflect/extensions/<name>/
// and contribute commands (shell or python scripts) and host tools.
//
// An extension directory must contain a manifest.json:
//
//	{
//	  "name": "my-ext",
//	  "description": "...",
//	  "commands": [{"name": "hello", "description": "...", "script": "hello.py"}]
//	}
package extensions

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/GrayCodeAI/reflect/internal/storage"
)

// Command describes an extension-provided command.
type Command struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Script      string `json:"script"` // relative to extension dir
}

// Manifest describes an extension.
type Manifest struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Commands    []Command `json:"commands,omitempty"`
}

// Dir returns the extensions directory.
func Dir() string {
	return filepath.Join(storage.ConfigDir(), "extensions")
}

// Load reads all extension manifests.
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
		if !e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name(), "manifest.json"))
		if err != nil {
			continue
		}
		var m Manifest
		if err := json.Unmarshal(data, &m); err != nil {
			continue
		}
		if m.Name == "" {
			m.Name = e.Name()
		}
		out = append(out, m)
	}
	return out, nil
}

// RunCommand executes an extension command script and returns its stdout.
func RunCommand(ctx context.Context, ext Manifest, cmd Command, args []string) (string, error) {
	script := filepath.Join(Dir(), ext.Name, cmd.Script)
	c := exec.CommandContext(ctx, "python3", append([]string{script}, args...)...) // #nosec G204 -- script path from trusted local manifest
	// fall back to sh if python3 missing
	if _, err := exec.LookPath("python3"); err != nil {
		c = exec.CommandContext(ctx, "sh", append([]string{script}, args...)...) // #nosec G204 -- script path from trusted local manifest
	}
	out, err := c.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("extension %s/%s failed: %w: %s", ext.Name, cmd.Name, err, string(out))
	}
	return string(out), nil
}

// FindCommand looks up an extension command by name.
func FindCommand(name string) (Manifest, Command, bool) {
	exts, err := Load()
	if err != nil {
		return Manifest{}, Command{}, false
	}
	for _, e := range exts {
		for _, c := range e.Commands {
			if c.Name == name {
				return e, c, true
			}
		}
	}
	return Manifest{}, Command{}, false
}

// ListCommandNames returns all extension command names.
func ListCommandNames() []string {
	exts, _ := Load()
	var out []string
	for _, e := range exts {
		for _, c := range e.Commands {
			out = append(out, c.Name)
		}
	}
	return out
}
