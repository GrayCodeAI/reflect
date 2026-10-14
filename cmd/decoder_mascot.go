package cmd

import (
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/GrayCodeAI/decoder/internal/features/welcome"
)

func decoderMascotEnabled() bool {
	// Text art is the default; raster graphics are explicitly opt-in.
	return os.Getenv("DECODER_GRAPHICS") == "1" && welcome.Enabled()
}

func emitDecoderMascotCmd() tea.Cmd {
	return func() tea.Msg {
		_ = welcome.Emit(os.Stderr)
		return nil
	}
}
