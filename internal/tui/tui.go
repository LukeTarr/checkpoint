// Package tui provides the optional Bubble Tea checkpoint interface.
package tui

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/LukeTarr/checkpoint/internal/checkpoints"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type screen int

const (
	browsing screen = iota
	naming
	confirmRestore
	confirmDelete
)

type checkpointItem struct {
	meta  checkpoints.Meta
	stats checkpoints.Stats
}

func (i checkpointItem) Title() string { return i.meta.Name }
func (i checkpointItem) Description() string {
	return fmt.Sprintf("%s • %d files, %d loc", i.meta.CreatedAt.UTC().Format("2006-01-02 15:04:05 UTC"), i.stats.FileCount, i.stats.LineCount)
}
func (i checkpointItem) FilterValue() string { return i.meta.Name }

type loadedMsg struct {
	items  []list.Item
	status string
	err    error
}

type operationMsg struct {
	status string
	err    error
}

type model struct {
	root   string
	list   list.Model
	input  textinput.Model
	screen screen
	busy   bool
	status string
	target string
	width  int
	height int
}

func newModel(root string) model {
	l := list.New(nil, list.NewDefaultDelegate(), 80, 14)
	l.SetFilteringEnabled(false)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.DisableQuitKeybindings()
	input := textinput.New()
	input.Prompt = "Name: "
	input.Placeholder = "Leave blank for a timestamped name"
	input.CharLimit = 255
	input.Width = 60
	return model{root: root, list: l, input: input, busy: true, status: "Loading checkpoints…", width: 80, height: 24}
}

// Run opens a full-screen interface and restores the terminal on exit.
func Run(root string) error {
	_, err := tea.NewProgram(newModel(root), tea.WithAltScreen()).Run()
	if err != nil {
		return fmt.Errorf("run TUI: %w", err)
	}
	return nil
}

func (m model) Init() tea.Cmd { return load(m.root, "") }

func load(root, status string) tea.Cmd {
	return func() tea.Msg {
		dir := filepath.Join(root, checkpoints.DirName)
		metas, err := checkpoints.ListCheckpoints(dir)
		if err != nil {
			return loadedMsg{err: err}
		}
		sort.Slice(metas, func(i, j int) bool {
			if metas[i].CreatedAt.Equal(metas[j].CreatedAt) {
				return metas[i].Name < metas[j].Name
			}
			return metas[i].CreatedAt.After(metas[j].CreatedAt)
		})
		items := make([]list.Item, 0, len(metas))
		for _, meta := range metas {
			stats, err := checkpoints.CheckpointStats(filepath.Join(dir, meta.Name))
			if err != nil {
				return loadedMsg{err: err}
			}
			items = append(items, checkpointItem{meta: meta, stats: stats})
		}
		return loadedMsg{items: items, status: status}
	}
}

func operation(status string, action func() error) tea.Cmd {
	return func() tea.Msg { return operationMsg{status: status, err: action()} }
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.list.SetSize(max(1, msg.Width-2), max(3, msg.Height-10))
		m.input.Width = max(1, msg.Width-10)
		return m, nil
	case loadedMsg:
		m.busy = false
		if msg.err != nil {
			m.status = "Error: " + msg.err.Error()
			return m, nil
		}
		m.status = msg.status
		return m, m.list.SetItems(msg.items)
	case operationMsg:
		m.screen = browsing
		if msg.err != nil {
			m.busy = false
			m.status = "Error: " + msg.err.Error()
			return m, nil
		}
		m.status = msg.status
		return m, load(m.root, msg.status)
	case tea.KeyMsg:
		// Let file operations finish before accepting another action or quitting.
		if m.busy {
			return m, nil
		}
		key := msg.String()
		if key == "ctrl+c" {
			return m, tea.Quit
		}
		switch m.screen {
		case naming:
			switch key {
			case "esc":
				m.screen, m.status = browsing, "Create cancelled"
				m.input.Blur()
				return m, nil
			case "enter":
				name := strings.TrimSpace(m.input.Value())
				if name == "" {
					name = checkpoints.DefaultName()
				}
				if err := checkpoints.ValidateCheckpointName(name); err != nil {
					m.status = "Error: " + err.Error()
					return m, nil
				}
				m.input.Blur()
				m.busy, m.status = true, "Creating checkpoint…"
				return m, operation(fmt.Sprintf("Created checkpoint '%s'", name), func() error {
					return checkpoints.Create(m.root, name, false)
				})
			}
		case confirmRestore, confirmDelete:
			switch key {
			case "esc", "n", "N", "q", "enter":
				m.screen, m.status = browsing, "Operation cancelled"
				return m, nil
			case "y", "Y":
				m.busy = true
				if m.screen == confirmRestore {
					m.status = "Restoring checkpoint…"
					return m, operation(fmt.Sprintf("Restored checkpoint '%s'", m.target), func() error {
						return checkpoints.Restore(m.root, m.target)
					})
				}
				m.status = "Deleting checkpoints…"
				return m, operation("All checkpoints deleted", func() error { return checkpoints.DeleteAll(m.root) })
			}
			return m, nil
		default:
			switch key {
			case "q", "esc":
				return m, tea.Quit
			case "n":
				m.screen, m.status = naming, ""
				m.input.Reset()
				return m, m.input.Focus()
			case "r":
				m.busy, m.status = true, "Refreshing checkpoints…"
				return m, load(m.root, "Checkpoints refreshed")
			case "enter":
				if item, ok := m.list.SelectedItem().(checkpointItem); ok {
					m.screen, m.target, m.status = confirmRestore, item.meta.Name, ""
				}
				return m, nil
			case "x":
				if len(m.list.Items()) > 0 {
					m.screen, m.status = confirmDelete, ""
				}
				return m, nil
			}
		}
	}
	if m.busy {
		return m, nil
	}
	var cmd tea.Cmd
	if m.screen == naming {
		m.input, cmd = m.input.Update(msg)
	} else if m.screen == browsing {
		m.list, cmd = m.list.Update(msg)
	}
	return m, cmd
}

func (m model) View() string {
	header := "Checkpoint\n" + m.root + "\n\n"
	var body, help string
	switch m.screen {
	case naming:
		body = "Create checkpoint\n\n" + m.input.View()
		help = "enter: create • esc: cancel"
	case confirmRestore:
		body = fmt.Sprintf("Restore checkpoint '%s'?\nThis will overwrite files and remove files absent from the checkpoint.", m.target)
		help = "y: restore • n/enter/esc: cancel"
	case confirmDelete:
		body = "Delete all checkpoints?\nThis cannot be undone."
		help = "y: delete all • n/enter/esc: cancel"
	default:
		if len(m.list.Items()) == 0 {
			body = "No checkpoints found. Press n to create one."
		} else {
			body = m.list.View()
		}
		help = "↑/↓ or j/k: select • enter: restore • n: new • x: delete all • r: refresh • q: quit"
	}
	if m.busy {
		help = "Please wait…"
	}
	return lipgloss.NewStyle().MaxWidth(m.width).MaxHeight(m.height).Render(header + body + "\n\n" + m.status + "\n\n" + help)
}
