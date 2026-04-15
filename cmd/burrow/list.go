package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/XenomorphingTV/burrow/internal/config"
)

type taskJSON struct {
	Name        string            `json:"name"`
	Cmd         string            `json:"cmd"`
	Description string            `json:"description,omitempty"`
	Cwd         string            `json:"cwd,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	DependsOn   []string          `json:"depends_on,omitempty"`
	Watch       []string          `json:"watch,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
	Timeout     int               `json:"timeout,omitempty"`
	Retries     int               `json:"retries,omitempty"`
	OnFailure   string            `json:"on_failure,omitempty"`
	External    bool              `json:"external,omitempty"`
}

func runList(jsonOut bool) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if len(cfg.Tasks) == 0 {
		if jsonOut {
			fmt.Println("[]")
		} else {
			fmt.Println("No tasks configured.")
		}
		return nil
	}

	var names []string
	for n := range cfg.Tasks {
		names = append(names, n)
	}
	sort.Strings(names)

	if jsonOut {
		tasks := make([]taskJSON, 0, len(names))
		for _, n := range names {
			t := cfg.Tasks[n]
			tasks = append(tasks, taskJSON{
				Name:        n,
				Cmd:         t.Cmd,
				Description: t.Description,
				Cwd:         t.Cwd,
				Tags:        t.Tags,
				DependsOn:   t.DependsOn,
				Watch:       t.Watch,
				Env:         t.Env,
				Timeout:     t.Timeout,
				Retries:     t.Retries,
				OnFailure:   t.OnFailure,
				External:    t.External,
			})
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(tasks)
	}

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
