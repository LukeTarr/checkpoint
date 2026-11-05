package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/LukeTarr/checkpoint/internal/checkpoints"
	"github.com/spf13/cobra"
)

var pushCmd = &cobra.Command{
	Use:   "push [name]",
	Short: "Create or update a checkpoint",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runPush,
}

func init() {
	pushCmd.Flags().Bool("force", false, "overwrite existing checkpoint")
}

func runPush(cmd *cobra.Command, args []string) error {
	root, err := checkpoints.FindRepoRoot()
	if err != nil {
		return err
	}

	name := ""
	if len(args) > 0 {
		name = strings.TrimSpace(args[0])
	}
	if name == "" {
		name = defaultCheckpointName()
	}
	if err := checkpoints.ValidateCheckpointName(name); err != nil {
		return err
	}

	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return err
	}

	if err := checkpoints.Create(root, name, force); err != nil {
		return err
	}

	fmt.Printf("Checkpoint '%s' created at %s\n", name, filepath.Join(root, checkpoints.DirName, name))
	return nil
}

func defaultCheckpointName() string {
	return checkpoints.DefaultName()
}
