package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTempTOML(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "*.toml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return f.Name()
}

func newCfg() *Config {
	return &Config{
		Tasks:     make(map[string]Task),
		Schedules: make(map[string]Schedule),
	}
}

// loadFile tests

func TestLoadFileSettings(t *testing.T) {
	path := writeTempTOML(t, `
[settings]
max_parallel = 8
log_dir = "/tmp/logs"
`)
	cfg := newCfg()
	if err := loadFile(cfg, path); err != nil {
		t.Fatal(err)
	}
	if cfg.Settings.MaxParallel != 8 {
		t.Errorf("max_parallel: got %d, want 8", cfg.Settings.MaxParallel)
	}
	if cfg.Settings.LogDir != "/tmp/logs" {
		t.Errorf("log_dir: got %q, want /tmp/logs", cfg.Settings.LogDir)
	}
}

func TestLoadFileFlatTask(t *testing.T) {
	path := writeTempTOML(t, `
[tasks.hello]
cmd = "echo hello"
description = "Say hi"
tags = ["greet"]
`)
	cfg := newCfg()
	if err := loadFile(cfg, path); err != nil {
		t.Fatal(err)
	}
	task, ok := cfg.Tasks["hello"]
	if !ok {
		t.Fatal("expected task 'hello'")
	}
	if task.Cmd != "echo hello" {
		t.Errorf("cmd: got %q, want 'echo hello'", task.Cmd)
	}
	if task.Description != "Say hi" {
		t.Errorf("description: got %q", task.Description)
	}
	if len(task.Tags) != 1 || task.Tags[0] != "greet" {
		t.Errorf("tags: got %v", task.Tags)
	}
}

func TestLoadFileNamespacedTasks(t *testing.T) {
	path := writeTempTOML(t, `
[tasks.db.seed]
cmd = "node seed.js"

[tasks.db.migrate]
cmd = "node migrate.js"
`)
	cfg := newCfg()
	if err := loadFile(cfg, path); err != nil {
		t.Fatal(err)
	}
	if _, ok := cfg.Tasks["db.seed"]; !ok {
		t.Error("expected task 'db.seed'")
	}
	if _, ok := cfg.Tasks["db.migrate"]; !ok {
		t.Error("expected task 'db.migrate'")
	}
}

func TestLoadFileSchedule(t *testing.T) {
	path := writeTempTOML(t, `
[tasks.hello]
cmd = "echo hi"

[schedules.nightly]
task = "hello"
cron = "0 2 * * *"
`)
	cfg := newCfg()
	if err := loadFile(cfg, path); err != nil {
		t.Fatal(err)
	}
	sched, ok := cfg.Schedules["nightly"]
	if !ok {
		t.Fatal("expected schedule 'nightly'")
	}
	if sched.Task != "hello" || sched.Cron != "0 2 * * *" {
		t.Errorf("unexpected schedule: %+v", sched)
	}
}

func TestLoadFileTaskEnvAndDeps(t *testing.T) {
	path := writeTempTOML(t, `
[tasks.seed]
cmd = "node seed.js"
cwd = "./backend"
depends_on = ["migrate"]
on_failure = "notify"

[tasks.seed.env]
NODE_ENV = "development"
`)
	cfg := newCfg()
	if err := loadFile(cfg, path); err != nil {
		t.Fatal(err)
	}
	task, ok := cfg.Tasks["seed"]
	if !ok {
		t.Fatal("expected task 'seed'")
	}
	if task.Cwd != "./backend" {
		t.Errorf("cwd: got %q", task.Cwd)
	}
	if len(task.DependsOn) != 1 || task.DependsOn[0] != "migrate" {
		t.Errorf("depends_on: got %v", task.DependsOn)
	}
	if task.OnFailure != "notify" {
		t.Errorf("on_failure: got %q", task.OnFailure)
	}
	if task.Env["NODE_ENV"] != "development" {
		t.Errorf("env: got %v", task.Env)
	}
}

