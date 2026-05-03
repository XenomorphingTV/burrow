package v2

import (
	"context"
	"sort"
	"time"

	"github.com/XenomorphingTV/burrow/internal/config"
	"github.com/XenomorphingTV/burrow/internal/runner"
	"github.com/XenomorphingTV/burrow/internal/store"
	style "github.com/XenomorphingTV/burrow/internal/tui/v2/styles"
	"github.com/XenomorphingTV/burrow/internal/tui/v2/messages"
	"github.com/XenomorphingTV/burrow/internal/tui/v2/views/schedule"
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

	taskView  tasks.Model
	schedView schedule.Model

	disabledSchedules map[string]bool

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

type initMsg struct{}

func (m Model) Init() tea.Cmd {
	return tea.Batch(tick(), func() tea.Msg { return initMsg{} })
}

// contentHeight returns the vertical space available for the active tab view,
// subtracting the tab bar (2 lines: bar + separator) and status bar (1 line).
func (m Model) contentHeight() int {
	if m.height == 0 {
		return 0
	}
	return m.height - 4 // 2 for tab bar (bar + separator) + 2 for status bar (separator + bar)
}

func New(cfg *config.Config, st store.Storer, sched *runner.Scheduler, pool *runner.Pool) Model {
	theme := style.ResolveTheme(cfg.Settings.Theme)
	style := style.NewStyle(theme)

	disabled := make(map[string]bool)
	if st != nil {
		if loaded, err := st.LoadDisabledSchedules(); err == nil {
			disabled = loaded
		}
	}

	m := Model{
		cfg:   cfg,
		st:    st,
		sched: sched,
		pool:  pool,
		keys:  DefaultKeyMap(),
		theme: theme,
		style: style,

		activeTab: TabTasks,

		taskView:          tasks.New(cfg, style),
		schedView:         schedule.New(cfg, style, theme),
		disabledSchedules: disabled,

		executors:      make(map[string]*runner.Executor),
		taskStartTimes: make(map[string]time.Time),
		watchers:       make(map[string]context.CancelFunc),
		pipelineQueues: make(map[string][]string),
		onFailureRuns:  make(map[string]bool),
		onSuccessRuns:  make(map[string]bool),
		pendingEnv:     make(map[string]map[string]string),
	}
	m.schedView.Entries = m.buildScheduleEntries()
	return m
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.taskView.Width = msg.Width
		m.taskView.Height = m.contentHeight()
		m.taskView.RecalcViewport()
		m.schedView.Width = msg.Width
		m.schedView.Height = m.contentHeight()
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
				// Start watcher if task has watch patterns and succeeded and not disabled
				if msg.ExitCode == 0 && len(task.Watch) > 0 && !m.disabledSchedules["watch:"+name] {
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

	case initMsg:
		var cmds []tea.Cmd
		for name, task := range m.cfg.Tasks {
			if len(task.Watch) > 0 && !m.disabledSchedules["watch:"+name] {
				ctx, cancel := context.WithCancel(context.Background())
				m.watchers[name] = cancel
				for i, t := range m.taskView.Tasks {
					if t.Name == name {
						m.taskView.Tasks[i].Status = tasks.StatusWatching
						break
					}
				}
				cmds = append(cmds, startWatcher(ctx, name, task.Watch, task.Cwd))
			}
		}
		m.schedView.Entries = m.buildScheduleEntries()
		return m, tea.Batch(cmds...)

	case messages.WatchTriggeredMessage:
		return m.startTask(msg.TaskName, "watch")

	case messages.ToggleScheduleMessage:
		name := msg.Name
		if msg.Kind == "cron" {
			if m.sched != nil {
				if m.sched.IsEnabled(name) {
					m.sched.Disable(name)
					m.disabledSchedules[name] = true
				} else {
					m.sched.Enable(name) //nolint:errcheck
					delete(m.disabledSchedules, name)
				}
			}
		} else {
			key := "watch:" + name
			if m.disabledSchedules[key] {
				delete(m.disabledSchedules, key)
				if _, running := m.executors[name]; !running {
					if task, ok := m.cfg.Tasks[name]; ok {
						ctx, cancel := context.WithCancel(context.Background())
						m.watchers[name] = cancel
						for i, t := range m.taskView.Tasks {
							if t.Name == name {
								m.taskView.Tasks[i].Status = tasks.StatusWatching
								break
							}
						}
						if m.st != nil {
							m.st.SaveDisabledSchedules(m.disabledSchedules) //nolint:errcheck
						}
						m.schedView.Entries = m.buildScheduleEntries()
						return m, startWatcher(ctx, name, task.Watch, task.Cwd)
					}
				}
			} else {
				m.disabledSchedules["watch:"+name] = true
				if cancel, ok := m.watchers[name]; ok {
					cancel()
					delete(m.watchers, name)
					for i, t := range m.taskView.Tasks {
						if t.Name == name && t.Status == tasks.StatusWatching {
							m.taskView.Tasks[i].Status = tasks.StatusIdle
							break
						}
					}
				}
			}
		}
		if m.st != nil {
			m.st.SaveDisabledSchedules(m.disabledSchedules) //nolint:errcheck
		}
		m.schedView.Entries = m.buildScheduleEntries()

	case messages.EditCronMessage:
		local, err := config.LoadLocal()
		if err == nil {
			if _, ok := local.Schedules[msg.Name]; ok {
				s := local.Schedules[msg.Name]
				s.Cron = msg.Cron
				local.Schedules[msg.Name] = s
			} else if gs, ok := m.cfg.Schedules[msg.Name]; ok {
				local.Schedules[msg.Name] = config.Schedule{Task: gs.Task, Cron: msg.Cron}
			}
			if err := config.SaveLocal(local); err == nil {
				if newCfg, err := config.Load(); err == nil {
					m.cfg = newCfg
				}
			}
		}
		if m.sched != nil {
			wasEnabled := m.sched.IsEnabled(msg.Name)
			m.sched.UpdateSpec(msg.Name, msg.Cron)
			m.sched.Disable(msg.Name)
			if wasEnabled {
				m.sched.Enable(msg.Name) //nolint:errcheck
			}
		}
		m.schedView.Entries = m.buildScheduleEntries()
	}
	return m, nil
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}
	var content string
	switch m.activeTab {
	case TabTasks:
		content = m.taskView.View()
	case TabSchedule:
		content = m.schedView.View()
	// TODO: implement TabHistory view (views/history)
	// TODO: implement TabStats view (views/stats)
	// TODO: implement TabServices view (views/services)
	default:
		content = ""
	}
	full := m.renderTabBar() + "\n" + content + "\n" + m.renderStatusBar()
	if m.showHelp {
		return m.renderHelpOverlay(full)
	}
	return full
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.showHelp {
		switch msg.String() {
		case "?", "esc":
			m.showHelp = false
		}
		return m, nil
	}

	switch msg.String() {
	case "q", "ctrl+c":
		if m.pendingQuit {
			return m, tea.Quit
		}
		m.pendingQuit = true
		return m, nil
	case "tab":
		m.activeTab = (m.activeTab + 1) % 5
		return m, nil
	// TODO: add shift+tab for reverse tab cycling
	case "?":
		m.showHelp = !m.showHelp
		return m, nil
	default:
		m.pendingQuit = false
	}

	switch m.activeTab {
	case TabTasks:
		var cmd tea.Cmd
		m.taskView, cmd = m.taskView.Update(msg)
		return m, cmd
	case TabSchedule:
		var cmd tea.Cmd
		m.schedView, cmd = m.schedView.Update(msg)
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
		m.taskView.UpdateViewportForSelected()
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

func (m Model) buildScheduleEntries() []schedule.ScheduleEntry {
	var entries []schedule.ScheduleEntry

	var schedNames []string
	for name := range m.cfg.Schedules {
		schedNames = append(schedNames, name)
	}
	sort.Strings(schedNames)
	for _, name := range schedNames {
		s := m.cfg.Schedules[name]
		enabled := m.sched != nil && m.sched.IsEnabled(name)
		entries = append(entries, schedule.ScheduleEntry{
			Name:    name,
			Kind:    "cron",
			Cron:    s.Cron,
			Enabled: enabled,
		})
	}

	var watchNames []string
	for name, task := range m.cfg.Tasks {
		if len(task.Watch) > 0 {
			watchNames = append(watchNames, name)
		}
	}
	sort.Strings(watchNames)
	for _, name := range watchNames {
		task := m.cfg.Tasks[name]
		entries = append(entries, schedule.ScheduleEntry{
			Name:     name,
			Kind:     "watch",
			Patterns: task.Watch,
			Enabled:  !m.disabledSchedules["watch:"+name],
		})
	}

	return entries
}
