package tui

import (
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/xenomorphingtv/burrow/internal/config"
	"github.com/xenomorphingtv/burrow/internal/runner"
	"github.com/xenomorphingtv/burrow/internal/store"
)

// TaskStatus represents the current state of a task.
type TaskStatus int

const (
	StatusIdle TaskStatus = iota
	StatusRunning
	StatusSuccess
	StatusFailed
)

// Tab represents which tab is currently active.
type Tab int

const (
	TabTasks Tab = iota
	TabSchedule
	TabHistory
	TabStats
)

// TaskEntry holds the display state for a single task.
type TaskEntry struct {
	Name       string
	Cfg        config.Task
	Status     TaskStatus
	DurationMs int64
	ExitCode   int
}

// ScheduledRunMsg is sent by the scheduler when a task should run.
type ScheduledRunMsg struct {
	TaskName string
	Trigger  string
}

// logLineMsg wraps a runner.LogLine as a tea.Msg.
type logLineMsg runner.LogLine

// historyLoadedMsg carries loaded history records.
type historyLoadedMsg []*store.RunRecord

// statsLoadedMsg carries all run records for the stats tab.
type statsLoadedMsg []*store.RunRecord

// tickMsg is sent periodically for spinner animation.
type tickMsg struct{}

// Model is the root Bubbletea model.
type Model struct {
	cfg   *config.Config
	st    store.Storer
	sched *runner.Scheduler
	pool  *runner.Pool

	tasks    []TaskEntry
	selected int
	tab      Tab

	taskLogs  map[string][]string
	executors map[string]*runner.Executor

	viewport   viewport.Model
	scrollLock bool // true when user has manually scrolled up; suppresses auto-scroll
	altScroll  int  // scroll offset for schedule tab

	historySelected int
	historyViewport viewport.Model
	filterInput     string
	filterMode      bool
	showHelp        bool

	scheduleSelected  int
	disabledSchedules map[string]bool

	scheduleEditMode  bool
	scheduleEditInput textinput.Model
	scheduleEditName  string
	scheduleEditErr   string

	addTaskMode   bool
	addTaskStep   int
	addTaskInputs []textinput.Model
	addTaskErr    string

	confirmClearHistory bool

	collapsedGroups map[string]bool
	onFailureRuns   map[string]bool // tracks tasks started as on_failure to prevent recursion
	pipelineQueues  map[string][]string

	width  int
	height int

	history      []*store.RunRecord
	statsRecords []*store.RunRecord
	keys         KeyMap

	tickCount int
}

// New creates a new TUI Model.
func New(cfg *config.Config, st store.Storer, sched *runner.Scheduler, pool *runner.Pool) Model {
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

	bgStyle := lipgloss.NewStyle().Background(lipgloss.Color(colorBg))

	vp := viewport.New(80, 20)
	vp.Style = bgStyle

	hvp := viewport.New(80, 20)
	hvp.Style = bgStyle

	ti := textinput.New()
	ti.Placeholder = "cron expression, e.g. 0 2 * * *"
	ti.CharLimit = 60
	ti.Width = 40

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

	disabledSchedules := make(map[string]bool)
	if st != nil {
		if loaded, err := st.LoadDisabledSchedules(); err == nil {
			disabledSchedules = loaded
		}
	}

	return Model{
		cfg:               cfg,
		st:                st,
		sched:             sched,
		pool:              pool,
		tasks:             tasks,
		tab:               TabTasks,
		taskLogs:          make(map[string][]string),
		executors:         make(map[string]*runner.Executor),
		viewport:          vp,
		historyViewport:   hvp,
		keys:              DefaultKeyMap(),
		disabledSchedules: disabledSchedules,
		collapsedGroups:   make(map[string]bool),
		onFailureRuns:     make(map[string]bool),
		pipelineQueues:    make(map[string][]string),
		scheduleEditInput: ti,
		addTaskInputs:     addInputs,
	}
}

