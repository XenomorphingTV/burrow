package config

// Settings holds global application settings.
type Settings struct {
	MaxParallel int      `toml:"max_parallel"`
	LogDir      string   `toml:"log_dir"`
	MaxLogRun   int      `toml:"max_log_run"` // max history records to keep; 0 = unlimited
	MaxLogAge   int      `toml:"max_log_age"` // max age of records and log files in days; 0 = unlimited
	Notify      []string `toml:"notify"`      // default notification events: "success", "failure"
	Terminal    string   `toml:"terminal"`    // terminal emulator for external tasks; auto-detected if empty
}

// Task represents a named task definition.
type Task struct {
	Cmd         string            `toml:"cmd"`
	Description string            `toml:"description"`
	Cwd         string            `toml:"cwd"`
	Env         map[string]string `toml:"env"`
	Tags        []string          `toml:"tags"`
	DependsOn   []string          `toml:"depends_on"`
	OnFailure   string            `toml:"on_failure"`
	Notify      []string          `toml:"notify"`   // overrides settings.notify for this task
	External    bool              `toml:"external"` // launch in a terminal emulator instead of capturing output
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
