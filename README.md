# Checkpoint

Lightweight, local checkpoints for a git repo. It snapshots tracked + untracked files (excluding ignored files) into `.checkpoints/`, and lets you (or your AI Agent) restore later.

## Install

#### Prerequisites
1. [Go](https://go.dev/) >= 1.25.6 in $PATH

```bash
go install github.com/LukeTarr/checkpoint@latest
```

## Usage

Run commands from inside a git repo.

```bash
#create checkpoint
checkpoint push [name]
#restore from checkpoint
checkpoint pop [name]
#show current repo checkpoints
checkpoint list
#remove all current repo checkpoints
checkpoint nuke
#outputs shell script for completions (pipe into shell file in completions directory to get tab auto complete when invoking)
checkpoint completion
#open the optional interactive terminal interface
checkpoint tui
```

### Interactive interface

Run `checkpoint tui` in a terminal inside your repository to open the
[Bubble Tea](https://github.com/charmbracelet/bubbletea) interface. Existing CLI
commands continue to work as before, including for scripts.

The interface lists checkpoints newest first with creation time, file count, and
line count. Use the arrow keys or `j`/`k` to select a checkpoint; long lists scroll
and support page up/down.

| Key | Action |
| --- | --- |
| `n` | Create a checkpoint; enter a name or leave it blank for a timestamped name |
| `Enter` | Restore the selected checkpoint after confirmation |
| `x` | Delete all checkpoints after confirmation |
| `r` | Refresh the list |
| `q` / `Esc` / `Ctrl+C` | Quit from the list |

In name entry, `Enter` creates and `Esc` cancels. Existing checkpoint names are
rejected; use the CLI's `push --force` to overwrite one. Confirmation prompts
require `y`; `n`, `Enter`, or `Esc` cancels. Restore overwrites current files and
removes non-ignored files that are absent from the checkpoint. File operations
run in the background, and new actions and quitting wait until they finish.

## Building

#### Prerequisites
1. [git](https://git-scm.com/)
2. [Go](https://go.dev/) >= 1.25.6 in $PATH

```bash
git clone https://github.com/LukeTarr/checkpoint.git
cd checkpoint
#will output a ./checkpoint executable file
go build
```

## Notes

- Requires a `.git` directory.
- Ignores files matched by git ignore rules.
- `.checkpoints` is never included in snapshots.
