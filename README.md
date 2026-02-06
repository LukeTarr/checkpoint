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
```

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