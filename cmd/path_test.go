package cmd

import (
	"context"
	"strings"
	"testing"

	reflectconfig "github.com/GrayCodeAI/reflect/internal/config"
	"github.com/GrayCodeAI/reflect/internal/provider/gateway"
)

func TestPathCmdRuns(t *testing.T) {
	useInMemoryCredentials(t)
	if err := pathCmd.RunE(pathCmd, nil); err == nil {
		t.Skip("machine has full developer path setup") // TODO: https://github.com/GrayCodeAI/reflect/issues/30
	}
}

func TestPathCmdPrintsReport(t *testing.T) {
	useInMemoryCredentials(t)
	out := reflectconfig.FormatDeveloperPathReport(context.Background())
	if !strings.Contains(out, "Developer path") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func useInMemoryCredentials(t *testing.T) {
	t.Helper()
	gateway.SetDefaultStore(&gateway.MapStore{})
	t.Cleanup(func() { gateway.SetDefaultStore(nil) })
}