func TestLoadFileNotExist(t *testing.T) {
	cfg := newCfg()
	err := loadFile(cfg, "/nonexistent/path/file.toml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !os.IsNotExist(err) {
		t.Errorf("expected IsNotExist error, got: %v", err)
	}
}

func TestLoadFileDoesNotOverrideWithZero(t *testing.T) {
	// Settings already set; a file with no [settings] block should not reset them.
	path := writeTempTOML(t, `
[tasks.hello]
cmd = "echo hi"
`)
	cfg := newCfg()
	cfg.Settings.MaxParallel = 4
	cfg.Settings.LogDir = "/original/logs"

	if err := loadFile(cfg, path); err != nil {
		t.Fatal(err)
	}
	if cfg.Settings.MaxParallel != 4 {
		t.Errorf("max_parallel should not be overridden with zero value")
	}
	if cfg.Settings.LogDir != "/original/logs" {
		t.Errorf("log_dir should not be overridden with empty string")
	}
}

// mergeConfigs tests

func TestMergeConfigsOverridesCmd(t *testing.T) {
	base := &Config{
		Tasks:     map[string]Task{"deploy": {Cmd: "old-cmd"}},
		Schedules: make(map[string]Schedule),
	}
	local := &Config{
		Tasks:     map[string]Task{"deploy": {Cmd: "new-cmd"}},
		Schedules: make(map[string]Schedule),
	}
	mergeConfigs(base, local)
	if base.Tasks["deploy"].Cmd != "new-cmd" {
		t.Errorf("expected new-cmd, got %q", base.Tasks["deploy"].Cmd)
	}
}

func TestMergeConfigsUnionTags(t *testing.T) {
	base := &Config{
		Tasks:     map[string]Task{"t": {Tags: []string{"a", "b"}}},
		Schedules: make(map[string]Schedule),
	}
	local := &Config{
		Tasks:     map[string]Task{"t": {Tags: []string{"b", "c"}}},
		Schedules: make(map[string]Schedule),
	}
	mergeConfigs(base, local)

	seen := make(map[string]int)
	for _, tag := range base.Tasks["t"].Tags {
		seen[tag]++
	}
	for _, want := range []string{"a", "b", "c"} {
		if seen[want] == 0 {
			t.Errorf("tag %q missing after merge", want)
		}
		if seen[want] > 1 {
			t.Errorf("tag %q duplicated after merge", want)
		}
	}
}

func TestMergeConfigsUnionEnv(t *testing.T) {
	base := &Config{
		Tasks:     map[string]Task{"t": {Env: map[string]string{"A": "1"}}},
		Schedules: make(map[string]Schedule),
	}
	local := &Config{
		Tasks:     map[string]Task{"t": {Env: map[string]string{"B": "2"}}},
		Schedules: make(map[string]Schedule),
	}
	mergeConfigs(base, local)
	env := base.Tasks["t"].Env
	if env["A"] != "1" || env["B"] != "2" {
		t.Errorf("env merge incorrect: %v", env)
	}
}

func TestMergeConfigsLocalEnvOverridesBase(t *testing.T) {
	base := &Config{
		Tasks:     map[string]Task{"t": {Env: map[string]string{"A": "base"}}},
		Schedules: make(map[string]Schedule),
	}
	local := &Config{
		Tasks:     map[string]Task{"t": {Env: map[string]string{"A": "local"}}},
		Schedules: make(map[string]Schedule),
	}
	mergeConfigs(base, local)
	if base.Tasks["t"].Env["A"] != "local" {
		t.Errorf("local env should override base; got %q", base.Tasks["t"].Env["A"])
	}
}

func TestMergeConfigsAddsNewTask(t *testing.T) {
	base := &Config{
		Tasks:     map[string]Task{"a": {Cmd: "echo a"}},
		Schedules: make(map[string]Schedule),
	}
	local := &Config{
		Tasks:     map[string]Task{"b": {Cmd: "echo b"}},
		Schedules: make(map[string]Schedule),
	}
	mergeConfigs(base, local)
	if _, ok := base.Tasks["b"]; !ok {
		t.Error("local-only task 'b' should be added to base")
	}
}

