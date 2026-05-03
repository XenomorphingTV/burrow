package history

import (
	"github.com/XenomorphingTV/burrow/internal/tui/v2/messages"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	records := m.filteredRecords()

	switch {
	case key.Matches(msg, m.keys.Cancel):
		m.confirmClear = false

	case key.Matches(msg, m.keys.Confirm):
		if m.confirmClear {
			m.confirmClear = false
			return m, func() tea.Msg { return messages.ClearHistoryMessage{} }
		}

	case key.Matches(msg, m.keys.Up):
		if !m.confirmClear && m.Selected > 0 {
			m.Selected--
			m.scrollToSelected()
			m.updateViewport()
		}

	case key.Matches(msg, m.keys.Down):
		if !m.confirmClear && m.Selected < len(records)-1 {
			m.Selected++
			m.scrollToSelected()
			m.updateViewport()
		}

	case key.Matches(msg, m.keys.ScrollUp):
		m.viewport.HalfViewUp()

	case key.Matches(msg, m.keys.ScrollDown):
		m.viewport.HalfViewDown()

	case key.Matches(msg, m.keys.Filter):
		if !m.confirmClear {
			m.FilterMode = true
			m.FilterInput = ""
		}

	case key.Matches(msg, m.keys.DeleteHistory):
		if len(records) > 0 {
			m.confirmClear = true
		}
	}

	return m, nil
}

func (m Model) handleFilterKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter":
		m.FilterMode = false
		m.Selected = 0
		m.Scroll = 0
		m.updateViewport()
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
