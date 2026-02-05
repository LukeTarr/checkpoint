package checkpoints

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const DirName = ".checkpoints"
const metaFileName = "meta.json"

const (
	DirPerm  os.FileMode = 0o755
	FilePerm os.FileMode = 0o644
)

type Meta struct {
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

func FindRepoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}

	current := wd
	for {
		gitPath := filepath.Join(current, ".git")
		if info, err := os.Stat(gitPath); err == nil && info.IsDir() {
			return current, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	return "", errors.New("no .git directory found (run inside a git repository)")
}

func ListRepoFiles(root string) ([]string, error) {
	cmd := exec.Command("git", "-C", root, "ls-files", "-co", "--exclude-standard", "-z")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-files failed: %w", err)
	}

	entries := strings.Split(string(output), "\x00")
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry == "" {
			continue
		}
		if IsCheckpointPath(entry) {
			continue
		}
		files = append(files, entry)
	}
	return files, nil
}

func ListCheckpointFiles(checkpointDir string) ([]string, error) {
	var files []string
	rootLen := len(checkpointDir) + 1

	if err := filepath.WalkDir(checkpointDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == checkpointDir {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		rel := path[rootLen:]
		if rel == metaFileName {
			return nil
		}
		files = append(files, rel)
		return nil
	}); err != nil {
		return nil, fmt.Errorf("read checkpoint: %w", err)
	}

	sort.Strings(files)
	return files, nil
}

func CopyEntry(srcPath, dstPath string) error {
	info, err := os.Lstat(srcPath)
	if err != nil {
		return err
	}

	if info.Mode()&os.ModeSymlink != 0 {
		return copySymlink(srcPath, dstPath)
	}
	if info.IsDir() {
		return os.MkdirAll(dstPath, DirPerm)
	}
	return copyFile(srcPath, dstPath, info.Mode())
}

func WriteMeta(checkpointDir string, meta Meta) error {
	path := filepath.Join(checkpointDir, metaFileName)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, FilePerm)
	if err != nil {
		return fmt.Errorf("write metadata: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(meta); err != nil {
		return fmt.Errorf("write metadata: %w", err)
	}
	return nil
}

func LatestCheckpointName(checkpointsRoot string) (string, error) {
	entries, err := os.ReadDir(checkpointsRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", errors.New("no checkpoints found")
		}
		return "", fmt.Errorf("read checkpoints: %w", err)
	}

	var candidates []Meta
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		meta, err := ReadMeta(filepath.Join(checkpointsRoot, name))
		if err != nil {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			candidates = append(candidates, Meta{Name: name, CreatedAt: info.ModTime()})
			continue
		}
		candidates = append(candidates, meta)
	}

	if len(candidates) == 0 {
		return "", errors.New("no checkpoints found")
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].CreatedAt.After(candidates[j].CreatedAt)
	})
	return candidates[0].Name, nil
}

func ListCheckpoints(checkpointsRoot string) ([]Meta, error) {
	entries, err := os.ReadDir(checkpointsRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read checkpoints: %w", err)
	}

	metas := make([]Meta, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		meta, err := ReadMeta(filepath.Join(checkpointsRoot, name))
		if err != nil {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			metas = append(metas, Meta{Name: name, CreatedAt: info.ModTime()})
			continue
		}
		metas = append(metas, meta)
	}
	return metas, nil
}

func ReadMeta(checkpointDir string) (Meta, error) {
	path := filepath.Join(checkpointDir, metaFileName)
	file, err := os.Open(path)
	if err != nil {
		return Meta{}, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	var meta Meta
	if err := decoder.Decode(&meta); err != nil {
		return Meta{}, err
	}
	return meta, nil
}

func IsCheckpointPath(rel string) bool {
	if rel == DirName {
		return true
	}
	if strings.HasPrefix(rel, DirName+string(filepath.Separator)) {
		return true
	}
	return false
}

func ValidateCheckpointName(name string) error {
	if name == "" {
		return errors.New("checkpoint name cannot be empty")
	}
	if strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return errors.New("checkpoint name cannot contain path separators")
	}
	if name == "." || name == ".." {
		return errors.New("checkpoint name is invalid")
	}
	return nil
}

type Stats struct {
	FileCount int
	LineCount int
}

func RequiredDirs(files []string) map[string]struct{} {
	required := make(map[string]struct{})
	for _, rel := range files {
		dir := filepath.Dir(rel)
		for dir != "." && dir != string(filepath.Separator) {
			required[dir] = struct{}{}
			next := filepath.Dir(dir)
			if next == dir {
				break
			}
			dir = next
		}
	}
	return required
}

func RemoveEmptyDirs(root string, candidates []string, required map[string]struct{}) error {
	unique := make(map[string]struct{}, len(candidates))
	for _, dir := range candidates {
		clean := filepath.Clean(dir)
		if clean == "." || clean == string(filepath.Separator) {
			continue
		}
		if IsCheckpointPath(clean) || clean == ".git" || strings.HasPrefix(clean, ".git"+string(filepath.Separator)) {
			continue
		}
		unique[clean] = struct{}{}
	}

	sorted := make([]string, 0, len(unique))
	for dir := range unique {
		sorted = append(sorted, dir)
	}
	sort.Slice(sorted, func(i, j int) bool {
		return strings.Count(sorted[i], string(filepath.Separator)) > strings.Count(sorted[j], string(filepath.Separator))
	})

	for _, dir := range sorted {
		if _, ok := required[dir]; ok {
			continue
		}
		path := filepath.Join(root, dir)
		if err := os.Remove(path); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			var pathErr *os.PathError
			if errors.As(err, &pathErr) {
				switch pathErr.Err {
				case os.ErrPermission:
					continue
				case os.ErrInvalid:
					continue
				}
				if strings.Contains(pathErr.Err.Error(), "directory not empty") {
					continue
				}
			}
			if strings.Contains(err.Error(), "directory not empty") {
				continue
			}
			return fmt.Errorf("remove directory '%s': %w", dir, err)
		}
	}

	return nil
}

func CheckpointStats(checkpointDir string) (Stats, error) {
	files, err := ListCheckpointFiles(checkpointDir)
	if err != nil {
		return Stats{}, err
	}

	stats := Stats{FileCount: len(files)}
	for _, rel := range files {
		path := filepath.Join(checkpointDir, rel)
		count, err := countLines(path)
		if err != nil {
			return Stats{}, err
		}
		stats.LineCount += count
	}

	return stats, nil
}

func copySymlink(srcPath, dstPath string) error {
	link, err := os.Readlink(srcPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dstPath), DirPerm); err != nil {
		return err
	}
	if err := os.RemoveAll(dstPath); err != nil {
		return err
	}
	return os.Symlink(link, dstPath)
}

func copyFile(srcPath, dstPath string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dstPath), DirPerm); err != nil {
		return err
	}

	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.OpenFile(dstPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode.Perm())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}
	return os.Chmod(dstPath, mode.Perm())
}

func countLines(path string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	buf := make([]byte, 32*1024)
	lines := 0
	sawData := false
	lastByte := byte(0)

	for {
		n, err := file.Read(buf)
		if n > 0 {
			sawData = true
			for _, b := range buf[:n] {
				if b == '\n' {
					lines++
				}
				lastByte = b
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return 0, err
		}
	}

	if sawData && lastByte != '\n' {
		lines++
	}

	return lines, nil
}