func TestMergeConfigsSettings(t *testing.T) {
	base := &Config{
		Settings:  Settings{MaxParallel: 4, LogDir: "/base/logs"},
		Tasks:     make(map[string]Task),
		Schedules: make(map[string]Schedule),
	}
	local := &Config{
		Settings:  Settings{MaxParallel: 8},
		Tasks:     make(map[string]Task),
		Schedules: make(map[string]Schedule),
	}
	mergeConfigs(base, local)
	if base.Settings.MaxParallel != 8 {
		t.Errorf("max_parallel: got %d, want 8", base.Settings.MaxParallel)
	}
	if base.Settings.LogDir != "/base/logs" {
		t.Error("log_dir should not be overridden by a zero value from local")
	}
}

func TestMergeConfigsScheduleOverride(t *testing.T) {
	base := &Config{
		Tasks:     make(map[string]Task),
		Schedules: map[string]Schedule{"nightly": {Task: "foo", Cron: "0 2 * * *"}},
	}
	local := &Config{
		Tasks:     make(map[string]Task),
		Schedules: map[string]Schedule{"nightly": {Task: "foo", Cron: "0 3 * * *"}},
	}
	mergeConfigs(base, local)
	if base.Schedules["nightly"].Cron != "0 3 * * *" {
		t.Errorf("local schedule should override base; got %q", base.Schedules["nightly"].Cron)
	}
}

// looksLikeTask tests

func TestLooksLikeTask(t *testing.T) {
	cases := []struct {
		name string
		m    map[string]interface{}
		want bool
	}{
		{"has cmd", map[string]interface{}{"cmd": "echo hi"}, true},
		{"has description", map[string]interface{}{"description": "something"}, true},
		{"has tags", map[string]interface{}{"tags": []interface{}{"a"}}, true},
		{"has env", map[string]interface{}{"env": map[string]interface{}{}}, true},
		{"has depends_on", map[string]interface{}{"depends_on": []interface{}{}}, true},
		{"has cwd", map[string]interface{}{"cwd": "/tmp"}, true},
		{"has inputs", map[string]interface{}{"inputs": []interface{}{}}, true},
		{"namespace group", map[string]interface{}{"seed": map[string]interface{}{"cmd": "node seed.js"}}, false},
		{"unknown key only", map[string]interface{}{"foo": "bar"}, false},
		{"empty", map[string]interface{}{}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := looksLikeTask(tc.m)
			if got != tc.want {
				t.Errorf("looksLikeTask() = %v, want %v", got, tc.want)
			}
		})
	}
}

// inputs parsing tests

func TestLoadFileTaskInputsArrayOfTables(t *testing.T) {
	// [[tasks.x.inputs]] produces []map[string]interface{} in BurntSushi/toml —
	// this is the format the example config uses and the original bug.
	path := writeTempTOML(t, `
[tasks.deploy]
cmd = "deploy.sh --env $BURROW_ENV --tag $BURROW_TAG"
description = "Deploy"

[[tasks.deploy.inputs]]
name   = "BURROW_ENV"
prompt = "Environment?"
options = ["staging", "prod"]

[[tasks.deploy.inputs]]
name   = "BURROW_TAG"
prompt = "Git tag?"
`)
	cfg := newCfg()
	if err := loadFile(cfg, path); err != nil {
		t.Fatal(err)
	}
	task, ok := cfg.Tasks["deploy"]
	if !ok {
		t.Fatal("expected task 'deploy'")
	}
	if len(task.Inputs) != 2 {
		t.Fatalf("expected 2 inputs, got %d", len(task.Inputs))
	}
	if task.Inputs[0].Name != "BURROW_ENV" {
		t.Errorf("input[0].name: got %q, want BURROW_ENV", task.Inputs[0].Name)
	}
	if task.Inputs[0].Prompt != "Environment?" {
		t.Errorf("input[0].prompt: got %q", task.Inputs[0].Prompt)
	}
	if len(task.Inputs[0].Options) != 2 || task.Inputs[0].Options[0] != "staging" {
		t.Errorf("input[0].options: got %v", task.Inputs[0].Options)
	}
	if task.Inputs[1].Name != "BURROW_TAG" {
		t.Errorf("input[1].name: got %q, want BURROW_TAG", task.Inputs[1].Name)
	}
	if len(task.Inputs[1].Options) != 0 {
		t.Errorf("input[1] should have no options, got %v", task.Inputs[1].Options)
	}
}

