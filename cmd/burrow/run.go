package main

import (
	"fmt"
	"os"

	"github.com/xenomorphingtv/burrow/internal/config"
	"github.com/xenomorphingtv/burrow/internal/daemon"
	"github.com/xenomorphingtv/burrow/internal/runner"
	"github.com/xenomorphingtv/burrow/internal/store"
)

func runHeadless(taskName string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if _, ok := cfg.Tasks[taskName]; !ok {
		return fmt.Errorf("task %q not found", taskName)
	}

	ordered, err := runner.Resolve(taskName, cfg.Tasks)
	if err != nil {
		return fmt.Errorf("resolve pipeline: %w", err)
	}

	cfgDir := config.DefaultConfigDir()
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	var st store.Storer
	if _, running := daemon.IsRunning(); running {
		st = daemon.NewRemoteStore()
	} else {
		localSt, err := store.Open(cfgDir)
		if err != nil {
			return fmt.Errorf("open store: %w", err)
		}
		defer localSt.Close()
		st = localSt
	}

	exitCode := 0
	for i, name := range ordered {
		t := cfg.Tasks[name]
		trigger := "manual"
		if i < len(ordered)-1 {
			trigger = "pipeline"
		}

		fmt.Printf("==> Running task: %s\n", name)
		if t.Cmd != "" {
			fmt.Printf("    %s\n", t.Cmd)
		}

		exec := runner.NewExecutor(name, t, trigger, cfg.Settings.LogDir, st, cfg.Settings.Notify)
		exec.Start()

		for line := range exec.LogCh() {
			if line.Done {
				exitCode = line.ExitCode
				if exitCode != 0 {
					fmt.Fprintf(os.Stderr, "==> Task %s failed with exit code %d\n", name, exitCode)
					if t.OnFailure != "" {
						runOnFailure(t.OnFailure, name, cfg, st)
					}
					os.Exit(exitCode)
				}
				break
			}
			if line.IsErr {
				fmt.Fprintln(os.Stderr, line.Text)
			} else {
				fmt.Println(line.Text)
			}
		}
	}

	os.Exit(exitCode)
	return nil
}

// runOnFailure executes the on_failure value for a failed task.
// If value is a known task name, that task is run headlessly.
// Otherwise it is treated as a shell command.
func runOnFailure(onFailure, parentName string, cfg *config.Config, st store.Storer) {
	fmt.Fprintf(os.Stderr, "==> Running on_failure: %s\n", onFailure)
	if task, ok := cfg.Tasks[onFailure]; ok {
		exec := runner.NewExecutor(onFailure, task, "on_failure", cfg.Settings.LogDir, st, cfg.Settings.Notify)
		exec.Start()
		for line := range exec.LogCh() {
			if line.Done {
				break
			}
			if line.IsErr {
				fmt.Fprintln(os.Stderr, line.Text)
			} else {
				fmt.Println(line.Text)
			}
		}
	} else {
		task := config.Task{Cmd: onFailure}
		exec := runner.NewExecutor(parentName+".on_failure", task, "on_failure", cfg.Settings.LogDir, st, nil)
		exec.Start()
		for line := range exec.LogCh() {
			if line.Done {
				break
			}
			if line.IsErr {
				fmt.Fprintln(os.Stderr, line.Text)
			} else {
				fmt.Println(line.Text)
			}
		}
	}
}
