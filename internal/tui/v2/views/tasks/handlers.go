package tasks

import (
	"fmt"
	"sort"
	"strings"

	"github.com/XenomorphingTV/burrow/internal/config"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handlePromptKey(msg tea.KeyMsg) (Model, tea.Cmd) {
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
		m.RecalcViewport()
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

func (m Model) handleAddTask(msg tea.KeyMsg) (Model, tea.Cmd) {
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
		m.RecalcViewport()
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

func (m Model) submitAddTask() (Model, tea.Cmd) {
	name := strings.TrimSpace(m.addTaskInputs[0].Value())
	cmdStr := strings.TrimSpace(m.addTaskInputs[1].Value())
	desc := strings.TrimSpace(m.addTaskInputs[2].Value())
	tagsStr := strings.TrimSpace(m.addTaskInputs[3].Value())
	cwd := strings.TrimSpace(m.addTaskInputs[4].Value())

	focusField := func(idx int) (Model, tea.Cmd) {
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
	for _, t := range m.Tasks {
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
	m.Tasks = tasks

	for i, t := range m.Tasks {
		if t.Name == name {
			m.Selected = i
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
	m.RecalcViewport()

	return m, nil
}

func (m Model) handleFilterKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter":
		m.FilterMode = false
		m.Selected = 0
	case "ctrl+c":
		return m, tea.Quit
	case "backspace", "ctrl+h":
		runes := []rune(m.FilterInput)
		if len(runes) > 0 {
			m.FilterInput = string(runes[:len(runes)-1])
		}
	default:
		if len(msg.Runes) > 0 {
			m.FilterInput += string(msg.Runes)
		}
	}
	return m, nil
}
