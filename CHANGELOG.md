# Changelog

All notable changes to this project will be documented here.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

## [0.1.0] - 2026-04-11

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