func TestLoadFileTaskInputsNoOptions(t *testing.T) {
	path := writeTempTOML(t, `
[tasks.greet]
cmd = "echo $BURROW_NAME"

[[tasks.greet.inputs]]
name   = "BURROW_NAME"
prompt = "Your name?"
`)
	cfg := newCfg()
	if err := loadFile(cfg, path); err != nil {
		t.Fatal(err)
	}
	task := cfg.Tasks["greet"]
	if len(task.Inputs) != 1 {
		t.Fatalf("expected 1 input, got %d", len(task.Inputs))
	}
	if task.Inputs[0].Name != "BURROW_NAME" || task.Inputs[0].Prompt != "Your name?" {
		t.Errorf("unexpected input: %+v", task.Inputs[0])
	}
}

func TestMergeConfigsInputsOverride(t *testing.T) {
	base := &Config{
		Tasks: map[string]Task{
			"deploy": {Inputs: []TaskInput{{Name: "OLD", Prompt: "Old prompt?"}}},
		},
		Schedules: make(map[string]Schedule),
	}
	local := &Config{
		Tasks: map[string]Task{
			"deploy": {Inputs: []TaskInput{{Name: "NEW", Prompt: "New prompt?"}}},
		},
		Schedules: make(map[string]Schedule),
	}
	mergeConfigs(base, local)
	inputs := base.Tasks["deploy"].Inputs
	if len(inputs) != 1 || inputs[0].Name != "NEW" {
		t.Errorf("local inputs should replace base inputs; got %+v", inputs)
	}
}

func TestMergeConfigsInputsPreservedWhenLocalEmpty(t *testing.T) {
	base := &Config{
		Tasks: map[string]Task{
			"deploy": {Inputs: []TaskInput{{Name: "ENV", Prompt: "Env?"}}},
		},
		Schedules: make(map[string]Schedule),
	}
	local := &Config{
		Tasks:     map[string]Task{"deploy": {Cmd: "new-cmd"}},
		Schedules: make(map[string]Schedule),
	}
	mergeConfigs(base, local)
	inputs := base.Tasks["deploy"].Inputs
	if len(inputs) != 1 || inputs[0].Name != "ENV" {
		t.Errorf("base inputs should be preserved when local has none; got %+v", inputs)
	}
}

// integration: Load picks up a file from a temp dir

func TestLoadFileRoundTrip(t *testing.T) {
	dir := t.TempDir()
	content := `
[settings]
max_parallel = 3

[tasks.greet]
cmd = "echo hi"
description = "Say hi"
tags = ["personal"]
`
	if err := os.WriteFile(filepath.Join(dir, "tasks.toml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		Tasks:     make(map[string]Task),
		Schedules: make(map[string]Schedule),
		Settings:  Settings{MaxParallel: 4, LogDir: filepath.Join(dir, "logs")},
	}
	if err := loadFile(cfg, filepath.Join(dir, "tasks.toml")); err != nil {
		t.Fatal(err)
	}

	if cfg.Settings.MaxParallel != 3 {
		t.Errorf("max_parallel: got %d, want 3", cfg.Settings.MaxParallel)
	}
	task, ok := cfg.Tasks["greet"]
	if !ok {
		t.Fatal("expected task 'greet'")
	}
	if task.Cmd != "echo hi" {
		t.Errorf("cmd: got %q", task.Cmd)
	}
}
