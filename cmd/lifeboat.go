package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/GrayCodeAI/reflect/internal/session"
	"github.com/spf13/cobra"
)

var lifeboatCmd = &cobra.Command{
	Use:   "lifeboat",
	Short: "Inspect or remove saved lifeboats captured before compaction",
}

var lifeboatShowCmd = &cobra.Command{
	Use:   "show <session-id>",
	Short: "Print the saved lifeboat (if any) for a session",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		state, err := session.LoadLifeboat(args[0])
		if err != nil {
			return err
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(state)
	},
}

var lifeboatRemoveCmd = &cobra.Command{
	Use:   "remove <session-id>",
	Short: "Delete the saved lifeboat for a session",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		if err := session.RemoveLifeboat(args[0]); err != nil {
			return err
		}
		fmt.Printf("removed lifeboat for %s\n", args[0])
		return nil
	},
}

func init() {
	lifeboatCmd.AddCommand(lifeboatShowCmd, lifeboatRemoveCmd)
	rootCmd.AddCommand(lifeboatCmd)
}
