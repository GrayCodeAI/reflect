package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/GrayCodeAI/reflect/internal/storage"

	mission "github.com/GrayCodeAI/reflect/internal/multiagent"
	"github.com/spf13/cobra"
)

var missionLineageCmd = &cobra.Command{
	Use:   "lineage <mission-dir>",
	Short: "Print a read-only lineage view for a finished mission",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		dir := args[0]
		if _, err := os.Stat(filepath.Join(dir, "mission.json")); err != nil {
			candidate := filepath.Join(storage.StateDir(), "missions", dir)
			if _, err2 := os.Stat(filepath.Join(candidate, "mission.json")); err2 == nil {
				dir = candidate
			}
		}
		m, err := mission.LoadMission(dir)
		if err != nil {
			return fmt.Errorf("load mission: %w", err)
		}
		fmt.Print(m.LineageMarkdown())
		// Mirror to stdout only; never mutate artifacts.
		if m.Dir == "" {
			_ = os.Stdout
		}
		return nil
	},
}

func init() {
	missionCmd.AddCommand(missionLineageCmd)
}
