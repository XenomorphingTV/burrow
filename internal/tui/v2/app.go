package v2

import (
	"context"
	"time"

	"github.com/XenomorphingTV/burrow/internal/config"
	"github.com/XenomorphingTV/burrow/internal/runner"
	"github.com/XenomorphingTV/burrow/internal/store"
	style "github.com/XenomorphingTV/burrow/internal/tui/v2/styles"
	"github.com/XenomorphingTV/burrow/internal/tui/v2/messages"
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
	taskStartTimes map[string]time.Time
	watchers       map[string]context.CancelFunc
	pipelineQueues map[string][]string
	onFailureRuns  map[string]bool
	onSuccessRuns  map[string]bool
	pendingEnv     map[string]map[string]string

	tickCount   int
	pendingQuit bool
}

func (m Model) Init() tea.Cmd {
	return tick()
}

// contentHeight returns the vertical space available for the active tab view.
// Subtract tab bar and status bar heights here as they are added.
func (m Model) contentHeight() int {
	return m.height
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
		taskStartTimes: make(map[string]time.Time),
		watchers:       make(map[string]context.CancelFunc),
		pipelineQueues: make(map[string][]string),
		onFailureRuns:  make(map[string]bool),
		onSuccessRuns:  make(map[string]bool),
		pendingEnv:     make(map[string]map[string]string),
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.taskView.Width = msg.Width
		m.taskView.Height = m.contentHeight()
		m.taskView.RecalcViewport()
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case tickMsg:
		m.tickCount++
		m.taskView.TickCount = m.tickCount
		return m, tick()

	case messages.RunTaskMessage:
		if len(msg.Env) > 0 {
			m.pendingEnv[msg.Name] = msg.Env
		}
		return m.startPipeline(msg.Name, msg.Trigger)

	case messages.KillTaskMessage:
		exec, ok := m.executors[msg.Name]
		if !ok {
			break
		}
		exec.Kill()
		if cancel, ok := m.watchers[msg.Name]; ok {
			cancel()
			delete(m.watchers, msg.Name)
		}
		for i, t := range m.taskView.Tasks {
			if t.Name == msg.Name {
				m.taskView.Tasks[i].Status = tasks.StatusFailed
				break
			}
		}

	case runner.LogLine:
		name := msg.TaskName
		if msg.Done {
			delete(m.executors, name)
			m.pool.Release()

			durationMs := int64(0)
			if start, ok := m.taskStartTimes[name]; ok {
				durationMs = time.Since(start).Milliseconds()
				delete(m.taskStartTimes, name)
			}

			status := tasks.StatusSuccess
			if msg.ExitCode != 0 {
				status = tasks.StatusFailed
			}
			for i, t := range m.taskView.Tasks {
				if t.Name == name {
					m.taskView.Tasks[i].Status = status
					m.taskView.Tasks[i].DurationMs = durationMs
					m.taskView.Tasks[i].ExitCode = msg.ExitCode
					break
				}
			}
			m.taskView.UpdateViewportForSelected()

			// Continue pipeline if task succeeded
			if msg.ExitCode == 0 {
				if queue, ok := m.pipelineQueues[name]; ok && len(queue) > 0 {
					next := queue[0]
					remaining := queue[1:]
					delete(m.pipelineQueues, name)
					if len(remaining) > 0 {
						m.pipelineQueues[next] = remaining
					}
					return m.startTask(next, "pipeline")
				}
			}

			// Fire hooks
			task, ok := m.cfg.Tasks[name]
			if ok {
				if msg.ExitCode == 0 && task.OnSuccess != "" && !m.onSuccessRuns[name] {
					return m.fireOnSuccess(name, task.OnSuccess)
				}
				if msg.ExitCode != 0 && task.OnFailure != "" && !m.onFailureRuns[name] {
					return m.fireOnFailure(name, task.OnFailure)
				}
				// Start watcher if task has watch patterns and succeeded
				if msg.ExitCode == 0 && len(task.Watch) > 0 {
					ctx, cancel := context.WithCancel(context.Background())
					m.watchers[name] = cancel
					for i, t := range m.taskView.Tasks {
						if t.Name == name {
							m.taskView.Tasks[i].Status = tasks.StatusWatching
							break
						}
					}
					return m, startWatcher(ctx, name, task.Watch, task.Cwd)
				}
			}
			delete(m.onSuccessRuns, name)
			delete(m.onFailureRuns, name)
			return m, nil
		}

		// Streaming line — append and keep listening
		m.taskView.TaskLogs[name] = append(m.taskView.TaskLogs[name], msg.Text)
		m.taskView.UpdateViewportForSelected()
		exec, ok := m.executors[name]
		if !ok {
			return m, nil
		}
		return m, awaitLog(exec.LogCh())

	case messages.WatchTriggeredMessage:
		return m.startTask(msg.TaskName, "watch")
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
		if m.pendingQuit {
			return m, tea.Quit
		}
		m.pendingQuit = true
		return m, nil
	default:
		m.pendingQuit = false
	}

	switch m.activeTab {
	case TabTasks:
		var cmd tea.Cmd
		m.taskView, cmd = m.taskView.Update(msg)
		return m, cmd
	}

	return m, nil
}

