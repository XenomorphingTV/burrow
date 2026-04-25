package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/XenomorphingTV/burrow/internal/config"
	"github.com/XenomorphingTV/burrow/internal/daemon"
	"github.com/XenomorphingTV/burrow/internal/runner"
	"github.com/XenomorphingTV/burrow/internal/store"
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

		// Collect runtime inputs via stdin if the task defines them.
		if len(t.Inputs) > 0 {
			fmt.Printf("==> Inputs for task: %s\n", name)
			extras, err := collectInputsHeadless(t.Inputs)
			if err != nil {
				return fmt.Errorf("collect inputs for %s: %w", name, err)
			}
			merged := make(map[string]string, len(t.Env)+len(extras))
			for k, v := range t.Env {
				merged[k] = v
			}
			for k, v := range extras {
				merged[k] = v
			}
			t.Env = merged
		}

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
				if t.OnSuccess != "" {
					runOnSuccess(t.OnSuccess, name, cfg, st)
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

// collectInputsHeadless prompts for each task input on stdin and returns the collected values.
func collectInputsHeadless(inputs []config.TaskInput) (map[string]string, error) {
	values := make(map[string]string, len(inputs))
	reader := bufio.NewReader(os.Stdin)
	for _, inp := range inputs {
		if len(inp.Options) > 0 {
			fmt.Printf("  %s\n", inp.Prompt)
			for i, opt := range inp.Options {
				fmt.Printf("    [%d] %s\n", i+1, opt)
			}
			fmt.Printf("  choice (1-%d): ", len(inp.Options))
			line, err := reader.ReadString('\n')
			if err != nil {
				return nil, fmt.Errorf("read input for %s: %w", inp.Name, err)
			}
			line = strings.TrimSpace(line)
			chosen := inp.Options[0] // default to first option
			for i, opt := range inp.Options {
				if line == opt || line == fmt.Sprintf("%d", i+1) {
					chosen = opt
					break
				}
			}
			values[inp.Name] = chosen
		} else {
			fmt.Printf("  %s: ", inp.Prompt)
			line, err := reader.ReadString('\n')
			if err != nil {
				return nil, fmt.Errorf("read input for %s: %w", inp.Name, err)
			}
			values[inp.Name] = strings.TrimSpace(line)
		}
	}
	return values, nil
}

// runOnSuccess executes the on_success value for a succeeded task.
// If value is a known task name, that task is run headlessly.
// Otherwise it is treated as a shell command.
func runOnSuccess(onSuccess, parentName string, cfg *config.Config, st store.Storer) {
	fmt.Fprintf(os.Stderr, "==> Running on_success: %s\n", onSuccess)
	if task, ok := cfg.Tasks[onSuccess]; ok {
		exec := runner.NewExecutor(onSuccess, task, "on_success", cfg.Settings.LogDir, st, cfg.Settings.Notify)
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
		task := config.Task{Cmd: onSuccess}
		exec := runner.NewExecutor(parentName+".on_success", task, "on_success", cfg.Settings.LogDir, st, nil)
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
