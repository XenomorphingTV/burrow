# Changelog

All notable changes to this project will be documented here.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Released]

## [0.1.5] - 2026-04-24

### Added

* TUI themes — set `settings.theme` to a built-in name (`catppuccin-mocha`, `nord`, `dracula`, `gruvbox`) or a path to a custom TOML theme file; partial theme files are valid and fall back to Catppuccin Mocha for any unset fields
* File watch mode — define `watch` on any task as a list of glob patterns (supports `**` for recursive matching); the TUI and daemon re-run the task automatically whenever a matched file changes
* `burrow list --json` — print the task catalogue as a JSON array; pass an optional file path (`burrow list --json out.json`) to write directly to a file instead of stdout
* Daemon watch status — `burrow daemon status` now lists active watch tasks alongside scheduled runs

## [0.1.4] - 2026-04-14

### Added

* Task retries — define `retries` and `retry_delay` on any task; burrow re-runs the command on failure up to the configured number of attempts, emitting `[retry N/M]` log lines between attempts; output from all attempts is accumulated into a single log file and history record
* Filter on schedule and history tabs — `/` now filters all tabs except stats; schedule tab matches on schedule name or task name; filter resets on tab switch
* `burrow check` — validates `burrow.toml` without starting the TUI; reports missing `cmd`, broken `depends_on` references, dependency cycles, unknown `on_failure` task names, invalid cron expressions, and negative `timeout`/`retries`/`retry_delay` values; exits non-zero on errors for use in CI

## [0.1.3] - 2026-04-13

### Added

* Interactive task inputs — define `inputs` on any task to have burrow prompt for values at run time before execution; supports free-text and option-select prompts; values injected as environment variables
* Task status badges now reflect run history on startup — dots show the last known result (ok/failed) rather than always starting idle
* Keybinding documentation — all binds now documented in the help overlay (`?`), status bar (context-sensitive per tab), inline hint lines, and README

### Fixed

* Commands now run through `/bin/sh -c`, enabling `$VAR` expansion, pipes, redirects, and other shell features expected from a task runner
* Input prompts parser — `[[task.inputs]]` array-of-tables TOML syntax now parsed correctly (BurntSushi/toml decodes these as `[]map[string]interface{}`, not `[]interface{}`)

## [0.1.2] - 2026-04-12

### Added

* Edit task — press `e` on any selected task to open its source file at the correct line in `$EDITOR` directly from the TUI

### Fixed

* `depends_on` pipeline execution — dependency ordering in `burrow.toml` now resolves correctly when running tasks from the TUI


## [0.1.1] - 2026-04-11

### Added
- TUI with four tabs: tasks, schedule, history, and stats
- Live log streaming with automatic colourisation (ok, error, warn, info)
- Task catalogue loaded from `~/.config/burrow/tasks.toml` and `./burrow.toml`, merged on startup
- Task namespacing - dot-notation groups tasks into collapsible sidebar sections (`db.seed`, `db.migrate`)
- Pipeline execution via `depends_on` — full DAG resolved with DFS before any process starts, cycles rejected immediately
- `on_failure` hook - runs a named task or inline shell command when a task fails
- Cron scheduling via `robfig/cron` — schedules registered on TUI launch and daemon start
- Schedule tab - enable/disable schedules at runtime, edit cron expressions inline with live preview
- History tab - per-run records stored in bbolt, showing exit code, duration, trigger, and last 200 lines of output
- Stats tab - per-task aggregate view: runs in the last 7 days, success rate, average duration, last run time
- Run history pruning by count (`max_log_run`) and age (`max_log_age`)
- Desktop notifications via `notify-send` (Linux) and `osascript` (macOS), configurable globally and per task
- `external = true` task flag - launches command in a separate terminal emulator instead of capturing output
- Auto-detection of terminal emulator (`kitty`, `alacritty`, `wezterm`, `ghostty`, `foot`, and others)
- Background daemon - runs scheduled tasks without the TUI open
- Daemon IPC over a Unix socket - TUI routes history and schedule operations through the daemon when it is running
- Daemon config hot-reload - watches `tasks.toml` for changes every 5 seconds, also reloads on `SIGHUP`
- `systemd` user service support on Linux (`burrow daemon install/start/stop`)
- `launchd` user agent support on macOS
- `burrow run <task>` - headless execution, streams output to stdout, exits with the task's own exit code
- `burrow list` - prints task catalogue as plain text for scripting
- `burrow init` - scaffolds `~/.config/burrow/tasks.toml` and opens it in `$EDITOR`
- `burrow help` - full usage reference including config schema and TUI keybindings
- `burrow version` - prints the build-time version string
- Man page (`burrow.1`)
- `burrow.example.toml` - annotated example config covering every feature
- Build-time version injection via `-ldflags`
- `Makefile` with `build`, `test`, `vet`, `check`, `install`, `install-man`, and `release` targets
- GitHub Actions CI - runs `go vet`, `go test`, and `go build` on Linux
