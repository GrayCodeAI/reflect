package cmd

import (
	"context"
	"strings"

	"github.com/GrayCodeAI/reflect/internal/engine"
)

func runSlashShellCommand(m *chatModel, command string) (string, bool) {
	result := runSlashShellCommandResult(m, command)
	return result.Output, result.IsError
}

func runSlashShellCommandResult(m *chatModel, command string) engine.ShellCommandResult {
	if m == nil || m.session == nil {
		return engine.ShellCommandResult{Output: "Error: no active session", IsError: true}
	}
	result := m.session.RunUserShellCommandResult(context.Background(), command, 0)
	result.Output = strings.TrimSpace(result.Output)
	return result
}

func shellCommandFailed(output string, isErr bool) bool {
	if isErr {
		return true
	}
	return strings.Contains(output, "exit code:") || strings.Contains(output, "(command timed out)")
}
