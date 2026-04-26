package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/XenomorphingTV/burrow/internal/config"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/robfig/cron/v3"
)

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.String() == "esc" && m.servicesLogMode:
		m.servicesLogMode = false
		return m, nil

	case m.tab == TabServices && !m.servicesLogMode && msg.String() == "E":
		if svc, ok := m.selectedService(); ok {
			path := findUnitFilePath(svc)
			if path != "" {
				c := openInEditor(path, 0)
				return m, tea.ExecProcess(c, func(err error) tea.Msg { return nil })
			}
		}

	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Tab):
		m.tab = (m.tab + 1) % 5
		m.altScroll = 0
		m.filterInput = ""
		m.filterMode = false
		m.servicesLogMode = false
		m.servicesStatusMsg = ""
		if m.tab == TabStats {
			return m, loadAllHistory(m.st)
		}
		if m.tab == TabServices {
			return m, loadServicesCmd()
		}

	case key.Matches(msg, m.keys.Help):
		m.showHelp = !m.showHelp

	case key.Matches(msg, m.keys.Up):
		switch m.tab {
		case TabTasks:
			if m.selected > 0 {
				m.selected--
				m.updateViewportForSelected()
			}
		case TabSchedule:
			if m.scheduleSelected > 0 {
				m.scheduleSelected--
			}
		case TabHistory:
			if m.historySelected > 0 {
				m.historySelected--
				m.updateHistoryViewport()
			}
		case TabServices:
			if m.servicesLogMode {
				m.servicesViewport.LineUp(1)
			} else if m.servicesSelected > 0 {
				m.servicesSelected--
				if m.servicesSelected < m.servicesScroll {
					m.servicesScroll = m.servicesSelected
				}
			}
		}

	case key.Matches(msg, m.keys.Down):
		switch m.tab {
		case TabTasks:
			if m.selected < len(m.visibleItems())-1 {
				m.selected++
				m.updateViewportForSelected()
			}
		case TabSchedule:
			if m.scheduleSelected < len(m.scheduleTabEntries())-1 {
				m.scheduleSelected++
			}
		case TabHistory:
			if m.historySelected < len(m.filteredHistory())-1 {
				m.historySelected++
				m.updateHistoryViewport()
			}
		case TabServices:
			if m.servicesLogMode {
				m.servicesViewport.LineDown(1)
			} else if m.servicesSelected < len(m.filteredServices())-1 {
				m.servicesSelected++
				rowsAvail := m.height - 3 - 2
				if rowsAvail < 1 {
					rowsAvail = 1
				}
				if m.servicesSelected >= m.servicesScroll+rowsAvail {
					m.servicesScroll = m.servicesSelected - rowsAvail + 1
				}
			}
		}

	case key.Matches(msg, m.keys.Run):
		if m.tab == TabServices {
			return m, loadServicesCmd()
		}
		if m.tab == TabTasks {
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
			return m, m.startPipeline(name, "manual")
		}

	case key.Matches(msg, m.keys.Kill):
		if m.tab == TabTasks {
			name := m.selectedTaskName()
			if exec, ok := m.executors[name]; ok {
				exec.Kill()
			}
			if cancel, ok := m.watchers[name]; ok {
				cancel()
				delete(m.watchers, name)
				for i, t := range m.tasks {
					if t.Name == name && t.Status == StatusWatching {
						m.tasks[i].Status = StatusIdle
						break
					}
				}
			}
		}

	case key.Matches(msg, m.keys.Clear):
		if m.tab == TabTasks {
			name := m.selectedTaskName()
			m.taskLogs[name] = nil
			m.scrollLock = false
			m.viewport.SetContent("")
		}
		if m.tab == TabServices && !m.servicesLogMode {
			if svc, ok := m.selectedService(); ok {
				m.servicesLogMode = true
				m.servicesViewport.SetContent("loading...")
				return m, serviceLogsCmd(svc)
			}
		}

	case key.Matches(msg, m.keys.ScrollUp):
		switch m.tab {
		case TabTasks:
			m.scrollLock = true
			m.viewport.HalfViewUp()
		case TabSchedule, TabStats:
			step := max(1, m.height/4)
			m.altScroll -= step
			if m.altScroll < 0 {
				m.altScroll = 0
			}
		case TabHistory:
			m.historyViewport.HalfViewUp()
		case TabServices:
			if m.servicesLogMode {
				m.servicesViewport.HalfViewUp()
			} else {
				step := max(1, m.height/4)
				m.servicesScroll -= step
				if m.servicesScroll < 0 {
					m.servicesScroll = 0
				}
			}
		}

	case key.Matches(msg, m.keys.ScrollDown):
		switch m.tab {
		case TabTasks:
			m.viewport.HalfViewDown()
			if m.viewport.AtBottom() {
				m.scrollLock = false
			}
		case TabSchedule, TabStats:
			m.altScroll += max(1, m.height/4)
		case TabHistory:
			m.historyViewport.HalfViewDown()
		case TabServices:
			if m.servicesLogMode {
				m.servicesViewport.HalfViewDown()
			} else {
				m.servicesScroll += max(1, m.height/4)
			}
		}

	case key.Matches(msg, m.keys.AddTask):
		if m.tab == TabTasks {
			m.addTaskMode = true
			m.addTaskStep = 0
			m.addTaskInputs[0].Focus()
			m.recalcViewport()
		}

	case key.Matches(msg, m.keys.Filter):
		if m.tab != TabStats {
			m.filterMode = true
			m.filterInput = ""
		}

	case key.Matches(msg, m.keys.DeleteHistory):
		if m.tab == TabHistory && len(m.history) > 0 {
			m.confirmClearHistory = true
		}

	case key.Matches(msg, m.keys.ToggleSchedule):
		switch m.tab {
		case TabServices:
			if !m.servicesLogMode {
				if svc, ok := m.selectedService(); ok {
					action := "start"
					if strings.HasPrefix(svc.status, "active") {
						action = "stop"
					}
					return m, serviceActionCmd(svc, action)
				}
			}
		case TabTasks:
			items := m.visibleItems()
			if m.selected < len(items) && items[m.selected].isGroup {
				ns := items[m.selected].name
				if m.collapsedGroups[ns] {
					delete(m.collapsedGroups, ns)
				} else {
					m.collapsedGroups[ns] = true
				}
				// Clamp selection if items collapsed away beneath cursor.
				if newLen := len(m.visibleItems()); m.selected >= newLen {
					m.selected = max(0, newLen-1)
				}
			}
		case TabSchedule:
			entries := m.scheduleTabEntries()
			if m.scheduleSelected < len(entries) {
				e := entries[m.scheduleSelected]
				if e.kind == "cron" {
					if m.sched != nil {
						if m.sched.IsEnabled(e.name) {
							m.sched.Disable(e.name)
							m.disabledSchedules[e.name] = true
						} else {
							m.sched.Enable(e.name)
							delete(m.disabledSchedules, e.name)
						}
						if m.st != nil {
							m.st.SaveDisabledSchedules(m.disabledSchedules)
						}
					}
				} else {
					// Watch entry: toggle disabled state and sync watcher.
					key := "watch:" + e.name
					if m.disabledSchedules[key] {
						// Re-enable: remove from disabled, start watcher if task is done.
						delete(m.disabledSchedules, key)
						for i, t := range m.tasks {
							if t.Name == e.name && (t.Status == StatusSuccess || t.Status == StatusFailed) {
								ctx, cancel := context.WithCancel(context.Background())
								m.watchers[e.name] = cancel
								task := m.cfg.Tasks[e.name]
								baseDir := task.Cwd
								if baseDir == "" {
									baseDir = "."
								}
								m.tasks[i].Status = StatusWatching
								if m.st != nil {
									m.st.SaveDisabledSchedules(m.disabledSchedules)
								}
								return m, startWatcher(ctx, e.name, task.Watch, baseDir)
							}
						}
					} else {
						// Disable: cancel active watcher if any.
						m.disabledSchedules[key] = true
						if cancel, ok := m.watchers[e.name]; ok {
							cancel()
							delete(m.watchers, e.name)
							for i, t := range m.tasks {
								if t.Name == e.name && t.Status == StatusWatching {
									m.tasks[i].Status = StatusIdle
									break
								}
							}
						}
					}
					if m.st != nil {
						m.st.SaveDisabledSchedules(m.disabledSchedules)
					}
				}
			}
		}

	case key.Matches(msg, m.keys.EditSchedule):
		if m.tab == TabServices && !m.servicesLogMode {
			if svc, ok := m.selectedService(); ok {
				action := "enable"
				if svc.enabled == "enabled" {
					action = "disable"
				}
				return m, serviceActionCmd(svc, action)
			}
			return m, nil
		}
		if m.tab == TabTasks {
			name := m.selectedTaskName()
			if name == "" {
				break
			}
			path := configFileForTask(name)
			line := taskLineInFile(path, name)
			c := openInEditor(path, line)
			return m, tea.ExecProcess(c, func(err error) tea.Msg { return nil })
		}
		if m.tab == TabSchedule {
			entries := m.scheduleTabEntries()
			if m.scheduleSelected < len(entries) {
				e := entries[m.scheduleSelected]
				if e.kind == "cron" {
					m.scheduleEditMode = true
					m.scheduleEditName = e.name
					m.scheduleEditInput.SetValue(e.cron)
					m.scheduleEditInput.Focus()
					m.scheduleEditErr = ""
				} else {
					// Watch entry: open config file in $EDITOR at the task line.
					path := configFileForTask(e.name)
					line := taskLineInFile(path, e.name)
					c := openInEditor(path, line)
					return m, tea.ExecProcess(c, func(err error) tea.Msg { return nil })
				}
			}
		}
	}

	return m, nil
}

