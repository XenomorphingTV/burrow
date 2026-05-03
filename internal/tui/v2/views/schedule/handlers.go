package schedule

import (
	"github.com/XenomorphingTV/burrow/internal/tui/v2/messages"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	entries := m.filteredEntries()
	switch {
	case key.Matches(msg, m.keys.Up):
		if m.Selected > 0 {
			m.Selected--
			m.scrollToSelected()
		}
	case key.Matches(msg, m.keys.Down):
		if m.Selected < len(entries)-1 {
			m.Selected++
			m.scrollToSelected()
		}
	case key.Matches(msg, m.keys.Filter):
		m.FilterMode = true
		m.FilterInput = ""
	case key.Matches(msg, m.keys.Toggle):
		if m.Selected < len(entries) {
			e := entries[m.Selected]
			return m, func() tea.Msg {
				return messages.ToggleScheduleMessage{Name: e.Name, Kind: e.Kind}
			}
		}
	case key.Matches(msg, m.keys.Edit):
		if m.Selected < len(entries) && entries[m.Selected].Kind == "cron" {
			e := entries[m.Selected]
			m.editMode = true
			m.editName = e.Name
			m.editInput.SetValue(e.Cron)
			m.editErr = ""
			return m, m.editInput.Focus()
		}
	}
	return m, nil
}

func (m Model) handleEditKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.editMode = false
		m.editErr = ""
		m.editInput.Blur()
		return m, nil
	case "enter":
		spec := m.editInput.Value()
		if err := validateCron(spec); err != nil {
			m.editErr = err.Error()
			return m, nil
		}
		name := m.editName
		m.editMode = false
		m.editErr = ""
		m.editInput.Blur()
		return m, func() tea.Msg {
			return messages.EditCronMessage{Name: name, Cron: spec}
		}
	}
	var cmd tea.Cmd
	m.editInput, cmd = m.editInput.Update(msg)
	if spec := m.editInput.Value(); spec != "" {
		if err := validateCron(spec); err != nil {
			m.editErr = err.Error()
		} else {
			m.editErr = ""
		}
	} else {
		m.editErr = ""
	}
	return m, cmd
}

func (m Model) handleFilterKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter":
		m.FilterMode = false
		m.Selected = 0
		m.Scroll = 0
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
