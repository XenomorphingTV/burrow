package v2

import (
	"context"
	"time"

	"github.com/XenomorphingTV/burrow/internal/config"
	"github.com/XenomorphingTV/burrow/internal/runner"
	"github.com/XenomorphingTV/burrow/internal/store"
	style "github.com/XenomorphingTV/burrow/internal/tui/v2/styles"
	"github.com/XenomorphingTV/burrow/internal/tui/v2/views/tasks"
	tea "github.com/charmbracelet/bubbletea"
)

// Tab represents which tab is currently active.
type Tab int

const (
	TabTasks Tab = iota
	TabSchedule
	TabHistory
	TabStats
	TabServices
)

// Model is the root bubbletea model. It owns all orchestration state and
// delegates rendering and tab-local key handling to sub-view models.
type Model struct {
	cfg   *config.Config
	st    store.Storer
	sched *runner.Scheduler
	pool  *runner.Pool
	keys  GlobalKeyMap
	theme style.Theme
	style style.Styles

	width     int
	height    int
	activeTab Tab
	showHelp  bool

	taskView tasks.Model
	// schedView   schedule.Model
	// historyView history.Model
	// statsView   stats.Model
	// serviceView services.Model

	// Orchestration state — not owned by any sub-view.
	executors      map[string]*runner.Executor
	watchers       map[string]context.CancelFunc
	pipelineQueues map[string][]string
	onFailureRuns  map[string]bool
	onSuccessRuns  map[string]bool

	tickCount int
}

func (m Model) Init() tea.Cmd {
	return tick()
}

func New(cfg *config.Config, st store.Storer, sched *runner.Scheduler, pool *runner.Pool) Model {
	theme := style.ResolveTheme(cfg.Settings.Theme)
	style := style.NewStyle(theme)

	return Model{
		cfg:   cfg,
		st:    st,
		sched: sched,
		pool:  pool,
		keys:  DefaultKeyMap(),
		theme: theme,
		style: style,

		activeTab: TabTasks,

		taskView: tasks.New(cfg, style),
		// schedView:   schedule.New(cfg, st, sched, style),
		// historyView: history.New(st, style),
		// statsView:   stats.New(style),
		// serviceView: services.New(style),

		executors:      make(map[string]*runner.Executor),
		watchers:       make(map[string]context.CancelFunc),
		pipelineQueues: make(map[string][]string),
		onFailureRuns:  make(map[string]bool),
		onSuccessRuns:  make(map[string]bool),
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case tickMsg:
		m.tickCount++
		return m, tick()
	}
	return m, nil
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}
	return m.taskView.View()
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

type tickMsg struct{}

func tick() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}
