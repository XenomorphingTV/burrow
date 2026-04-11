package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/xenomorphingtv/burrow/internal/config"
)

func runInit() error {
	cfgDir := config.DefaultConfigDir()
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	cfgPath := filepath.Join(cfgDir, "tasks.toml")
	if _, err := os.Stat(cfgPath); err == nil {
		fmt.Printf("Config already exists: %s\n", cfgPath)
		return nil
	}

	template := `[settings]
max_parallel = 4
log_dir = "~/.config/burrow/logs"

# Define your tasks here
[tasks.hello]
cmd = "echo hello from burrow"
description = "Simple hello world"
tags = ["example"]

# [tasks.my-script]
# cmd = "bash ~/scripts/my-script.sh"
# description = "My personal script"
# cwd = "~"
# env = { MY_VAR = "value" }
# tags = ["personal"]

# [schedules.nightly]
# task = "my-script"
# cron = "0 2 * * *"
`

	if err := os.WriteFile(cfgPath, []byte(template), 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	fmt.Printf("Created: %s\n", cfgPath)

	editor := os.Getenv("EDITOR")
	if editor != "" {
		fmt.Printf("Opening in %s...\n", editor)
	}

	return nil
}
