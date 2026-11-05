package checkpoints

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCheckpointLifecycle(t *testing.T) {
	root := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		if out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	read := func(name string) string {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	git("init", "--quiet")
	write(".gitignore", "ignored.txt\n")
	write("tracked.txt", "original\n")
	git("add", "tracked.txt", ".gitignore")
	write("untracked.txt", "also saved\n")
	write("ignored.txt", "ignored\n")
	if err := Create(root, "before", false); err != nil {
		t.Fatal(err)
	}
	write("tracked.txt", "changed\n")
	write("untracked.txt", "changed too\n")
	write("new.txt", "remove on restore\n")
	write("ignored.txt", "keep ignored changes\n")
	if err := Create(root, "before", false); err == nil {
		t.Fatal("duplicate checkpoint should be rejected")
	}
	if _, err := os.Stat(filepath.Join(root, DirName, "before", "ignored.txt")); !os.IsNotExist(err) {
		t.Fatalf("ignored file was included: %v", err)
	}
	if err := Restore(root, "before"); err != nil {
		t.Fatal(err)
	}
	if got := read("tracked.txt"); got != "original\n" {
		t.Fatalf("tracked content = %q", got)
	}
	if got := read("untracked.txt"); got != "also saved\n" {
		t.Fatalf("untracked content = %q", got)
	}
	if got := read("ignored.txt"); got != "keep ignored changes\n" {
		t.Fatalf("ignored content = %q", got)
	}
	if _, err := os.Stat(filepath.Join(root, "new.txt")); !os.IsNotExist(err) {
		t.Fatalf("new file was not removed: %v", err)
	}
	write("tracked.txt", "replacement\n")
	if err := Create(root, "before", true); err != nil {
		t.Fatal(err)
	}
	if err := Restore(root, "before"); err != nil {
		t.Fatal(err)
	}
	if got := read("tracked.txt"); got != "replacement\n" {
		t.Fatalf("forced snapshot content = %q", got)
	}
	files, err := ListCheckpointFiles(filepath.Join(root, DirName, "before"))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range files {
		if IsCheckpointPath(name) {
			t.Fatalf("nested checkpoint included: %s", name)
		}
	}
	if err := DeleteAll(root); err != nil {
		t.Fatal(err)
	}
	if metas, err := ListCheckpoints(filepath.Join(root, DirName)); err != nil || len(metas) != 0 {
		t.Fatalf("checkpoints after deletion: %v, %v", metas, err)
	}
}

func TestOperationsRejectInvalidNames(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"", ".", "..", "../outside", "nested/name", `nested\name`} {
		if err := Create(root, name, false); err == nil {
			t.Errorf("Create accepted %q", name)
		}
		if err := Restore(root, name); err == nil {
			t.Errorf("Restore accepted %q", name)
		}
	}
	if err := Restore(root, "missing"); err == nil {
		t.Fatal("Restore accepted a missing checkpoint")
	}
}
