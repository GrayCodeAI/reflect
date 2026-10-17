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
	ModeOff          Mode = "off"
	ModeDocker       Mode = "docker"
	ModeMicrosandbox Mode = "microsandbox"
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
	case ModeOff, "":
		return true // host is always "available"
	case ModeDocker:
		_, err := exec.LookPath("docker")
		return err == nil
	case ModeMicrosandbox:
		_, err := exec.LookPath("msb")
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

// MicrosandboxAvailable reports whether the msb CLI is on PATH.
func MicrosandboxAvailable() bool {
	_, err := exec.LookPath("msb")
	return err == nil
}

// Run executes command inside a sandbox with workdir bind-mounted.
// Returns combined stdout+stderr and an error. Falls back to host execution
// when mode is off.
func Run(ctx context.Context, cfg Config, workdir, command string) (string, error) {
	if cfg.Mode == ModeOff || cfg.Mode == "" {
		// Caller should handle host execution; sandbox.Run is a no-op passthrough
		// marker. Return empty to signal "not sandboxed".
		return "", fmt.Errorf("sandbox: mode off")
	}
	image := cfg.Image
	if image == "" {
		image = DefaultImage
	}
	switch cfg.Mode {
	case ModeDocker:
		if !DockerAvailable() {
			return "", fmt.Errorf("sandbox: docker not found on PATH")
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
	case ModeMicrosandbox:
		if !MicrosandboxAvailable() {
			return "", fmt.Errorf("sandbox: msb (microsandbox) not found on PATH")
		}
		// microsandbox CLI: msb run <image> -- <command>
		args := []string{"run", image, "--", "sh", "-c", command}
		out, err := exec.CommandContext(ctx, "msb", args...).CombinedOutput() // #nosec G204 -- args are controlled
		return string(out), err
	}
	return "", fmt.Errorf("sandbox: unknown mode %q", cfg.Mode)
}

// Enabled reports whether sandboxing is active.
func Enabled(cfg Config) bool {
	return cfg.Mode == ModeDocker || cfg.Mode == ModeMicrosandbox
}

// NormalizeMode coerces a raw string into a Mode.
func NormalizeMode(raw string) Mode {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "docker", "container":
		return ModeDocker
	case "microsandbox", "msb", "micro-vm", "microvm":
		return ModeMicrosandbox
	case "off", "":
		return ModeOff
	}
	return ModeOff
}
