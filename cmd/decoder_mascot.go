package cmd

import (
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/GrayCodeAI/decoder/internal/features/welcome"
)

func decoderMascotEnabled() bool {
	return welcome.Enabled()
}

func emitDecoderMascotCmd() tea.Cmd {
	return func() tea.Msg {
		_ = welcome.Emit(os.Stderr)
		return nil
	}
}
