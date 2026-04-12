package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/XenomorphingTV/burrow/internal/config"
)

func runList() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if len(cfg.Tasks) == 0 {
		fmt.Println("No tasks configured.")
		return nil
	}

	var names []string
	for n := range cfg.Tasks {
		names = append(names, n)
	}
	sort.Strings(names)

	maxLen := 0
	for _, n := range names {
		if len(n) > maxLen {
			maxLen = len(n)
		}
	}

	for _, n := range names {
		t := cfg.Tasks[n]
		desc := t.Description
		if desc == "" {
			desc = t.Cmd
		}
		tags := ""
		if len(t.Tags) > 0 {
			tags = " [" + strings.Join(t.Tags, ", ") + "]"
		}
		fmt.Printf("  %-*s  %s%s\n", maxLen, n, desc, tags)
	}

	return nil
}
