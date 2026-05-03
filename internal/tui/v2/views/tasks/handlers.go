package tasks

import (
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

func (m Model) handleFilterKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter":
		m.FilterMode = false
		m.Selected = 0
		m.sidebarScroll = 0
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