type tickMsg struct{}

func tick() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}

func (m Model) startTask(name, trigger string) (Model, tea.Cmd) {
	if _, running := m.executors[name]; running {
		return m, nil
	}

	task, ok := m.cfg.Tasks[name]
	if !ok {
		return m, nil
	}

	if extras, ok := m.pendingEnv[name]; ok {
		merged := make(map[string]string, len(task.Env)+len(extras))
		for k, v := range task.Env {
			merged[k] = v
		}
		for k, v := range extras {
			merged[k] = v
		}
		task.Env = merged
		delete(m.pendingEnv, name)
	}

	if task.External {
		m.taskView.TaskLogs[name] = nil
		m.taskView.ScrollLock = false
		err := runner.LaunchExternal(task.Cmd, task.Cwd, m.cfg.Settings.Terminal, task.Env)
		var msg string
		if err != nil {
			msg = "[err] " + err.Error()
		} else {
			msg = "launched in external terminal"
		}
		m.taskView.TaskLogs[name] = []string{"$ " + task.Cmd, msg}
		m.taskView.UpdateViewportForSelected() // TODO: implement on tasks.Model
		return m, nil
	}

	if !m.pool.TryAcquire() {
		return m, nil
	}

	exec := runner.NewExecutor(name, task, trigger, m.cfg.Settings.LogDir, m.st, m.cfg.Settings.Notify)
	exec.Start()
	m.executors[name] = exec
	m.taskStartTimes[name] = time.Now()

	for i, t := range m.taskView.Tasks {
		if t.Name == name {
			m.taskView.Tasks[i].Status = tasks.StatusRunning
			break
		}
	}

	m.taskView.TaskLogs[name] = nil
	m.taskView.ScrollLock = false

	return m, awaitLog(exec.LogCh())
}

func (m Model) startPipeline(target, trigger string) (Model, tea.Cmd) {
	ordered, err := runner.Resolve(target, m.cfg.Tasks)
	if err != nil || len(ordered) == 0 {
		return m, nil
	}

	if len(ordered) == 1 {
		return m.startTask(ordered[0], trigger)
	}

	m.pipelineQueues[ordered[0]] = ordered[1:]
	return m.startTask(ordered[0], "pipeline")
}

func (m Model) fireOnFailure(parentName, onFailure string) (Model, tea.Cmd) {
	if _, ok := m.cfg.Tasks[onFailure]; ok {
		m.onFailureRuns[onFailure] = true
		return m.startTask(onFailure, "on_failure")
	}

	return m, func() tea.Msg {
		task := config.Task{Cmd: onFailure}
		exec := runner.NewExecutor(parentName+".on_failure", task, "on_failure", m.cfg.Settings.LogDir, m.st, nil)
		exec.Start()
		for range exec.LogCh() {
		}
		return nil
	}
}

func (m Model) fireOnSuccess(parentName, onSuccess string) (Model, tea.Cmd) {
	if _, ok := m.cfg.Tasks[onSuccess]; ok {
		m.onSuccessRuns[onSuccess] = true
		return m.startTask(onSuccess, "on_success")
	}
	return m, func() tea.Msg {
		task := config.Task{Cmd: onSuccess}
		exec := runner.NewExecutor(parentName+".on_success", task, "on_success", m.cfg.Settings.LogDir, m.st, nil)
		exec.Start()
		for range exec.LogCh() {
		}
		return nil
	}
}
