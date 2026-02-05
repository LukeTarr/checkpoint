# Checkpoint

Lightweight, local checkpoints for a git repo. It snapshots tracked + untracked files (excluding ignored files) into `.checkpoints`, and lets you restore later.

## Install

```bash
go build -o checkpoint
```

## Usage

Run commands from inside a git repo.

```bash
./checkpoint push [name]
./checkpoint pop [name]
./checkpoint list
./checkpoint nuke
```

### Commands

- `push [name]` create a checkpoint. Use `--force` to overwrite an existing name.
- `pop [name]` restore a checkpoint (defaults to latest). Prompts before overwriting files.
- `list` show checkpoints with timestamps and quick stats.
- `nuke` delete all checkpoints (with confirmation).

## Notes

- Requires a `.git` directory.
- Ignores files matched by git ignore rules.
- `.checkpoints` is never included in snapshots.
