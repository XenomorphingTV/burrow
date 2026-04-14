package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/XenomorphingTV/burrow/internal/config"
	"github.com/XenomorphingTV/burrow/internal/runner"
	"github.com/robfig/cron/v3"
)

func runCheck() error {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "✗ parse error: %v\n", err)
		os.Exit(1)
	}

	var errs []string
	var warns []string

	addErr := func(format string, a ...any) {
		errs = append(errs, fmt.Sprintf(format, a...))
	}
	addWarn := func(format string, a ...any) {
		warns = append(warns, fmt.Sprintf(format, a...))
	}

	// --- tasks ---
	var taskNames []string
	for n := range cfg.Tasks {
		taskNames = append(taskNames, n)
	}
	sort.Strings(taskNames)

	for _, name := range taskNames {
		t := cfg.Tasks[name]

		if t.Cmd == "" {
			addErr("task %q: missing cmd", name)
		}
		if t.Timeout < 0 {
			addErr("task %q: timeout must be >= 0", name)
		}
		if t.Retries < 0 {
			addErr("task %q: retries must be >= 0", name)
		}
		if t.RetryDelay < 0 {
			addErr("task %q: retry_delay must be >= 0", name)
		}

		// depends_on: check references exist and no cycles
		for _, dep := range t.DependsOn {
			if _, ok := cfg.Tasks[dep]; !ok {
				addErr("task %q: depends_on references unknown task %q", name, dep)
			}
		}
		if len(t.DependsOn) > 0 {
			if _, err := runner.Resolve(name, cfg.Tasks); err != nil {
				addErr("task %q: %v", name, err)
			}
		}

		// on_failure: warn if it looks like a task name but doesn't exist
		if t.OnFailure != "" {
			// Only flag it if it contains no shell metacharacters — a plain word is
			// almost certainly intended as a task reference.
			looksLikeName := true
			for _, c := range t.OnFailure {
				if c == ' ' || c == '|' || c == '&' || c == ';' || c == '$' {
					looksLikeName = false
					break
				}
			}
			if looksLikeName {
				if _, ok := cfg.Tasks[t.OnFailure]; !ok {
					addWarn("task %q: on_failure %q is not a known task (treated as shell command)", name, t.OnFailure)
				}
			}
		}

		// cwd: warn if the path doesn't exist
		if t.Cwd != "" {
			expanded := t.Cwd
			if len(expanded) >= 2 && expanded[:2] == "~/" {
				if home, err := os.UserHomeDir(); err == nil {
					expanded = home + expanded[1:]
				}
			}
			if _, err := os.Stat(expanded); os.IsNotExist(err) {
				addWarn("task %q: cwd %q does not exist", name, t.Cwd)
			}
		}

		// inputs: each entry must have a name
		for i, inp := range t.Inputs {
			if inp.Name == "" {
				addErr("task %q: inputs[%d] is missing a name", name, i)
			}
		}
	}

	// --- schedules ---
	cronParser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	var schedNames []string
	for n := range cfg.Schedules {
		schedNames = append(schedNames, n)
	}
	sort.Strings(schedNames)

	for _, name := range schedNames {
		s := cfg.Schedules[name]

		if s.Task == "" {
			addErr("schedule %q: missing task", name)
		} else if _, ok := cfg.Tasks[s.Task]; !ok {
			addErr("schedule %q: references unknown task %q", name, s.Task)
		}

		if s.Cron == "" {
			addErr("schedule %q: missing cron expression", name)
		} else if _, err := cronParser.Parse(s.Cron); err != nil {
			addErr("schedule %q: invalid cron expression %q: %v", name, s.Cron, err)
		}
	}

	// --- report ---
	for _, w := range warns {
		fmt.Fprintf(os.Stderr, "  warn: %s\n", w)
	}
	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "  error: %s\n", e)
		}
		fmt.Fprintf(os.Stderr, "\n✗ %d error(s)", len(errs))
		if len(warns) > 0 {
			fmt.Fprintf(os.Stderr, ", %d warning(s)", len(warns))
		}
		fmt.Fprintln(os.Stderr)
		os.Exit(1)
	}

	tasks := len(cfg.Tasks)
	scheds := len(cfg.Schedules)
	fmt.Printf("✓ %d task(s), %d schedule(s) — no errors", tasks, scheds)
	if len(warns) > 0 {
		fmt.Printf(", %d warning(s)", len(warns))
	}
	fmt.Println()
	return nil
}
