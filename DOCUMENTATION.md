# Burrow — Full Documentation

**Version:** 0.1.5

Burrow is a personal task catalogue, runner, and scheduler with an interactive terminal UI. Define named scripts in a TOML file, run them with a keystroke, stream their output live, chain them into pipelines, schedule them with cron, and browse the full run history — all without leaving the terminal.

---

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Config Files](#config-files)
- [Config Schema](#config-schema)
  - [Settings](#settings)
  - [Tasks](#tasks)
  - [Task Inputs](#task-inputs)
  - [Schedules](#schedules)
- [CLI Commands](#cli-commands)
- [TUI Interface](#tui-interface)
  - [Tabs](#tabs)
  - [Keybindings](#keybindings)
  - [Filter Mode](#filter-mode)
  - [Add Task Form](#add-task-form)
  - [Input Prompts](#input-prompts)
  - [Log Colorization](#log-colorization)
- [Themes](#themes)
  - [Built-in Themes](#built-in-themes)
  - [Custom Theme Files](#custom-theme-files)
- [Daemon](#daemon)
- [Desktop Notifications](#desktop-notifications)
- [Terminal Auto-Detection](#terminal-auto-detection)
- [Environment Variables](#environment-variables)
- [Special Files](#special-files)

---

## Installation

Requires Go 1.22+.

**Via `go install`:**
```bash
go install github.com/XenomorphingTV/burrow/cmd/burrow@latest
```

**Build from source:**
```bash
git clone https://github.com/xenomorphingtv/burrow
cd burrow
make install   # build and install to $GOPATH/bin
# or
make build     # build binary in current directory
```

---

## Quick Start

```bash
burrow init   # scaffold ~/.config/burrow/tasks.toml and open it in $EDITOR
burrow        # launch the TUI
```

See `burrow.example.toml` in the repo for a full tour of the config format.

---

## Config Files

Burrow merges two files on startup:

| File | Purpose |
|------|---------|
| `~/.config/burrow/tasks.toml` | Global personal catalogue — always loaded |
| `./burrow.toml` | Repo-local catalogue — loaded when present in the working directory |

**Merge rules:** Local (`burrow.toml`) keys win over the global file on a per-field basis. For tasks that appear in both files, `env` vars and `tags` are merged (unioned); all other fields from the local file override the global. Schedules are merged by name; local wins.

---

## Config Schema

### Settings

Defined under `[settings]`:

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `max_parallel` | int | `4` | Maximum number of concurrently running tasks |
| `log_dir` | string | `~/.config/burrow/logs` | Directory where per-run log files are written |
| `max_log_run` | int | `0` (unlimited) | Prune history after N runs per task |
| `max_log_age` | int | `0` (unlimited) | Prune history and log files older than N days |
| `notify` | `[]string` | `[]` | Default notification events: `"success"`, `"failure"`, or `"always"` |
| `terminal` | string | (auto-detected) | Terminal emulator used for `external = true` tasks; overrides `$TERMINAL` and auto-detection |
| `theme` | string | `"catppuccin-mocha"` | Built-in theme name or path to a `.toml` theme file |

Example:
```toml
[settings]
max_parallel = 4
log_dir      = "~/.config/burrow/logs"
max_log_run  = 50
max_log_age  = 30
notify       = ["failure"]
theme        = "nord"
```

### Tasks

Defined under `[tasks.<name>]`. Task names with dots (`[tasks.db.seed]`) are automatically grouped into collapsible namespace sections in the TUI sidebar.

| Key | Type | Description |
|-----|------|-------------|
| `cmd` | string | **Required.** Shell command to run. Executed via `/bin/sh -c`, so `$VAR`, pipes, and redirects all work. |
| `description` | string | Human-readable description shown in the TUI and `burrow list` |
| `cwd` | string | Working directory for the command. Supports `~` expansion. |
| `env` | `{string: string}` | Environment variables injected alongside the inherited environment |
| `tags` | `[]string` | Labels for filtering and grouping |
| `depends_on` | `[]string` | List of task names to run first. The full DAG is resolved; cycles are rejected. |
| `on_failure` | string | Task name or shell command to run when this task fails |
| `notify` | `[]string` | Per-task override of `settings.notify`. Set to `[]` to silence all notifications for this task. |
| `external` | bool | If `true`, launch the command in a separate terminal window instead of capturing output in the TUI log pane. Useful for TUIs and REPLs. |
| `timeout` | int | Kill the task after N seconds. `0` = no timeout. |
| `retries` | int | Number of retry attempts on failure (beyond the initial run). `0` = no retries. |
| `retry_delay` | int | Seconds to wait between retry attempts. `0` = immediate. |
| `watch` | `[]string` | Glob patterns (supports `**` for recursive matching). The TUI and daemon automatically re-run the task whenever a matched file changes. |
| `inputs` | `[]TaskInput` | Runtime prompts collected before the task runs (see [Task Inputs](#task-inputs)) |

Example:
```toml
[tasks.db-seed]
cmd         = "node scripts/seed.js"
description = "Seed the development database"
cwd         = "./backend"
env         = { NODE_ENV = "development" }
tags        = ["db", "dev"]
depends_on  = ["db-migrate"]
on_failure  = "notify-ops"
notify      = ["failure"]
timeout     = 120
retries     = 2
retry_delay = 5
watch       = ["src/**/*.js"]
```

### Task Inputs

Defined as an array under `[[tasks.<name>.inputs]]`. Each entry prompts the user for a value before the task runs. The collected value is injected as an environment variable.

| Key | Type | Description |
|-----|------|-------------|
| `name` | string | **Required.** Name of the environment variable to set |
| `prompt` | string | Prompt text shown to the user |
| `options` | `[]string` | If provided, presents a selection list. Omit for free-text entry. |

Example:
```toml
[[tasks.deploy.inputs]]
name    = "ENV"
prompt  = "Deploy to which environment?"
options = ["staging", "production"]

[[tasks.deploy.inputs]]
name   = "TAG"
prompt = "Docker image tag to deploy:"
```

Reference collected values in `cmd` as normal environment variables (`$ENV`, `$TAG`).

### Schedules

Defined under `[schedules.<name>]`:

| Key | Type | Description |
|-----|------|-------------|
| `task` | string | **Required.** Name of the task to run |
| `cron` | string | **Required.** Standard 5-field cron expression (`minute hour dom month dow`) |

Example:
```toml
[schedules.nightly-seed]
task = "db-seed"
cron = "0 2 * * *"

[schedules.weekly-cleanup]
task = "cleanup"
cron = "0 9 * * 1"
```

---

## CLI Commands

### `burrow`
Launches the interactive TUI. This is the default when no subcommand is given.

### `burrow run <task-name>`
Runs a named task headlessly, streaming stdout/stderr to the terminal. The full `depends_on` pipeline is resolved and executed in order. Exits with the task's own exit code. If the task defines `inputs`, burrow prompts for them on stdin before running.

```bash
burrow run db-seed
burrow run deploy | tee /tmp/deploy.log
```

### `burrow list`
Prints the task catalogue as formatted plain text (name, description, tags).

**Flags:**
| Flag | Description |
|------|-------------|
| `--json` | Emit a JSON array instead of plain text. Each object includes: `name`, `cmd`, `description`, `cwd`, `tags`, `depends_on`, `watch`, `env`, `timeout`, `retries`, `on_failure`, `external`. |
| `<file>` | (Positional, after `--json`) Write output to a file path instead of stdout. |

```bash
burrow list
burrow list | grep db
burrow list --json
burrow list --json tasks.json
```

### `burrow init`
Scaffolds `~/.config/burrow/tasks.toml` with a starter template. Does nothing if the file already exists. Opens the file in `$EDITOR` after creation.

### `burrow check`
Validates the merged config without starting the TUI. Reports:

- Missing `cmd` on any task
- Broken `depends_on` references (unknown task names)
- Dependency cycles
- `on_failure` values that look like task names but don't exist (warning)
- `cwd` paths that do not exist (warning)
- `inputs` entries missing a `name`
- Negative `timeout`, `retries`, or `retry_delay` values
- Missing or invalid cron expressions in `[schedules]`

Exits non-zero on errors. Suitable for use in CI.

```bash
burrow check
# ✓ 12 task(s), 3 schedule(s) — no errors
```

### `burrow daemon <subcommand>`
Manages the background scheduler daemon.

| Subcommand | Description |
|------------|-------------|
| `start` | Start the daemon. Uses the OS service manager if installed; otherwise spawns a detached process. |
| `stop` | Stop the running daemon via the service manager or SIGTERM. |
| `status` | Show daemon PID, uptime, last heartbeat, config load time, next scheduled run times (with countdowns), and active watch tasks. |
| `install` | Write the OS service file: systemd user service on Linux (`~/.config/systemd/user/burrow.service`), launchd user agent on macOS (`~/Library/LaunchAgents/com.xenodium.burrow.plist`). |
| `uninstall` | Remove the service file and stop the daemon first. |

```bash
burrow daemon install
burrow daemon start
burrow daemon status
burrow daemon stop
burrow daemon uninstall
```

### `burrow help`
Prints a usage summary including all subcommands, config schema, and TUI keybindings. Also triggered by `--help` or `-h`.

### `burrow version`
Prints the build-time version string (e.g., `burrow 0.1.5`). Also triggered by `--version` or `-v`.

---

## TUI Interface

### Tabs

The TUI has four tabs, cycled with `Tab`:

| Tab | Description |
|-----|-------------|
| **tasks** | Task catalogue with live log streaming |
| **schedule** | All cron schedules and watch tasks; enable/disable/edit inline |
| **history** | Per-run history with exit code, duration, trigger, and last 200 lines of output |
| **stats** | Aggregate per-task view: runs in the last 7 days, success rate, average duration, last run time |

### Keybindings

| Key | Action |
|-----|--------|
| `↑` / `k` | Navigate up |
| `↓` / `j` | Navigate down |
| `Tab` | Switch to next tab (tasks → schedule → history → stats → tasks) |
| `r` | Run selected task |
| `x` | Kill running task (or cancel active watcher) |
| `l` | Clear log view for the selected task |
| `a` | Add task (opens an inline form) |
| `e` | Open task's source config in `$EDITOR` at the task's line (tasks tab); edit cron expression inline (schedule tab) |
| `Enter` / `Space` | Toggle schedule enabled/disabled (schedule tab); collapse/expand namespace group (tasks tab) |
| `D` | Clear all task history (history tab; requires `y` confirmation) |
| `/` | Enter fuzzy filter mode |
| `Esc` or `Enter` | Exit filter mode |
| `?` | Toggle help overlay |
| `pgup` / `ctrl+u` | Scroll log up (tasks tab); scroll view up (schedule/stats tabs) |
| `pgdn` / `ctrl+d` | Scroll log down (tasks tab); scroll view down (schedule/stats tabs) |
| `q` / `ctrl+c` | Quit |

### Filter Mode

Press `/` to enter filter mode. Type to fuzzy-filter by task name or tag substring. The filter resets when switching tabs. Works on the tasks, schedule, and history tabs. Press `Esc` or `Enter` to exit.

### Add Task Form

Press `a` to open the inline add-task form. Fields:

- **Name** — no spaces allowed
- **Cmd** — the shell command
- **Description** — human-readable label
- **Tags** — comma-separated
- **Cwd** — working directory

Navigate fields with `Tab` / `Shift+Tab`. Confirm on the last field with `Enter`. Cancel with `Esc`. The new task is saved to the global `~/.config/burrow/tasks.toml`.

### Input Prompts

When running a task that defines `inputs` (via `r`):

- **Free-text inputs:** type and press `Enter`
- **Option-select inputs:** navigate with `↑`/`k` and `↓`/`j`, confirm with `Enter`
- Cancel the whole run with `Esc`

### Log Colorization

Lines in the TUI log pane are colorized automatically based on their prefix:

| Prefix | Color |
|--------|-------|
| `error`, `err`, `fail`, `fatal` | Red |
| `warn` | Yellow |
| `info`, `==>` | Blue |
| `$` (echoed command) | Dimmed |
| All other lines | Default text color |

---

## Themes

Set `settings.theme` in the config to one of the built-in names or a path to a `.toml` file.

### Built-in Themes

| Name | Description |
|------|-------------|
| `catppuccin-mocha` | **Default.** Catppuccin Mocha dark palette |
| `nord` | Nord palette |
| `dracula` | Dracula palette |
| `gruvbox` | Gruvbox dark palette |

### Custom Theme Files

A custom theme is a TOML file containing any subset of the following keys (all values are hex color strings). Any omitted key falls back to the Catppuccin Mocha default — partial theme files are valid.

| Key | Used for |
|-----|----------|
| `bg` | Main background |
| `bg_header` | Header and status bar background |
| `border` | Border and separator lines |
| `selected` | Selected row highlight background |
| `text` | Primary text |
| `dim` | Dimmed / secondary text |
| `purple` | Accent color — tabs, key hints, log task names |
| `blue` | Task names, info log lines |
| `green` | Success status, ok log lines |
| `red` | Failure/error status, error log lines |
| `yellow` | Warnings |

Example theme file (`~/.config/burrow/mytheme.toml`):
```toml
bg       = "#1e1e2e"
text     = "#cdd6f4"
green    = "#a6e3a1"
red      = "#f38ba8"
yellow   = "#f9e2af"
blue     = "#89b4fa"
purple   = "#cba6f7"
```

Reference in config:
```toml
[settings]
theme = "~/.config/burrow/mytheme.toml"
```

---

## Daemon

The daemon runs scheduled tasks and file watchers in the background, independent of the TUI.

**Behavior:**
- Runs all configured cron schedules
- Watches files for tasks with `watch` patterns and re-triggers them on changes
- Hot-reloads config when `~/.config/burrow/tasks.toml` changes (polls every 5 seconds)
- Also reloads on `SIGHUP`
- Communicates with the TUI via a Unix socket (`~/.config/burrow/burrow.sock`) to coordinate history writes

**Setup:**
```bash
burrow daemon install   # write the systemd/launchd service file
burrow daemon start     # enable and start the service
burrow daemon status    # inspect uptime, next runs, active watchers
```

Daemon logs are written to `~/.config/burrow/logs/daemon.log` when running without a service manager.

---

## Desktop Notifications

Notifications require `notify-send` on Linux or `osascript` on macOS. Errors are non-fatal (logged to stderr).

Configure globally in `[settings]`:
```toml
[settings]
notify = ["failure"]   # "success", "failure", or "always"
```

Override per task:
```toml
[tasks.my-task]
notify = ["always"]   # override global
# notify = []         # silence all notifications for this task
```

---

## Terminal Auto-Detection

Used when `settings.terminal` is not set and `$TERMINAL` is unset, for tasks with `external = true`. Burrow scans `PATH` for these terminals in order:

`kitty`, `alacritty`, `wezterm`, `ghostty`, `foot`, `gnome-terminal`, `konsole`, `xfce4-terminal`, `xterm`

The first one found is used. If none are found, an error is shown.

---

## Environment Variables

| Variable | Description |
|----------|-------------|
| `EDITOR` | Editor opened by `burrow init` and the `e` keybinding. Line-number jumping is supported for `vi`, `vim`, `nvim`, `nano`, `emacs`, `micro`, `kak` (`+N file`) and `hx`, `helix`, `code` (`file:N`). Defaults to `vi` if unset. |
| `TERMINAL` | Fallback terminal emulator for `external = true` tasks. Checked after `settings.terminal` and before auto-detection. |
| `XDG_RUNTIME_DIR` | Used by the notify subsystem to locate the D-Bus session socket when `DBUS_SESSION_BUS_ADDRESS` is unset. Falls back to `/run/user/<UID>`. |
| `DBUS_SESSION_BUS_ADDRESS` | Used directly by `notify-send` on Linux; auto-derived from `XDG_RUNTIME_DIR` when absent, so notifications work from daemon mode. |

---

## Special Files

| Path | Description |
|------|-------------|
| `~/.config/burrow/tasks.toml` | Global task catalogue (created by `burrow init`) |
| `./burrow.toml` | Repo-local task catalogue (loaded when present in the working directory) |
| `~/.config/burrow/history.db` | Run history stored as a bbolt embedded key/value database. Do not edit directly. |
| `~/.config/burrow/logs/` | Per-run log files named `<task>-<timestamp>.log` |
| `~/.config/burrow/logs/daemon.log` | Daemon stdout/stderr when running without a service manager |
| `~/.config/burrow/burrow.sock` | Unix domain socket for TUI–daemon IPC |
| `~/.config/burrow/burrow.pid` | Daemon PID file |
| `~/.config/systemd/user/burrow.service` | systemd user service file (Linux, written by `burrow daemon install`) |
| `~/Library/LaunchAgents/com.xenodium.burrow.plist` | launchd user agent file (macOS, written by `burrow daemon install`) |
