package main

import (
	"fmt"

	"github.com/xenomorphingtv/burrow/internal/version"
)

func printVersion() {
	fmt.Printf("burrow %s\n", version.Version)
}

func printHelp() {
	fmt.Printf(`burrow %s — personal task catalogue, runner, and scheduler

USAGE
    burrow                       launch the TUI
    burrow run <task>            run a task headlessly, stream output to stdout
    burrow list                  list all configured tasks
    burrow init                  scaffold ~/.config/burrow/tasks.toml
    burrow daemon <subcommand>   manage the background daemon
    burrow version               print version and exit
    burrow help                  show this message

DAEMON SUBCOMMANDS
    start      enable and start the burrow daemon
    stop       stop the burrow daemon
    status     show daemon status and next scheduled runs
    install    write the systemd/launchd service file
    uninstall  remove the service file

CONFIG FILES
    Global:  ~/.config/burrow/tasks.toml   (personal catalogue)
    Local:   ./burrow.toml                  (repo-scoped, merged on top)

CONFIG SCHEMA
    [settings]
    max_parallel = 4
    log_dir      = "~/.config/burrow/logs"

    [tasks.my-task]
    cmd         = "bash ~/scripts/foo.sh"
    description = "What this task does"
    cwd         = "~"
    env         = { MY_VAR = "value" }
    tags        = ["personal", "setup"]
    depends_on  = ["other-task"]   # pipeline — run other-task first

    [schedules.nightly]
    task = "my-task"
    cron = "0 2 * * *"            # standard 5-field cron expression

TUI KEYBINDINGS
    ↑/k              navigate up
    ↓/j              navigate down
    r                run selected task
    x                kill running task
    l                clear log view
    tab              switch pane tab (tasks / schedule / history)
    /                fuzzy filter by name or tag
    a                add task
    e                edit schedule
    enter/space      toggle schedule enabled/disabled
    D                clear task history
    ?                toggle help overlay
    q / ctrl+c       quit

FILES
    ~/.config/burrow/tasks.toml   global task catalogue
    ~/.config/burrow/history.db   run history (bbolt)
    ~/.config/burrow/logs/        per-run log files

For the full manual, run: man burrow
`, version.Version)
}