// loadHistory is a command that loads recent history from the store.
func loadHistory(st store.Storer) tea.Cmd {
	return func() tea.Msg {
		if st == nil {
			return historyLoadedMsg(nil)
		}
		records, err := st.Recent(100)
		if err != nil {
			return historyLoadedMsg(nil)
		}
		return historyLoadedMsg(records)
	}
}

// loadAllHistory loads every run record for stats computation.
func loadAllHistory(st store.Storer) tea.Cmd {
	return func() tea.Msg {
		if st == nil {
			return statsLoadedMsg(nil)
		}
		records, err := st.Recent(0) // 0 = no limit
		if err != nil {
			return statsLoadedMsg(nil)
		}
		return statsLoadedMsg(records)
	}
}

// awaitLog blocks on a log channel and returns the next line as a tea.Msg.
func awaitLog(ch <-chan runner.LogLine) tea.Cmd {
	return func() tea.Msg {
		line, ok := <-ch
		if !ok {
			return logLineMsg{Done: true, ExitCode: -1}
		}
		return logLineMsg(line)
	}
}

// tick sends periodic tick messages for spinner animation.
func tick() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}

// Init is called once when the program starts.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		loadHistory(m.st),
		tick(),
	)
}

// Update handles incoming messages and returns the updated model and next command.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.recalcViewport()
		return m, nil

	case tea.KeyMsg:
		if m.confirmClearHistory {
			return m.handleConfirmClear(msg)
		}
		if m.scheduleEditMode {
			return m.handleScheduleEdit(msg)
		}
		if m.addTaskMode {
			return m.handleAddTask(msg)
		}
		if m.filterMode {
			return m.handleFilterKey(msg)
		}
		return m.handleKey(msg)

	case ScheduledRunMsg:
		return m, m.startPipeline(msg.TaskName, msg.Trigger)

	case logLineMsg:
		return m.handleLogLine(msg)

	case historyLoadedMsg:
		m.history = []*store.RunRecord(msg)
		if m.historySelected >= len(m.history) {
			m.historySelected = 0
		}
		m.updateHistoryViewport()
		return m, nil

	case statsLoadedMsg:
		m.statsRecords = []*store.RunRecord(msg)
		return m, nil

	case tickMsg:
		m.tickCount++
		return m, tick()
	}

	return m, nil
}

// View renders the full TUI.
func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	title := StyleHeader.Width(m.width).Render("  burrow")
	tabBar := m.renderTabBar()

	var body string
	switch m.tab {
	case TabTasks:
		body = m.renderTasksTab()
	case TabSchedule:
		body = m.renderScheduleTab()
	case TabHistory:
		body = m.renderHistoryTab()
	case TabStats:
		body = m.renderStatsTab()
	}

	statusBar := m.renderStatusBar()

	view := lipgloss.JoinVertical(lipgloss.Left,
		title,
		tabBar,
		body,
		statusBar,
	)

	if m.showHelp {
		view = m.renderHelpOverlay(view)
	}

	return view
}

func (m *Model) recalcViewport() {
	sidebarWidth := 24
	dividerWidth := 1
	mainWidth := m.width - sidebarWidth - dividerWidth - 2
	if mainWidth < 10 {
		mainWidth = 10
	}
	addPanelHeight := 0
	if m.addTaskMode {
		addPanelHeight = 9
	}
	// height - title(1) - tabbar(2) - loghead(3) - statusbar(1) - addPanel = height - 7 - addPanel
	vpHeight := m.height - 7 - addPanelHeight
	if vpHeight < 3 {
		vpHeight = 3
	}
	m.viewport.Width = mainWidth
	m.viewport.Height = vpHeight
	m.historyViewport.Width = mainWidth
	m.historyViewport.Height = vpHeight
}

