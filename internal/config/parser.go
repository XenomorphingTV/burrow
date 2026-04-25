package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// DefaultConfigDir returns the default configuration directory (~/.config/burrow).
func DefaultConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".config/burrow"
	}
	return filepath.Join(home, ".config", "burrow")
}

// Load loads the global config, then merges the local config on top.
func Load() (*Config, error) {
	cfg := &Config{
		Tasks:     make(map[string]Task),
		Schedules: make(map[string]Schedule),
		Settings: Settings{
			MaxParallel: 4,
			LogDir:      filepath.Join(DefaultConfigDir(), "logs"),
		},
	}

	globalPath := filepath.Join(DefaultConfigDir(), "tasks.toml")
	if err := loadFile(cfg, globalPath); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	localPath := "burrow.toml"
	if _, err := os.Stat(localPath); err == nil {
		local := &Config{
			Tasks:     make(map[string]Task),
			Schedules: make(map[string]Schedule),
		}
		if err := loadFile(local, localPath); err != nil {
			return nil, err
		}
		mergeConfigs(cfg, local)
	}

	return cfg, nil
}

// LoadLocal reads only ./burrow.toml (no global merge).
func LoadLocal() (*Config, error) {
	cfg := &Config{
		Tasks:     make(map[string]Task),
		Schedules: make(map[string]Schedule),
	}
	if err := loadFile(cfg, "burrow.toml"); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return cfg, nil
}

// SaveLocal writes the config to ./burrow.toml using TOML encoding.
func SaveLocal(cfg *Config) error {
	f, err := os.Create("burrow.toml")
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(cfg)
}

// LoadGlobal reads only ~/.config/burrow/tasks.toml (no local merge).
func LoadGlobal() (*Config, error) {
	cfg := &Config{
		Tasks:     make(map[string]Task),
		Schedules: make(map[string]Schedule),
	}
	globalPath := filepath.Join(DefaultConfigDir(), "tasks.toml")
	if err := loadFile(cfg, globalPath); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return cfg, nil
}

// SaveGlobal writes the config to ~/.config/burrow/tasks.toml using TOML encoding.
func SaveGlobal(cfg *Config) error {
	cfgDir := DefaultConfigDir()
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		return err
	}
	f, err := os.Create(filepath.Join(cfgDir, "tasks.toml"))
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(cfg)
}

// loadFile decodes a TOML config file into cfg.
// Tasks support one level of namespace nesting: [tasks.db.seed] is loaded as
// Tasks["db.seed"]. A flat task [tasks.deploy] is loaded as Tasks["deploy"].
// Namespace vs flat is detected by whether the inner keys are known Task fields.
func loadFile(cfg *Config, path string) error {
	var raw struct {
		Settings  Settings               `toml:"settings"`
		Tasks     map[string]interface{} `toml:"tasks"`
		Schedules map[string]Schedule    `toml:"schedules"`
	}

	_, err := toml.DecodeFile(path, &raw)
	if err != nil {
		if os.IsNotExist(err) {
			return err
		}
		if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
			return statErr
		}
		return err
	}

	// Apply settings — only non-zero values override so defaults are preserved.
	if raw.Settings.MaxParallel != 0 {
		cfg.Settings.MaxParallel = raw.Settings.MaxParallel
	}
	if raw.Settings.LogDir != "" {
		cfg.Settings.LogDir = raw.Settings.LogDir
	}
	if raw.Settings.MaxLogRun != 0 {
		cfg.Settings.MaxLogRun = raw.Settings.MaxLogRun
	}
	if raw.Settings.MaxLogAge != 0 {
		cfg.Settings.MaxLogAge = raw.Settings.MaxLogAge
	}
	if len(raw.Settings.Notify) > 0 {
		cfg.Settings.Notify = raw.Settings.Notify
	}
	if raw.Settings.Terminal != "" {
		cfg.Settings.Terminal = raw.Settings.Terminal
	}
	if raw.Settings.Theme != "" {
		cfg.Settings.Theme = raw.Settings.Theme
	}

	// Flatten tasks, supporting one level of namespace nesting.
	// [tasks.deploy]       → Tasks["deploy"]
	// [tasks.db.seed]      → Tasks["db.seed"]
	for name, val := range raw.Tasks {
		inner, ok := val.(map[string]interface{})
		if !ok {
			continue
		}
		if looksLikeTask(inner) {
			cfg.Tasks[name] = taskFromMap(inner)
		} else {
			// Namespace group — each key is a sub-task.
			for subName, subVal := range inner {
				if subMap, ok := subVal.(map[string]interface{}); ok {
					cfg.Tasks[name+"."+subName] = taskFromMap(subMap)
				}
			}
		}
	}

	// Schedules are merged by name.
	for name, sched := range raw.Schedules {
		cfg.Schedules[name] = sched
	}

	return nil
}

// looksLikeTask returns true when m contains at least one known Task field,
// distinguishing a task table from a namespace group.
func looksLikeTask(m map[string]interface{}) bool {
	for k := range m {
		switch k {
		case "cmd", "description", "cwd", "env", "tags", "depends_on", "on_failure", "on_success", "notify", "external", "inputs", "timeout", "retries", "retry_delay", "watch":
			return true
		}
	}
	return false
}

