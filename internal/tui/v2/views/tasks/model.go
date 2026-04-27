package tasks

import (
	"sort"
	"strings"

	"github.com/XenomorphingTV/burrow/internal/config"
	"github.com/XenomorphingTV/burrow/internal/tui/v2/messages"
	style "github.com/XenomorphingTV/burrow/internal/tui/v2/styles"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
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

func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Up):
		if m.Selected > 0 {
			m.Selected--
			m.UpdateViewportForSelected()
		}

	case key.Matches(msg, m.keys.Down):
		if m.Selected < len(m.visibleItems())-1 {
			m.Selected++
			m.UpdateViewportForSelected()
		}

	case key.Matches(msg, m.keys.Run):
		name := m.selectedTaskName()
		if name == "" {
			break
		}
		if task := m.cfg.Tasks[name]; len(task.Inputs) > 0 {
			m.promptMode = true
			m.promptTaskName = name
			m.promptStep = 0
			m.promptValues = make(map[string]string)
			m = m.initPromptStep()
			m.recalcViewport()
			return m, textinput.Blink
		}
		return m, func() tea.Msg {
			return messages.RunTaskMessage{Name: name, Trigger: "manual"}
		}

	case key.Matches(msg, m.keys.Kill):
		name := m.selectedTaskName()
		if name == "" {
			break
		}
		return m, func() tea.Msg {
			return messages.KillTaskMessage{Name: name}
		}

	case key.Matches(msg, m.keys.Clear):
		name := m.selectedTaskName()
		m.TaskLogs[name] = nil
		m.ScrollLock = false
		m.Viewport.SetContent("")

	case key.Matches(msg, m.keys.ScrollUp):
		m.ScrollLock = true
		m.Viewport.HalfViewUp()

	case key.Matches(msg, m.keys.ScrollDown):
		m.Viewport.HalfViewDown()
		if m.Viewport.AtBottom() {
			m.ScrollLock = false
		}

	case key.Matches(msg, m.keys.AddTask):
		m.addTaskMode = true
		m.addTaskStep = 0
		m.addTaskInputs[0].Focus()
		m.recalcViewport()

	case key.Matches(msg, m.keys.Filter):
		m.FilterMode = true
		m.FilterInput = ""

	case key.Matches(msg, m.keys.Toggle):
		items := m.visibleItems()
		if m.Selected < len(items) && items[m.Selected].IsGroup {
			selectedName := items[m.Selected].Name
			if m.CollapsedGroups[selectedName] {
				delete(m.CollapsedGroups, selectedName)
			} else {
				m.CollapsedGroups[selectedName] = true
			}
			if newLen := len(m.visibleItems()); m.Selected >= newLen {
				m.Selected = max(0, newLen-1)
			}
		}

	case key.Matches(msg, m.keys.Edit):
		name := m.selectedTaskName()
		if name == "" {
			break
		}
		path := configFileForTask(name)
		line := taskLineInFile(path, name)
		c := openInEditor(path, line)
		return m, tea.Exec(c, func(err error) tea.Msg { return nil })
	}
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if m.promptMode {
		return m.handlePromptKey(msg)
	}
	if m.addTaskMode {
		return m.handleAddTask(msg)
	}
	if m.FilterMode {
		return m.handleFilterKey(msg)
	}

	if keyMessage, ok := msg.(tea.KeyMsg); ok {
		return m.handleKey(keyMessage)
	}

	return m, nil
}

func (m *Model) UpdateViewportForSelected() {
	name := m.selectedTaskName()
	lines := m.TaskLogs[name]
	m.Viewport.SetContent(strings.Join(lines, "\n"))
	if !m.ScrollLock {
		m.Viewport.GotoBottom()
	}
}

func (m *Model) UpdateViewportContent(name string) {
	lines := m.TaskLogs[name]
	m.Viewport.SetContent(strings.Join(lines, "\n"))
	if !m.ScrollLock {
		m.Viewport.GotoBottom()
	}
}