func (m Model) handleLogLine(msg logLineMsg) (tea.Model, tea.Cmd) {
	name := msg.TaskName

	if msg.Done {
		var onFailure string
		for i, t := range m.tasks {
			if t.Name == name {
				if msg.ExitCode == 0 {
					m.tasks[i].Status = StatusSuccess
				} else {
					m.tasks[i].Status = StatusFailed
					onFailure = t.Cfg.OnFailure
				}
				m.tasks[i].ExitCode = msg.ExitCode
				break
			}
		}
		delete(m.executors, name)
		m.pool.Release()

		var cmds []tea.Cmd
		cmds = append(cmds, loadHistory(m.st))

		if msg.ExitCode == 0 {
			// Advance the pipeline if one is queued behind this task.
			if remaining, ok := m.pipelineQueues[name]; ok {
				delete(m.pipelineQueues, name)
				if len(remaining) > 0 {
					next := remaining[0]
					if len(remaining) > 1 {
						m.pipelineQueues[next] = remaining[1:]
					}
					cmds = append(cmds, m.startTask(next, "pipeline"))
				}
			}
		} else {
			// Clear any pending continuation on failure.
			delete(m.pipelineQueues, name)
		}

		if onFailure != "" && !m.onFailureRuns[name] {
			cmds = append(cmds, m.fireOnFailure(name, onFailure))
		}
		delete(m.onFailureRuns, name)
		return m, tea.Batch(cmds...)
	}

	rendered := renderLogLine(msg.Text)
	m.taskLogs[name] = append(m.taskLogs[name], rendered)

	if m.selectedTaskName() == name {
		m.updateViewportContent(name)
	}

	if exec, ok := m.executors[name]; ok {
		return m, awaitLog(exec.LogCh())
	}
	return m, nil
}

func (m *Model) updateViewportForSelected() {
	name := m.selectedTaskName()
	m.updateViewportContent(name)
}

func (m *Model) updateViewportContent(name string) {
	lines := m.taskLogs[name]
	m.viewport.SetContent(strings.Join(lines, "\n"))
	if !m.scrollLock {
		m.viewport.GotoBottom()
	}
}

// sidebarItem is one entry in the visible sidebar list — a group header or a task.
type sidebarItem struct {
	isGroup bool
	name    string // full task name ("db.seed") or namespace ("db")
}

// taskNamespace returns the namespace prefix of a dotted task name, or "".
func taskNamespace(name string) string {
	if i := strings.IndexByte(name, '.'); i >= 0 {
		return name[:i]
	}
	return ""
}

// taskLocalName returns the local part of a task name (after the dot, or the whole name).
func taskLocalName(name string) string {
	if i := strings.IndexByte(name, '.'); i >= 0 {
		return name[i+1:]
	}
	return name
}

// visibleItems returns the ordered sidebar list — group headers and tasks — respecting
// collapse state and the current filter. m.selected indexes into this slice.
func (m Model) visibleItems() []sidebarItem {
	filtered := m.filteredTasks()

	type groupData struct{ tasks []string }
	groups := make(map[string]*groupData)
	var ungrouped []string

	for _, t := range filtered {
		if ns := taskNamespace(t.Name); ns != "" {
			if groups[ns] == nil {
				groups[ns] = &groupData{}
			}
			groups[ns].tasks = append(groups[ns].tasks, t.Name)
		} else {
			ungrouped = append(ungrouped, t.Name)
		}
	}

	type entry struct {
		primary  string
		item     sidebarItem
		children []sidebarItem
	}
	var entries []entry

	for _, name := range ungrouped {
		entries = append(entries, entry{primary: name, item: sidebarItem{name: name}})
	}
	for ns, g := range groups {
		sort.Strings(g.tasks)
		var children []sidebarItem
		for _, name := range g.tasks {
			children = append(children, sidebarItem{name: name})
		}
		entries = append(entries, entry{
			primary:  ns,
			item:     sidebarItem{isGroup: true, name: ns},
			children: children,
		})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].primary < entries[j].primary })

	var items []sidebarItem
	for _, e := range entries {
		items = append(items, e.item)
		if e.item.isGroup && !m.collapsedGroups[e.item.name] {
			items = append(items, e.children...)
		}
	}
	return items
}

func (m Model) selectedTaskName() string {
	items := m.visibleItems()
	if len(items) == 0 || m.selected >= len(items) {
		return ""
	}
	item := items[m.selected]
	if item.isGroup {
		return ""
	}
	return item.name
}

