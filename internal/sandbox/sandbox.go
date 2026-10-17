// Package sandbox provides opt-in isolated execution for Reflect's tools.
//
// Mirror (reflection-oss/mirror) runs file/shell tools inside a sandbox
// (microsandbox/container) instead of on the host. Reflect mirrors that with an
// optional Docker-based sandbox: when enabled, shell commands run inside a
// container with the working directory bind-mounted, so the agent can execute
// arbitrary commands without touching the host filesystem directly.
//
// Default mode is "off" (host execution), preserving existing behavior.
package sandbox

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Mode describes the sandbox backend.
type Mode string

const (
	ModeOff    Mode = "off"
	ModeDocker Mode = "docker"
)

// Config holds sandbox settings.
type Config struct {
	Mode  Mode
	Image string
}

// DefaultImage is used when no image is configured.
const DefaultImage = "alpine:latest"

// Available reports whether the configured backend is usable on this machine.
func Available(cfg Config) bool {
	switch cfg.Mode {
	case ModeDocker, "":
		// empty mode treated as "try docker if present"? No — explicit off.
		if cfg.Mode == ModeOff {
			return true // host is always "available"
		}
		_, err := exec.LookPath("docker")
		return err == nil
	}
	return false
}

// DockerAvailable reports whether the docker binary is on PATH and the daemon
// is reachable.
func DockerAvailable() bool {
	if _, err := exec.LookPath("docker"); err != nil {
		return false
	}
	return exec.CommandContext(context.Background(), "docker", "info").Run() == nil
}

// Run executes command inside a container with workdir bind-mounted at /work.
// Returns combined stdout+stderr and an error. Falls back to host execution
// when mode is off.
func Run(ctx context.Context, cfg Config, workdir, command string) (string, error) {
	if cfg.Mode == ModeOff || cfg.Mode == "" {
		// Caller should handle host execution; sandbox.Run is a no-op passthrough
		// marker. Return empty to signal "not sandboxed".
		return "", fmt.Errorf("sandbox: mode off")
	}
	if !DockerAvailable() {
		return "", fmt.Errorf("sandbox: docker not found on PATH")
	}
	image := cfg.Image
	if image == "" {
		image = DefaultImage
	}
	args := []string{
		"run", "--rm",
		"-v", workdir + ":/work",
		"-w", "/work",
		image,
		"sh", "-c", command,
	}
	out, err := exec.CommandContext(ctx, "docker", args...).CombinedOutput() // #nosec G204 -- args are controlled (image + bind mount + sh -c)
	return string(out), err
}

// Enabled reports whether sandboxing is active.
func Enabled(cfg Config) bool {
	return cfg.Mode == ModeDocker
}

// NormalizeMode coerces a raw string into a Mode.
func NormalizeMode(raw string) Mode {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "docker", "container":
		return ModeDocker
	case "off", "":
		return ModeOff
	}
	return ModeOff
}