func (m Model) handleConfirmClear(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.confirmClearHistory = false
	if msg.String() == "y" || msg.String() == "Y" {
		if m.st != nil {
			m.st.ClearAll()
		}
		m.history = nil
		m.historySelected = 0
		m.updateHistoryViewport()
	}
	return m, nil
}

func (m Model) handleFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter":
		m.filterMode = false
		m.selected = 0
	case "ctrl+c":
		return m, tea.Quit
	case "backspace", "ctrl+h":
		runes := []rune(m.filterInput)
		if len(runes) > 0 {
			m.filterInput = string(runes[:len(runes)-1])
		}
	default:
		if len(msg.Runes) > 0 {
			m.filterInput += string(msg.Runes)
		}
	}
	return m, nil
}

func (m Model) handleScheduleEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.scheduleEditMode = false
		m.scheduleEditErr = ""
		return m, nil

	case "enter":
		spec := m.scheduleEditInput.Value()
		parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		if _, err := parser.Parse(spec); err != nil {
			m.scheduleEditErr = err.Error()
			return m, nil
		}

		local, err := config.LoadLocal()
		if err != nil {
			m.scheduleEditErr = "load config: " + err.Error()
			return m, nil
		}

		name := m.scheduleEditName
		if sched, ok := local.Schedules[name]; ok {
			sched.Cron = spec
			local.Schedules[name] = sched
		} else {
			if globalSched, ok := m.cfg.Schedules[name]; ok {
				if local.Schedules == nil {
					local.Schedules = make(map[string]config.Schedule)
				}
				local.Schedules[name] = config.Schedule{Task: globalSched.Task, Cron: spec}
			}
		}

		if err := config.SaveLocal(local); err != nil {
			m.scheduleEditErr = "save config: " + err.Error()
			return m, nil
		}

		if newCfg, err := config.Load(); err == nil {
			m.cfg = newCfg
		}

		if m.sched != nil {
			m.sched.UpdateSpec(name, spec)
			wasEnabled := m.sched.IsEnabled(name)
			m.sched.Disable(name)
			if wasEnabled {
				m.sched.Enable(name)
			}
		}

		m.scheduleEditMode = false
		m.scheduleEditErr = ""
		return m, nil

	default:
		var cmd tea.Cmd
		m.scheduleEditInput, cmd = m.scheduleEditInput.Update(msg)
		spec := m.scheduleEditInput.Value()
		if spec != "" {
			parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
			if _, err := parser.Parse(spec); err != nil {
				m.scheduleEditErr = err.Error()
			} else {
				m.scheduleEditErr = ""
			}
		} else {
			m.scheduleEditErr = ""
		}
		return m, cmd
	}
}

