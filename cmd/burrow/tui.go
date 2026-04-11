package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xenomorphingtv/burrow/internal/config"
	"github.com/xenomorphingtv/burrow/internal/daemon"
	"github.com/xenomorphingtv/burrow/internal/runner"
	"github.com/xenomorphingtv/burrow/internal/store"
	"github.com/xenomorphingtv/burrow/internal/tui"
)

func runTUI() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	cfgDir := config.DefaultConfigDir()
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	// When the daemon is running it holds the bbolt lock. Use the socket-based
	// RemoteStore so both can coexist. When the daemon is not running, open
	// bbolt directly and run startup pruning.
	var st store.Storer
	var localSt *store.Store
	if _, running := daemon.IsRunning(); running {
		st = daemon.NewRemoteStore()
	} else {
		var err error
		localSt, err = store.Open(cfgDir)
		if err != nil {
			return fmt.Errorf("open store: %w", err)
		}
		defer localSt.Close()
		st = localSt

		if cfg.Settings.MaxLogRun > 0 {
			if err := localSt.Prune(cfg.Settings.MaxLogRun); err != nil {
				fmt.Fprintf(os.Stderr, "warning: prune history: %v\n", err)
			}
		}
		if cfg.Settings.MaxLogAge > 0 {
			if err := localSt.PruneByAge(cfg.Settings.MaxLogAge); err != nil {
				fmt.Fprintf(os.Stderr, "warning: prune history by age: %v\n", err)
			}
			if err := store.PruneLogFiles(cfg.Settings.LogDir, cfg.Settings.MaxLogAge); err != nil {
				fmt.Fprintf(os.Stderr, "warning: prune log files: %v\n", err)
			}
		}
	}

	sched := runner.NewScheduler()

	maxParallel := cfg.Settings.MaxParallel
	if maxParallel <= 0 {
		maxParallel = 4
	}
	pool := runner.NewPool(maxParallel)

	model := tui.New(cfg, st, sched, pool)

	p := tea.NewProgram(model, tea.WithAltScreen())

	if err := sched.Register(cfg, func(taskName, trigger string) {
		p.Send(tui.ScheduledRunMsg{TaskName: taskName, Trigger: trigger})
	}); err != nil {
		fmt.Fprintf(os.Stderr, "warning: scheduler registration: %v\n", err)
	}

	if disabled, err := st.LoadDisabledSchedules(); err == nil {
		for name := range disabled {
			if disabled[name] {
				sched.Disable(name)
			}
		}
	}

	sched.Start()
	defer sched.Stop()

	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}
