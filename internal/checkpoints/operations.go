package checkpoints

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func DefaultName() string {
	return fmt.Sprintf("checkpoint-%s", time.Now().UTC().Format("20060102-150405"))
}

// Create snapshots the repository without prompting or writing to the terminal.
func Create(root, name string, force bool) error {
	if err := ValidateCheckpointName(name); err != nil {
		return err
	}
	dir := filepath.Join(root, DirName, name)
	if _, err := os.Stat(dir); err == nil {
		if !force {
			return fmt.Errorf("checkpoint '%s' already exists (use --force to overwrite)", name)
		}
		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("remove existing checkpoint: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("check checkpoint: %w", err)
	}
	if err := os.MkdirAll(dir, DirPerm); err != nil {
		return fmt.Errorf("create checkpoint: %w", err)
	}
	files, err := ListRepoFiles(root)
	if err != nil {
		return err
	}
	for _, rel := range files {
		if err := CopyEntry(filepath.Join(root, rel), filepath.Join(dir, rel)); err != nil {
			return fmt.Errorf("copy '%s': %w", rel, err)
		}
	}
	return WriteMeta(dir, Meta{Name: name, CreatedAt: time.Now().UTC()})
}

// Restore replaces repository files with a checkpoint. Callers must obtain any
// required user confirmation before calling it.
func Restore(root, name string) error {
	if err := ValidateCheckpointName(name); err != nil {
		return err
	}
	dir := filepath.Join(root, DirName, name)
	if _, err := os.Stat(dir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("checkpoint '%s' not found", name)
		}
		return fmt.Errorf("check checkpoint: %w", err)
	}
	files, err := ListCheckpointFiles(dir)
	if err != nil {
		return err
	}
	current, err := ListRepoFiles(root)
	if err != nil {
		return err
	}
	set := make(map[string]struct{}, len(files))
	for _, rel := range files {
		set[rel] = struct{}{}
	}
	var removalDirs []string
	for _, rel := range current {
		if _, exists := set[rel]; exists {
			continue
		}
		if err := os.Remove(filepath.Join(root, rel)); err != nil {
			return fmt.Errorf("remove '%s': %w", rel, err)
		}
		removalDirs = append(removalDirs, filepath.Dir(rel))
	}
	if err := RemoveEmptyDirs(root, removalDirs, RequiredDirs(files)); err != nil {
		return err
	}
	for _, rel := range files {
		if err := CopyEntry(filepath.Join(dir, rel), filepath.Join(root, rel)); err != nil {
			return fmt.Errorf("restore '%s': %w", rel, err)
		}
	}
	return nil
}

// DeleteAll removes the repository's checkpoints after caller confirmation.
func DeleteAll(root string) error {
	if err := os.RemoveAll(filepath.Join(root, DirName)); err != nil {
		return fmt.Errorf("delete checkpoints: %w", err)
	}
	return nil
}