func (m Model) handleAddTask(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit

	case "esc":
		m.addTaskMode = false
		m.addTaskErr = ""
		for i := range m.addTaskInputs {
			m.addTaskInputs[i].SetValue("")
			m.addTaskInputs[i].Blur()
		}
		m.addTaskStep = 0
		m.recalcViewport()
		return m, nil

	case "tab", "enter":
		isLast := m.addTaskStep == len(m.addTaskInputs)-1
		if msg.String() == "enter" && isLast {
			return m.submitAddTask()
		}
		m.addTaskInputs[m.addTaskStep].Blur()
		m.addTaskStep = (m.addTaskStep + 1) % len(m.addTaskInputs)
		m.addTaskInputs[m.addTaskStep].Focus()
		return m, textinput.Blink

	case "shift+tab":
		m.addTaskInputs[m.addTaskStep].Blur()
		if m.addTaskStep > 0 {
			m.addTaskStep--
		} else {
			m.addTaskStep = len(m.addTaskInputs) - 1
		}
		m.addTaskInputs[m.addTaskStep].Focus()
		return m, textinput.Blink

	default:
		var cmd tea.Cmd
		m.addTaskInputs[m.addTaskStep], cmd = m.addTaskInputs[m.addTaskStep].Update(msg)
		return m, cmd
	}
}