func (m Model) filteredTasks() []TaskEntry {
	if m.filterInput == "" {
		return m.tasks
	}
	filter := strings.ToLower(m.filterInput)
	var out []TaskEntry
	for _, t := range m.tasks {
		if strings.Contains(strings.ToLower(t.Name), filter) {
			out = append(out, t)
			continue
		}
		for _, tag := range t.Cfg.Tags {
			if strings.Contains(strings.ToLower(tag), filter) {
				out = append(out, t)
				break
			}
		}
	}
	return out
}

func (m Model) sortedScheduleNames() []string {
	var names []string
	for n := range m.cfg.Schedules {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func (m *Model) updateHistoryViewport() {
	if len(m.history) == 0 || m.historySelected >= len(m.history) {
		m.historyViewport.SetContent("")
		return
	}
	r := m.history[m.historySelected]
	lines := make([]string, len(r.LogTail))
	for i, line := range r.LogTail {
		lines[i] = renderLogLine(line)
	}
	m.historyViewport.SetContent(strings.Join(lines, "\n"))
	m.historyViewport.GotoTop()
}

// startTask creates and starts an executor for the named task.
func (m Model) startTask(name, trigger string) tea.Cmd {
	if _, running := m.executors[name]; running {
		return nil
	}

	task, ok := m.cfg.Tasks[name]
	if !ok {
		return nil
	}

	// External tasks are launched in a terminal emulator — no output capture.
	if task.External {
		m.taskLogs[name] = nil
		m.scrollLock = false
		err := runner.LaunchExternal(task.Cmd, task.Cwd, m.cfg.Settings.Terminal, task.Env)
		var msg string
		if err != nil {
			msg = "[err] " + err.Error()
		} else {
			msg = "launched in external terminal"
		}
		m.taskLogs[name] = []string{"$ " + task.Cmd, msg}
		m.updateViewportForSelected()
		return nil
	}

	if !m.pool.TryAcquire() {
		return nil
	}

	exec := runner.NewExecutor(name, task, trigger, m.cfg.Settings.LogDir, m.st, m.cfg.Settings.Notify)
	exec.Start()
	m.executors[name] = exec

	for i, t := range m.tasks {
		if t.Name == name {
			m.tasks[i].Status = StatusRunning
			break
		}
	}

	m.taskLogs[name] = nil
	m.scrollLock = false

	return awaitLog(exec.LogCh())
}

// startPipeline resolves the full depends_on chain for target and runs each
// task in order, chaining them through pipelineQueues as each one completes.
func (m Model) startPipeline(target, trigger string) tea.Cmd {
	ordered, err := runner.Resolve(target, m.cfg.Tasks)
	if err != nil || len(ordered) == 0 {
		return nil
	}
	if len(ordered) == 1 {
		return m.startTask(ordered[0], trigger)
	}
	// Store the remaining chain. When ordered[0] finishes, handleLogLine will
	// start ordered[1], set pipelineQueues[ordered[1]] = ordered[2:], and so on.
	m.pipelineQueues[ordered[0]] = ordered[1:]
	return m.startTask(ordered[0], "pipeline")
}

// StartTaskExternal allows external code (scheduler) to start a task.
func (m *Model) StartTaskExternal(name, trigger string) tea.Cmd {
	return m.startTask(name, trigger)
}

// fireOnFailure runs the on_failure value for a failed task.
// If the value matches a known task name, that task is started via the normal
// executor path (with recursion guard). Otherwise the value is treated as a
// shell command and run in a fire-and-forget goroutine.
func (m Model) fireOnFailure(parentName, onFailure string) tea.Cmd {
	if _, ok := m.cfg.Tasks[onFailure]; ok {
		// Mark the target as an on_failure run so it won't recurse.
		m.onFailureRuns[onFailure] = true
		return m.startTask(onFailure, "on_failure")
	}
	// Inline shell command — run headlessly and record to history.
	return func() tea.Msg {
		task := config.Task{Cmd: onFailure}
		exec := runner.NewExecutor(parentName+".on_failure", task, "on_failure", m.cfg.Settings.LogDir, m.st, nil)
		exec.Start()
		for range exec.LogCh() {
		}
		return nil
	}
}
