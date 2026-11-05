package cmd

import (
	"github.com/LukeTarr/checkpoint/internal/checkpoints"
	"github.com/LukeTarr/checkpoint/internal/tui"
	"github.com/spf13/cobra"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Open the interactive checkpoint interface",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := checkpoints.FindRepoRoot()
		if err != nil {
			return err
		}
		return tui.Run(root)
	},
}
