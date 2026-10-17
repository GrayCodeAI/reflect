package cmd

import (
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/GrayCodeAI/reflect/internal/features/welcome"
)

func reflectMascotEnabled() bool {
	// Text art is the default; raster graphics are explicitly opt-in.
	return os.Getenv("REFLECT_GRAPHICS") == "1" && welcome.Enabled()
}

func emitReflectMascotCmd() tea.Cmd {
	return func() tea.Msg {
		_ = welcome.Emit(os.Stderr)
		return nil
	}
}
