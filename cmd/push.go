package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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

	checkpointsRoot := filepath.Join(root, checkpoints.DirName)
	checkpointDir := filepath.Join(checkpointsRoot, name)

	if _, err := os.Stat(checkpointDir); err == nil {
		if !force {
			return fmt.Errorf("checkpoint '%s' already exists (use --force to overwrite)", name)
		}
		if err := os.RemoveAll(checkpointDir); err != nil {
			return fmt.Errorf("remove existing checkpoint: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("check checkpoint: %w", err)
	}

	if err := os.MkdirAll(checkpointsRoot, checkpoints.DirPerm); err != nil {
		return fmt.Errorf("create checkpoints directory: %w", err)
	}
	if err := os.MkdirAll(checkpointDir, checkpoints.DirPerm); err != nil {
		return fmt.Errorf("create checkpoint: %w", err)
	}

	files, err := checkpoints.ListRepoFiles(root)
	if err != nil {
		return err
	}

	for _, rel := range files {
		srcPath := filepath.Join(root, rel)
		dstPath := filepath.Join(checkpointDir, rel)
		if err := checkpoints.CopyEntry(srcPath, dstPath); err != nil {
			return fmt.Errorf("copy '%s': %w", rel, err)
		}
	}

	meta := checkpoints.Meta{
		Name:      name,
		CreatedAt: time.Now().UTC(),
	}
	if err := checkpoints.WriteMeta(checkpointDir, meta); err != nil {
		return err
	}

	fmt.Printf("Checkpoint '%s' created at %s\n", name, checkpointDir)
	return nil
}

func defaultCheckpointName() string {
	return fmt.Sprintf("checkpoint-%s", time.Now().UTC().Format("20060102-150405"))
}
