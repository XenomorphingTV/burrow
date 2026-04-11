# burrow

[![CI](https://github.com/xenomorphingtv/burrow/actions/workflows/ci.yml/badge.svg)](https://github.com/xenomorphingtv/burrow/actions/workflows/ci.yml)
[![Latest Release](https://img.shields.io/github/v/release/xenomorphingtv/burrow)](https://github.com/xenomorphingtv/burrow/releases/latest)
[![Go 1.22+](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Platform: Linux](https://img.shields.io/badge/Platform-Linux-lightgrey?logo=linux&logoColor=white)]()
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A personal task catalogue, runner, and scheduler with a terminal UI.

Define named scripts in a TOML file, run them with a keystroke, stream their
output live, chain them into pipelines, schedule them with cron, and browse
the full run history, all without leaving the terminal.

## Why

Most developers accumulate a pile of scripts they run regularly - database seeds,
deployment shortcuts, cleanup jobs, one-off utilities. `make` and `just` are
project-scoped and file-centric. `cron` has no UI and no visibility into whether
jobs succeed or fail. `systemd` timers are powerful but heavyweight for personal
use.

Burrow fills that gap. It lives in your home directory, gives every script a name
and a description, logs every run automatically, and lets you schedule things
without touching a crontab or writing a service file.

<img width="941" height="508" alt="screenshot-2026-04-11_19-39-18" src="https://github.com/user-attachments/assets/1e8197fc-c265-41c2-b1b4-31bb0618b46b">

<img width="941" height="508" alt="screenshot-2026-04-11_19-42-18" src="https://github.com/user-attachments/assets/fc373f82-6878-4451-a765-881e3712834e" />

<img width="941" height="508" alt="screenshot-2026-04-11_19-40-58" src="https://github.com/user-attachments/assets/cb190985-651a-4ea3-a82c-76bf43003f97" />

<img width="941" height="508" alt="screenshot-2026-04-11_19-41-22" src="https://github.com/user-attachments/assets/5123382e-011c-4f8c-b876-2775ed9da611" />

## Install

Requires Go 1.22+.

```bash
git clone https://github.com/xenomorphingtv/burrow
cd burrow
make install
```

Or just build the binary:

```bash
make build
```


## Quick start

```bash
burrow init   # scaffold ~/.config/burrow/tasks.toml and open it in $EDITOR
burrow        # launch the TUI
```

See `burrow.example.toml` in this repo for a full tour of the config format.


## CLI

```bash
burrow                  # launch the TUI
burrow run <task>       # run a task headlessly, stream output to stdout
burrow list             # print the task catalogue as plain text
burrow init             # scaffold the global config
burrow daemon start     # start the background scheduler daemon
burrow daemon status    # show daemon uptime and upcoming scheduled runs
burrow help             # full usage reference
burrow version          # print version
```


## TUI keys

| Key | Action |
|---|---|
| `j` / `k` | Navigate |
| `r` | Run selected task |
| `x` | Kill running task |
| `l` | Clear log view |
| `/` | Filter by name or tag |
| `a` | Add a new task |
| `tab` | Switch tab (tasks / schedule / history / stats) |
| `?` | Help overlay |
| `q` | Quit |


## Config

Burrow merges two files on startup:

- `~/.config/burrow/tasks.toml` - your global personal catalogue
- `./burrow.toml` - repo-local tasks, merged on top when present

```toml
[settings]
max_parallel = 4
log_dir      = "~/.config/burrow/logs"

[tasks.db-seed]
cmd         = "node scripts/seed.js"
description = "Seed the development database"
cwd         = "./backend"
env         = { NODE_ENV = "development" }
tags        = ["db"]
depends_on  = ["db-migrate"]   # pipeline
on_failure  = "notify-ops"     # runs if this task fails

[schedules.nightly]
task = "db-seed"
cron = "0 2 * * *"
```


## Acknowledgements

Burrow is built on the shoulders of some excellent open source work:

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) <- the TUI framework
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) <- terminal styling
- [Bubbles](https://github.com/charmbracelet/bubbles) <- UI components (viewport, text input, spinner)
- [BurntSushi/toml](https://github.com/BurntSushi/toml) <- TOML config parsing
- [robfig/cron](https://github.com/robfig/cron) <- cron scheduling engine
- [bbolt](https://github.com/etcd-io/bbolt) <- embedded key/value store for run history

Thank you to the maintainers of all of these projects.


## License

MIT