// configFileForTask returns the absolute path of the config file that defines name.
// It prefers the local burrow.toml; falls back to the global tasks.toml.
func configFileForTask(name string) string {
	if local, err := config.LoadLocal(); err == nil {
		if _, ok := local.Tasks[name]; ok {
			if abs, err := filepath.Abs("burrow.toml"); err == nil {
				return abs
			}
			return "burrow.toml"
		}
	}
	return filepath.Join(config.DefaultConfigDir(), "tasks.toml")
}

// taskLineInFile scans path for the TOML section header that defines taskName
// and returns its 1-based line number, or 0 if not found.
func taskLineInFile(path, taskName string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}

	header := "[tasks." + taskName + "]"
	for i, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == header {
			return i + 1
		}
	}
	return 0
}

// openInEditor builds the exec.Cmd that opens path (at lineN, if > 0) in the
// editor named by $EDITOR (defaulting to vi).
func openInEditor(path string, lineN int) *exec.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	if lineN <= 0 {
		return exec.Command(editor, path)
	}

	base := filepath.Base(editor)
	switch base {
	case "hx", "helix", "code":
		//	file:N - hx (Helix), code (VS Code)
		return exec.Command(editor, fmt.Sprintf("%s:%d", path, lineN))
	default:
		//	+N file - vi, vim, nvim, nano, emacs, micro, kak
		return exec.Command(editor, fmt.Sprintf("+%d", lineN), path)
	}
}

func (m Model) submitAddTask() (tea.Model, tea.Cmd) {
	name := strings.TrimSpace(m.addTaskInputs[0].Value())
	cmdStr := strings.TrimSpace(m.addTaskInputs[1].Value())
	desc := strings.TrimSpace(m.addTaskInputs[2].Value())
	tagsStr := strings.TrimSpace(m.addTaskInputs[3].Value())
	cwd := strings.TrimSpace(m.addTaskInputs[4].Value())

	focusField := func(idx int) (tea.Model, tea.Cmd) {
		for i := range m.addTaskInputs {
			m.addTaskInputs[i].Blur()
		}
		m.addTaskStep = idx
		m.addTaskInputs[idx].Focus()
		return m, textinput.Blink
	}

	if name == "" {
		m.addTaskErr = "name is required"
		return focusField(0)
	}
	if strings.ContainsAny(name, " \t") {
		m.addTaskErr = "name must not contain spaces"
		return focusField(0)
	}
	if cmdStr == "" {
		m.addTaskErr = "cmd is required"
		return focusField(1)
	}
	if _, exists := m.cfg.Tasks[name]; exists {
		m.addTaskErr = fmt.Sprintf("task %q already exists", name)
		return focusField(0)
	}

	var tags []string
	for _, t := range strings.Split(tagsStr, ",") {
		if t = strings.TrimSpace(t); t != "" {
			tags = append(tags, t)
		}
	}

	task := config.Task{
		Cmd:         cmdStr,
		Description: desc,
		Cwd:         cwd,
		Tags:        tags,
	}

	global, err := config.LoadGlobal()
	if err != nil {
		m.addTaskErr = "load config: " + err.Error()
		return m, nil
	}
	global.Tasks[name] = task
	if err := config.SaveGlobal(global); err != nil {
		m.addTaskErr = "save config: " + err.Error()
		return m, nil
	}

	if newCfg, err := config.Load(); err == nil {
		m.cfg = newCfg
	}

	// Rebuild task list, preserving existing statuses
	prevStatuses := make(map[string]TaskEntry)
	for _, t := range m.tasks {
		prevStatuses[t.Name] = t
	}
	var taskNames []string
	for n := range m.cfg.Tasks {
		taskNames = append(taskNames, n)
	}
	sort.Strings(taskNames)
	tasks := make([]TaskEntry, len(taskNames))
	for i, n := range taskNames {
		entry := TaskEntry{Name: n, Cfg: m.cfg.Tasks[n], Status: StatusIdle}
		if prev, ok := prevStatuses[n]; ok {
			entry.Status = prev.Status
			entry.DurationMs = prev.DurationMs
			entry.ExitCode = prev.ExitCode
		}
		tasks[i] = entry
	}
	m.tasks = tasks

	for i, t := range m.tasks {
		if t.Name == name {
			m.selected = i
			break
		}
	}

	m.addTaskMode = false
	m.addTaskErr = ""
	for i := range m.addTaskInputs {
		m.addTaskInputs[i].SetValue("")
		m.addTaskInputs[i].Blur()
	}
	m.addTaskStep = 0
	m.recalcViewport()

	return m, nil
}

