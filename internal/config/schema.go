package config

// Settings holds global application settings.
type Settings struct {
	MaxParallel int      `toml:"max_parallel"`
	LogDir      string   `toml:"log_dir"`
	MaxLogRun   int      `toml:"max_log_run"` // max history records to keep; 0 = unlimited
	MaxLogAge   int      `toml:"max_log_age"` // max age of records and log files in days; 0 = unlimited
	Notify      []string `toml:"notify"`      // default notification events: "success", "failure"
	Terminal    string   `toml:"terminal"`    // terminal emulator for external tasks; auto-detected if empty
	Theme       string   `toml:"theme"`       // TUI color theme: catppuccin-mocha (default), nord, dracula, gruvbox
}

// TaskInput defines a runtime prompt for a task parameter.
type TaskInput struct {
	Name    string   `toml:"name"`
	Prompt  string   `toml:"prompt"`
	Options []string `toml:"options"` // if non-empty, present as a selection list
}

// Task represents a named task definition.
type Task struct {
	Cmd         string            `toml:"cmd"`
	Timeout     int               `toml:"timeout"` // kill task after N seconds; 0 = no timeout
	Retries     int               `toml:"retries"`
	RetryDelay  int               `toml:"retry_delay"`
	Description string            `toml:"description"`
	Cwd         string            `toml:"cwd"`
	Env         map[string]string `toml:"env"`
	Tags        []string          `toml:"tags"`
	DependsOn   []string          `toml:"depends_on"`
	OnFailure   string            `toml:"on_failure"`
	OnSuccess   string            `toml:"on_success"`
	Notify      []string          `toml:"notify"`   // overrides settings.notify for this task
	External    bool              `toml:"external"` // launch in a terminal emulator instead of capturing output
	Inputs      []TaskInput       `toml:"inputs"`
	Watch       []string          `toml:"watch"` // glob patterns; re-run task when matched files change
}

// Schedule represents a cron-triggered task.
type Schedule struct {
	Task string `toml:"task"`
	Cron string `toml:"cron"`
}

// Config is the root configuration structure.
type Config struct {
	Settings  Settings            `toml:"settings"`
	Tasks     map[string]Task     `toml:"tasks"`
	Schedules map[string]Schedule `toml:"schedules"`
}
