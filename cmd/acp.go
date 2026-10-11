package cmd

import (
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/GrayCodeAI/decoder/internal/acp"
	"github.com/GrayCodeAI/decoder/internal/attachment"
	decoderconfig "github.com/GrayCodeAI/decoder/internal/config"
	"github.com/GrayCodeAI/decoder/internal/engine"
	"github.com/GrayCodeAI/decoder/internal/observability/logger"
	"github.com/GrayCodeAI/decoder/internal/storage"
	"github.com/spf13/cobra"
)

var acpCmd = &cobra.Command{
	Use:   "acp",
	Short: "Run decoder as an Agent Client Protocol (ACP) server",
	Long: "Run decoder as an ACP server over stdio (JSON-RPC 2.0) so editors such as " +
		"Zed can drive it. Tool-permission prompts are routed back to the client " +
		"via session/request_permission.",
	RunE: runACP,
}

func init() {
	rootCmd.AddCommand(acpCmd)
}

func runACP(cmd *cobra.Command, _ []string) error {
	settings := decoderconfig.LoadSettings()
	newSession := newConfiguredDecoderSessionFactory(settings, logger.New(io.Discard, logger.Error))

	factory := func() (*engine.Session, error) {
		systemPrompt, err := buildSystemPrompt()
		if err != nil {
			return nil, err
		}
		// stdout is the JSON-RPC channel; keep logs off it.
		return newSession(systemPrompt, "")
	}

	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	srv := acp.NewServer(factory)

	// Mount a durable attachment store for inline image admission, gated on
	// the resolved active model's vision support. When no deployment (or a
	// non-vision model) is configured, image capability stays false and the
	// server rejects image prompts rather than advertising support.
	store := attachment.NewFSStore(filepath.Join(storage.StateDir(), "attachments"))
	effectiveModel, _ := effectiveModelAndProvider(settings)
	srv.SetAttachmentStore(store, engine.ModelSupportsVision(effectiveModel))

	return srv.ServeStdio(ctx)
}
