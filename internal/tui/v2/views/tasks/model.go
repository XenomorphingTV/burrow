package tasks

import (
	"sort"

	"github.com/XenomorphingTV/burrow/internal/config"
	style "github.com/XenomorphingTV/burrow/internal/tui/v2/styles"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

// TaskStatus represents the current state of a task.
type TaskStatus int

const (
	StatusIdle TaskStatus = iota
	StatusRunning
	StatusSuccess
	StatusFailed
	StatusWatching
)

// TaskEntry holds the display state for a single task.
type TaskEntry struct {
	Name       string
	Cfg        config.Task
	Status     TaskStatus
	DurationMs int64
	ExitCode   int
}

// Model is the self-contained bubbletea model for the tasks tab.
type Model struct {
	cfg    *config.Config
	styles style.Styles
	keys   KeyMap

	Width  int
	Height int

	Tasks    []TaskEntry
	Selected int
	TaskLogs map[string][]string

	Viewport   viewport.Model
	ScrollLock bool

	FilterMode  bool
	FilterInput string

	CollapsedGroups map[string]bool
	TickCount       int

	addTaskMode   bool
	addTaskStep   int
	addTaskInputs []textinput.Model
	addTaskErr    string

	promptMode      bool
	promptTaskName  string
	promptStep      int
	promptTextInput textinput.Model
	promptOptCursor int
	promptValues    map[string]string
}

func New(cfg *config.Config, styles style.Styles) Model {
	vp := viewport.New(80, 20)
	vp.Style = lipgloss.NewStyle().Background(styles.Base.GetBackground())

	placeholders := []string{
		"task-name (required)",
		"bash ~/scripts/foo.sh (required)",
		"What this task does",
		"tag1,tag2",
		"~/projects/foo",
	}
	addInputs := make([]textinput.Model, 5)
	for i := range addInputs {
		ai := textinput.New()
		ai.Placeholder = placeholders[i]
		ai.CharLimit = 256
		ai.Width = 40
		addInputs[i] = ai
	}

	var taskNames []string
	for name := range cfg.Tasks {
		taskNames = append(taskNames, name)
	}
	sort.Strings(taskNames)

	tasks := make([]TaskEntry, len(taskNames))
	for i, name := range taskNames {
		tasks[i] = TaskEntry{
			Name:   name,
			Cfg:    cfg.Tasks[name],
			Status: StatusIdle,
		}
	}

	return Model{
		cfg:             cfg,
		styles:          styles,
		keys:            DefaultKeyMap(),
		Tasks:           tasks,
		TaskLogs:        make(map[string][]string),
		Viewport:        vp,
		CollapsedGroups: make(map[string]bool),
		promptValues:    make(map[string]string),
		addTaskInputs:   addInputs,
	}
}