// initPromptStep configures the prompt widget for the current step and returns the updated model.
func (m Model) initPromptStep() Model {
	task, ok := m.cfg.Tasks[m.promptTaskName]
	if !ok || m.promptStep >= len(task.Inputs) {
		return m
	}
	inp := task.Inputs[m.promptStep]
	if len(inp.Options) > 0 {
		m.promptOptCursor = 0
		m.promptTextInput.Blur()
	} else {
		ti := textinput.New()
		ti.Placeholder = inp.Prompt
		ti.CharLimit = 256
		ti.Width = 40
		ti.Focus()
		m.promptTextInput = ti
		m.promptOptCursor = 0
	}
	return m
}

func (m Model) handlePromptKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	task, ok := m.cfg.Tasks[m.promptTaskName]
	if !ok {
		m.promptMode = false
		return m, nil
	}

	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit

	case "esc":
		m.promptMode = false
		m.promptValues = make(map[string]string)
		m.recalcViewport()
		return m, nil

	case "enter":
		return m.advancePromptStep()
	}

	if m.promptStep >= len(task.Inputs) {
		return m, nil
	}
	inp := task.Inputs[m.promptStep]

	if len(inp.Options) > 0 {
		switch msg.String() {
		case "up", "k":
			if m.promptOptCursor > 0 {
				m.promptOptCursor--
			}
		case "down", "j":
			if m.promptOptCursor < len(inp.Options)-1 {
				m.promptOptCursor++
			}
		}
		return m, nil
	}

	// Free-text: forward all other keys to the text input.
	var cmd tea.Cmd
	m.promptTextInput, cmd = m.promptTextInput.Update(msg)
	return m, cmd
}

func (m Model) advancePromptStep() (tea.Model, tea.Cmd) {
	task, ok := m.cfg.Tasks[m.promptTaskName]
	if !ok {
		m.promptMode = false
		return m, nil
	}
	if m.promptStep >= len(task.Inputs) {
		m.promptMode = false
		return m, nil
	}

	inp := task.Inputs[m.promptStep]
	var value string
	if len(inp.Options) > 0 {
		if m.promptOptCursor < len(inp.Options) {
			value = inp.Options[m.promptOptCursor]
		}
	} else {
		value = strings.TrimSpace(m.promptTextInput.Value())
	}
	m.promptValues[inp.Name] = value
	m.promptStep++

	if m.promptStep >= len(task.Inputs) {
		// All inputs collected — store and launch.
		collected := make(map[string]string, len(m.promptValues))
		for k, v := range m.promptValues {
			collected[k] = v
		}
		m.pendingEnv[m.promptTaskName] = collected
		m.promptMode = false
		m.promptValues = make(map[string]string)
		m.recalcViewport()
		return m, m.startPipeline(m.promptTaskName, "manual")
	}

	m = m.initPromptStep()
	m.recalcViewport()
	return m, textinput.Blink
}
