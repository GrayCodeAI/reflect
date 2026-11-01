//go:build !windows

package tool

import (
	"os"
	"os/exec"
	"syscall"
	"time"
)

func setCmdProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	// CommandContext otherwise kills only the shell. Descendants may keep the
	// output pipes open indefinitely after the shell's deadline expires.
	if cmd.Cancel != nil {
		cmd.Cancel = func() error { return killProcessGroup(cmd.Process) }
	}
	cmd.WaitDelay = time.Second
}

func killProcessGroup(proc *os.Process) error {
	err := syscall.Kill(-proc.Pid, syscall.SIGKILL)
	if err == syscall.ESRCH {
		return os.ErrProcessDone
	}
	return err
}
