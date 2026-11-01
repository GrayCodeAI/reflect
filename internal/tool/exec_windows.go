//go:build windows

package tool

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

func setCmdProcessGroup(cmd *exec.Cmd) {
	if cmd.Cancel != nil {
		cmd.Cancel = func() error { return killProcessGroup(cmd.Process) }
	}
	// Bound pipe draining even when descendants retain inherited handles.
	cmd.WaitDelay = time.Second
}

func killProcessGroup(proc *os.Process) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// taskkill /T terminates descendants as well as the shell. Resolve the
	// system executable explicitly rather than trusting a project PATH entry.
	taskkill := filepath.Join(os.Getenv("SystemRoot"), "System32", "taskkill.exe")
	if err := exec.CommandContext(ctx, taskkill, "/PID", strconv.Itoa(proc.Pid), "/T", "/F").Run(); err == nil { // #nosec G204 -- system taskkill with numeric PID
		return nil
	}
	return proc.Kill()
}
