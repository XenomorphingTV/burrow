package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/robfig/cron/v3"
	"github.com/xenomorphingtv/burrow/internal/config"
)

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Tab):
		m.tab = (m.tab + 1) % 4
		m.altScroll = 0
		if m.tab == TabStats {
			return m, loadAllHistory(m.st)
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
		}

	case key.Matches(msg, m.keys.Down):
		switch m.tab {
		case TabTasks:
			if m.selected < len(m.visibleItems())-1 {
				m.selected++
				m.updateViewportForSelected()
			}
		case TabSchedule:
			schedNames := m.sortedScheduleNames()
			if m.scheduleSelected < len(schedNames)-1 {
				m.scheduleSelected++
			}
		case TabHistory:
			if m.historySelected < len(m.history)-1 {
				m.historySelected++
				m.updateHistoryViewport()
			}
		}

	case key.Matches(msg, m.keys.Run):
		if m.tab == TabTasks {
			name := m.selectedTaskName()
			if name != "" {
				return m, m.startTask(name, "manual")
			}
		}

	case key.Matches(msg, m.keys.Kill):
		if m.tab == TabTasks {
			name := m.selectedTaskName()
			if exec, ok := m.executors[name]; ok {
				exec.Kill()
			}
		}

	case key.Matches(msg, m.keys.Clear):
		if m.tab == TabTasks {
			name := m.selectedTaskName()
			m.taskLogs[name] = nil
			m.scrollLock = false
			m.viewport.SetContent("")
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
		}

	case key.Matches(msg, m.keys.AddTask):
		if m.tab == TabTasks {
			m.addTaskMode = true
			m.addTaskStep = 0
			m.addTaskInputs[0].Focus()
			m.recalcViewport()
		}

	case key.Matches(msg, m.keys.Filter):
		if m.tab == TabTasks {
			m.filterMode = true
			m.filterInput = ""
		}

	case key.Matches(msg, m.keys.DeleteHistory):
		if m.tab == TabHistory && len(m.history) > 0 {
			m.confirmClearHistory = true
		}

	case key.Matches(msg, m.keys.ToggleSchedule):
		switch m.tab {
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
			names := m.sortedScheduleNames()
			if m.scheduleSelected < len(names) {
				name := names[m.scheduleSelected]
				if m.sched != nil {
					if m.sched.IsEnabled(name) {
						m.sched.Disable(name)
						m.disabledSchedules[name] = true
					} else {
						m.sched.Enable(name)
						delete(m.disabledSchedules, name)
					}
					if m.st != nil {
						m.st.SaveDisabledSchedules(m.disabledSchedules)
					}
				}
			}
		}

	case key.Matches(msg, m.keys.EditSchedule):
		if m.tab == TabTasks {
			name := m.selectedTaskName()
			if name == "" {
				break
			}
			path := configFileForTask(name)
			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = "vi"
			}
			c := exec.Command(editor, path)
			return m, tea.ExecProcess(c, func(err error) tea.Msg { return nil })
		}
		if m.tab == TabSchedule {
			names := m.sortedScheduleNames()
			if m.scheduleSelected < len(names) {
				name := names[m.scheduleSelected]
				m.scheduleEditMode = true
				m.scheduleEditName = name
				if sched, ok := m.cfg.Schedules[name]; ok {
					m.scheduleEditInput.SetValue(sched.Cron)
				}
				m.scheduleEditInput.Focus()
				m.scheduleEditErr = ""
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

// configFileForTask returns the path of the config file that defines name.
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
