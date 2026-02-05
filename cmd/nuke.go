package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/LukeTarr/checkpoint/internal/checkpoints"
	"github.com/spf13/cobra"
)

var nukeCmd = &cobra.Command{
	Use:   "nuke",
	Short: "Delete all checkpoints",
	Args:  cobra.NoArgs,
	RunE:  runNuke,
}

func runNuke(cmd *cobra.Command, args []string) error {
	root, err := checkpoints.FindRepoRoot()
	if err != nil {
		return err
	}

	checkpointsRoot := filepath.Join(root, checkpoints.DirName)
	if _, err := os.Stat(checkpointsRoot); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Println("No checkpoints to delete")
			return nil
		}
		return fmt.Errorf("check checkpoints: %w", err)
	}

	if err := confirmNuke(); err != nil {
		return err
	}

	if err := os.RemoveAll(checkpointsRoot); err != nil {
		return fmt.Errorf("delete checkpoints: %w", err)
	}

	fmt.Println("All checkpoints deleted")
	return nil
}

func confirmNuke() error {
	fmt.Print("Delete all checkpoints? This cannot be undone. [y/N]: ")
	return confirmAnswer(os.Stdin)
}
