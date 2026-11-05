package tui

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/LukeTarr/checkpoint/internal/checkpoints"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func update(m model, msg tea.Msg) (model, tea.Cmd) {
	next, cmd := m.Update(msg)
	return next.(model), cmd
}

func key(value string) tea.KeyMsg {
	switch value {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(value)}
	}
}

func finishOperation(t *testing.T, m model, cmd tea.Cmd) model {
	t.Helper()
	if cmd == nil || !m.busy {
		t.Fatal("expected an asynchronous operation")
	}
	m, cmd = update(m, cmd())
	if strings.HasPrefix(m.status, "Error:") || cmd == nil {
		t.Fatalf("operation failed: %s", m.status)
	}
	m, _ = update(m, cmd())
	if m.busy || strings.HasPrefix(m.status, "Error:") {
		t.Fatalf("reload failed: %s", m.status)
	}
	return m
}

func TestInteractiveLifecycle(t *testing.T) {
	root := t.TempDir()
	if out, err := exec.Command("git", "-C", root, "init", "--quiet").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	path := filepath.Join(root, "example.txt")
	if err := os.WriteFile(path, []byte("original\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := newModel(root)
	m, _ = update(m, m.Init()())
	if m.busy || !strings.Contains(m.View(), "No checkpoints found") {
		t.Fatal("empty state not shown")
	}
	m, _ = update(m, key("n"))
	m, _ = update(m, key("saved"))
	m, cmd := update(m, key("enter"))
	// Quitting cannot interrupt a snapshot halfway through.
	if _, quit := update(m, key("q")); quit != nil {
		t.Fatal("quit accepted during operation")
	}
	m = finishOperation(t, m, cmd)
	if len(m.list.Items()) != 1 || m.list.Items()[0].(checkpointItem).meta.Name != "saved" {
		t.Fatal("new checkpoint missing from list")
	}
	if err := os.WriteFile(path, []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m, cmd = update(m, key("enter"))
	if cmd != nil || m.screen != confirmRestore || !strings.Contains(m.View(), "overwrite files") {
		t.Fatal("restore did not request confirmation")
	}
	m, cmd = update(m, key("enter"))
	if cmd != nil || m.screen != browsing {
		t.Fatal("restore confirmation did not default to cancel")
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "changed\n" {
		t.Fatal("cancelled restore changed files")
	}
	m, _ = update(m, key("enter"))
	m, cmd = update(m, key("y"))
	m = finishOperation(t, m, cmd)
	data, err = os.ReadFile(path)
	if err != nil || string(data) != "original\n" {
		t.Fatalf("confirmed restore failed: %q, %v", data, err)
	}
	m, cmd = update(m, key("x"))
	if cmd != nil || m.screen != confirmDelete {
		t.Fatal("delete did not request confirmation")
	}
	m, cmd = update(m, key("n"))
	if cmd != nil || m.screen != browsing || len(m.list.Items()) != 1 {
		t.Fatal("delete cancellation failed")
	}
	m, _ = update(m, key("x"))
	m, cmd = update(m, key("y"))
	m = finishOperation(t, m, cmd)
	if len(m.list.Items()) != 0 {
		t.Fatal("confirmed delete left checkpoints")
	}
	_, cmd = update(m, key("q"))
	if cmd == nil {
		t.Fatal("quit did not return a command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("quit did not exit")
	}
}

func TestNameEntryAndErrors(t *testing.T) {
	m := newModel(t.TempDir())
	m.busy = false
	m, _ = update(m, key("n"))
	m, _ = update(m, key("../invalid"))
	m, cmd := update(m, key("enter"))
	if cmd != nil || m.busy || m.screen != naming || !strings.Contains(m.status, "path separators") {
		t.Fatal("invalid checkpoint name was accepted")
	}
	m, _ = update(m, key("esc"))
	if m.screen != browsing {
		t.Fatal("escape did not cancel name entry")
	}
	m, _ = update(m, key("n"))
	m, cmd = update(m, key("enter"))
	if cmd == nil || !m.busy {
		t.Fatal("blank name did not start creation")
	}
	// A filesystem error is displayed and returns to browsing so the user can retry.
	m, _ = update(m, operationMsg{err: errors.New("disk full")})
	if m.busy || m.screen != browsing || !strings.Contains(m.View(), "disk full") {
		t.Fatal("operation error was not displayed")
	}
	m, _ = update(m, loadedMsg{err: errors.New("permission denied")})
	if m.busy || !strings.Contains(m.View(), "permission denied") {
		t.Fatal("load error was not displayed")
	}
	m, cmd = update(m, key("r"))
	if !m.busy || cmd == nil {
		t.Fatal("refresh did not reload checkpoints")
	}
}

func TestListOrderingNavigationAndResize(t *testing.T) {
	root := t.TempDir()
	for i, name := range []string{"older", "newer"} {
		dir := filepath.Join(root, checkpoints.DirName, name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := checkpoints.WriteMeta(dir, checkpoints.Meta{Name: name, CreatedAt: time.Unix(int64(i), 0)}); err != nil {
			t.Fatal(err)
		}
	}
	m := newModel(root)
	m, _ = update(m, m.Init()())
	if m.list.Items()[0].(checkpointItem).meta.Name != "newer" {
		t.Fatal("list is not newest first")
	}
	m, _ = update(m, tea.WindowSizeMsg{Width: 40, Height: 15})
	m, _ = update(m, key("down"))
	if m.list.SelectedItem().(checkpointItem).meta.Name != "older" {
		t.Fatal("navigation did not select the next checkpoint")
	}
	if m.list.Width() != 38 || m.list.Height() != 5 {
		t.Fatal("list did not resize")
	}
	// Clearing the list also makes restore and delete harmless.
	m, _ = update(m, loadedMsg{items: []list.Item{}})
	for _, value := range []string{"enter", "x"} {
		var cmd tea.Cmd
		m, cmd = update(m, key(value))
		if cmd != nil || m.screen != browsing {
			t.Fatalf("%s acted on an empty list", value)
		}
	}
}
