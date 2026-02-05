package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/LukeTarr/checkpoint/internal/checkpoints"
	"github.com/spf13/cobra"
)

var popCmd = &cobra.Command{
	Use:   "pop [name]",
	Short: "Restore from a checkpoint",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runPop,
}

func runPop(cmd *cobra.Command, args []string) error {
	root, err := checkpoints.FindRepoRoot()
	if err != nil {
		return err
	}

	name := ""
	if len(args) > 0 {
		name = strings.TrimSpace(args[0])
	}

	checkpointsRoot := filepath.Join(root, checkpoints.DirName)
	if name == "" {
		name, err = checkpoints.LatestCheckpointName(checkpointsRoot)
		if err != nil {
			return err
		}
	}
	if err := checkpoints.ValidateCheckpointName(name); err != nil {
		return err
	}

	checkpointDir := filepath.Join(checkpointsRoot, name)
	if _, err := os.Stat(checkpointDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("checkpoint '%s' not found", name)
		}
		return fmt.Errorf("check checkpoint: %w", err)
	}

	if err := confirmPop(name); err != nil {
		return err
	}

	checkpointFiles, err := checkpoints.ListCheckpointFiles(checkpointDir)
	if err != nil {
		return err
	}

	currentFiles, err := checkpoints.ListRepoFiles(root)
	if err != nil {
		return err
	}

	checkpointSet := make(map[string]struct{}, len(checkpointFiles))
	for _, rel := range checkpointFiles {
		checkpointSet[rel] = struct{}{}
	}
	requiredDirs := checkpoints.RequiredDirs(checkpointFiles)
	removalDirs := make([]string, 0, len(currentFiles))

	for _, rel := range currentFiles {
		if _, exists := checkpointSet[rel]; exists {
			continue
		}
		path := filepath.Join(root, rel)
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("remove '%s': %w", rel, err)
		}
		removalDirs = append(removalDirs, filepath.Dir(rel))
	}

	if err := checkpoints.RemoveEmptyDirs(root, removalDirs, requiredDirs); err != nil {
		return err
	}

	for _, rel := range checkpointFiles {
		srcPath := filepath.Join(checkpointDir, rel)
		dstPath := filepath.Join(root, rel)
		if err := checkpoints.CopyEntry(srcPath, dstPath); err != nil {
			return fmt.Errorf("restore '%s': %w", rel, err)
		}
	}

	fmt.Printf("Restored checkpoint '%s'\n", name)
	return nil
}

func confirmPop(name string) error {
	fmt.Printf("Restore checkpoint '%s'? This will overwrite files. [y/N]: ", name)
	if err := confirmAnswer(os.Stdin); err != nil {
		return errors.New("restore cancelled")
	}
	return nil
}