// taskFromMap converts a raw TOML interface{} map to a Task.
func taskFromMap(m map[string]interface{}) Task {
	t := Task{}
	if v, ok := m["cmd"].(string); ok {
		t.Cmd = v
	}
	if v, ok := m["description"].(string); ok {
		t.Description = v
	}
	if v, ok := m["cwd"].(string); ok {
		t.Cwd = v
	}
	if env, ok := m["env"].(map[string]interface{}); ok {
		t.Env = make(map[string]string)
		for k, v := range env {
			if s, ok := v.(string); ok {
				t.Env[k] = s
			}
		}
	}
	if tags, ok := m["tags"].([]interface{}); ok {
		for _, v := range tags {
			if s, ok := v.(string); ok {
				t.Tags = append(t.Tags, s)
			}
		}
	}
	if deps, ok := m["depends_on"].([]interface{}); ok {
		for _, v := range deps {
			if s, ok := v.(string); ok {
				t.DependsOn = append(t.DependsOn, s)
			}
		}
	}
	if v, ok := m["on_failure"].(string); ok {
		t.OnFailure = v
	}
	if v, ok := m["on_success"].(string); ok {
		t.OnSuccess = v
	}
	if notify, ok := m["notify"].([]interface{}); ok {
		for _, v := range notify {
			if s, ok := v.(string); ok {
				t.Notify = append(t.Notify, s)
			}
		}
	}
	if v, ok := m["external"].(bool); ok {
		t.External = v
	}
	if v, ok := m["timeout"].(int64); ok {
		t.Timeout = int(v)
	}
	if v, ok := m["retries"].(int64); ok {
		t.Retries = int(v)
	}
	if v, ok := m["retry_delay"].(int64); ok {
		t.RetryDelay = int(v)
	}
	if watch, ok := m["watch"].([]interface{}); ok {
		for _, v := range watch {
			if s, ok := v.(string); ok {
				t.Watch = append(t.Watch, s)
			}
		}
	}
	// BurntSushi/toml decodes [[array.of.tables]] as []map[string]interface{},
	// but inline arrays decode as []interface{}. Handle both.
	var inputMaps []map[string]interface{}
	switch v := m["inputs"].(type) {
	case []map[string]interface{}:
		inputMaps = v
	case []interface{}:
		for _, ri := range v {
			if im, ok := ri.(map[string]interface{}); ok {
				inputMaps = append(inputMaps, im)
			}
		}
	}
	for _, im := range inputMaps {
		var inp TaskInput
		if v, ok := im["name"].(string); ok {
			inp.Name = v
		}
		if v, ok := im["prompt"].(string); ok {
			inp.Prompt = v
		}
		if opts, ok := im["options"].([]interface{}); ok {
			for _, o := range opts {
				if s, ok := o.(string); ok {
					inp.Options = append(inp.Options, s)
				}
			}
		}
		t.Inputs = append(t.Inputs, inp)
	}
	return t
}

// mergeConfigs merges local config on top of base config.
// Tasks: local fields override base; env and tags are unioned.
// Schedules: local overrides base by name.
// Settings: local overrides base for non-zero values.
func mergeConfigs(base, local *Config) {
	if local.Settings.MaxParallel != 0 {
		base.Settings.MaxParallel = local.Settings.MaxParallel
	}
	if local.Settings.LogDir != "" {
		base.Settings.LogDir = local.Settings.LogDir
	}
	if local.Settings.MaxLogRun != 0 {
		base.Settings.MaxLogRun = local.Settings.MaxLogRun
	}
	if local.Settings.MaxLogAge != 0 {
		base.Settings.MaxLogAge = local.Settings.MaxLogAge
	}
	if len(local.Settings.Notify) > 0 {
		base.Settings.Notify = local.Settings.Notify
	}
	if local.Settings.Terminal != "" {
		base.Settings.Terminal = local.Settings.Terminal
	}
	if local.Settings.Theme != "" {
		base.Settings.Theme = local.Settings.Theme
	}

	for name, localTask := range local.Tasks {
		if baseTask, exists := base.Tasks[name]; exists {
			if localTask.Cmd != "" {
				baseTask.Cmd = localTask.Cmd
			}
			if localTask.Description != "" {
				baseTask.Description = localTask.Description
			}
			if localTask.Cwd != "" {
				baseTask.Cwd = localTask.Cwd
			}
			if localTask.DependsOn != nil {
				baseTask.DependsOn = localTask.DependsOn
			}
			if localTask.OnFailure != "" {
				baseTask.OnFailure = localTask.OnFailure
			}
			if localTask.OnSuccess != "" {
				baseTask.OnSuccess = localTask.OnSuccess
			}
			if len(localTask.Notify) > 0 {
				baseTask.Notify = localTask.Notify
			}
			if len(localTask.Inputs) > 0 {
				baseTask.Inputs = localTask.Inputs
			}
			if localTask.Timeout != 0 {
				baseTask.Timeout = localTask.Timeout
			}
			if localTask.Retries != 0 {
				baseTask.Retries = localTask.Retries
			}
			if localTask.RetryDelay != 0 {
				baseTask.RetryDelay = localTask.RetryDelay
			}
			if len(localTask.Watch) > 0 {
				baseTask.Watch = localTask.Watch
			}
			if baseTask.Env == nil {
				baseTask.Env = make(map[string]string)
			}
			for k, v := range localTask.Env {
				baseTask.Env[k] = v
			}
			tagSet := make(map[string]struct{})
			for _, t := range baseTask.Tags {
				tagSet[t] = struct{}{}
			}
			for _, t := range localTask.Tags {
				if _, ok := tagSet[t]; !ok {
					baseTask.Tags = append(baseTask.Tags, t)
				}
			}
			base.Tasks[name] = baseTask
		} else {
			base.Tasks[name] = localTask
		}
	}

	for name, sched := range local.Schedules {
		base.Schedules[name] = sched
	}
}
