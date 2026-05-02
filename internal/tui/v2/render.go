package v2

import (
	"fmt"
	"strings"

	"github.com/XenomorphingTV/burrow/internal/tui/v2/views/tasks"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderTabBar() string {
	labels := []struct {
		label string
		tab   Tab
	}{
		{"tasks", TabTasks},
		{"schedule", TabSchedule},
		{"history", TabHistory},
		{"stats", TabStats},
		{"services", TabServices},
	}

	bg := lipgloss.Color(m.theme.BgHeader)
	var parts []string
	for _, t := range labels {
		if m.activeTab == t.tab {
			parts = append(parts, m.style.TabActive.Background(bg).Render(t.label))
		} else {
			parts = append(parts, m.style.TabInactive.Background(bg).Render(t.label))
		}
	}
	bar := lipgloss.NewStyle().
		Background(lipgloss.Color(m.theme.BgHeader)).
		Width(m.width).
		Render(strings.Join(parts, ""))
	sep := m.style.LogDim.Width(m.width).Render(strings.Repeat("─", m.width))
	return bar + "\n" + sep
}

func (m Model) renderStatusBar() string {
	sep := m.style.LogDim.Width(m.width).Render(strings.Repeat("─", m.width))
	if m.pendingQuit {
		hint := m.style.Key.Render("ctrl+c/q") + m.style.LogDim.Render(" again to quit")
		return sep + "\n" + m.style.StatusBar.Render(hint)
	}

	var keyHints []string
	switch m.activeTab {
	case TabTasks:
		keyHints = []string{
			m.style.Key.Render("r") + " run",
			m.style.Key.Render("x") + " kill",
			m.style.Key.Render("l") + " clear",
			m.style.Key.Render("e") + " edit",
			m.style.Key.Render("tab") + " switch",
			m.style.Key.Render("/") + " filter",
			m.style.Key.Render("?") + " help",
			m.style.Key.Render("q") + " quit",
		}
	default:
		keyHints = []string{
			m.style.Key.Render("tab") + " switch",
			m.style.Key.Render("q") + " quit",
		}
	}
	left := strings.Join(keyHints, "  ")

	running, idle := 0, 0
	for _, t := range m.taskView.Tasks {
		if t.Status == tasks.StatusRunning {
			running++
		} else {
			idle++
		}
	}
	right := m.style.LogDim.Render(fmt.Sprintf("%d running · %d idle", running, idle))

	leftLen := lipgloss.Width(left)
	rightLen := lipgloss.Width(right)
	pad := m.width - leftLen - rightLen - 2
	if pad < 0 {
		pad = 0
	}
	return sep + "\n" + m.style.StatusBar.Render(left+strings.Repeat(" ", pad)+right)
}

func (m Model) renderHelpOverlay(_ string) string {
	help := `  Keybindings

  ↑ / k           navigate up
  ↓ / j           navigate down
  pgup / ctrl+u   scroll up
  pgdn / ctrl+d   scroll down
  r               run selected task      (tasks tab)
  x               kill selected task
  l               clear log view         (tasks tab)
  e               open task in $EDITOR   (tasks tab)
  enter / space   collapse/expand group  (tasks tab)
  D               clear all run history  (history tab)
  tab             switch tab
  /               filter by name/tag
  esc             exit filter / add / prompt mode
  ?               toggle this help
  q / ctrl+c      quit`

	box := m.style.Help.Render(help)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}
