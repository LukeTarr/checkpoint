package cmd

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/LukeTarr/checkpoint/internal/checkpoints"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available checkpoints",
	Args:  cobra.NoArgs,
	RunE:  runList,
}

func runList(cmd *cobra.Command, args []string) error {
	root, err := checkpoints.FindRepoRoot()
	if err != nil {
		return err
	}

	checkpointsRoot := filepath.Join(root, checkpoints.DirName)
	metas, err := checkpoints.ListCheckpoints(checkpointsRoot)
	if err != nil {
		return err
	}
	if len(metas) == 0 {
		fmt.Println("No checkpoints found")
		return nil
	}

	sort.Slice(metas, func(i, j int) bool {
		return metas[i].CreatedAt.After(metas[j].CreatedAt)
	})

	for _, meta := range metas {
		checkpointDir := filepath.Join(checkpointsRoot, meta.Name)
		stats, err := checkpoints.CheckpointStats(checkpointDir)
		if err != nil {
			return err
		}
		fmt.Printf("%s\t%s\t%s\n", meta.CreatedAt.Format("2006-01-02 15:04:05 UTC"), formatStats(stats), meta.Name)
	}
	return nil
}

func formatStats(stats checkpoints.Stats) string {
	parts := make([]string, 0, 2)
	parts = append(parts, fmt.Sprintf("%d files", stats.FileCount))
	parts = append(parts, fmt.Sprintf("%s loc", formatCount(stats.LineCount)))
	return strings.Join(parts, ", ")
}

func formatCount(value int) string {
	if value >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(value)/1_000_000)
	}
	if value >= 1_000 {
		return fmt.Sprintf("%.1fk", float64(value)/1_000)
	}
	return fmt.Sprintf("%d", value)
}
