package tui

import (
	"fmt"
	"strings"

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
	}

	var parts []string
	for _, t := range labels {
		if m.tab == t.tab {
			parts = append(parts, StyleTabActive.Render(t.label))
		} else {
			parts = append(parts, StyleTabInactive.Render(t.label))
		}
	}
	bar := lipgloss.NewStyle().Background(lipgloss.Color(colorBgHeader)).Width(m.width).MaxWidth(m.width).Render(
		strings.Join(parts, ""),
	)
	sep := StyleLogDim.Render(strings.Repeat("─", m.width))
	return bar + "\n" + sep
}

func (m Model) renderStatusBar() string {
	if m.confirmClearHistory {
		prompt := StyleLogErr.Render("Clear all history?") +
			"  " + StyleKey.Render("y") + " confirm" +
			"  · " + StyleLogDim.Render("any other key") + " cancel"
		return StyleStatusBar.Width(m.width).Render(prompt)
	}

	keyHints := []string{
		StyleKey.Render("r") + " run",
		StyleKey.Render("x") + " kill",
		StyleKey.Render("l") + " clear",
		StyleKey.Render("a") + " add",
		StyleKey.Render("tab") + " switch",
		StyleKey.Render("/") + " filter",
		StyleKey.Render("?") + " help",
		StyleKey.Render("q") + " quit",
	}
	left := strings.Join(keyHints, "  ")

	running := 0
	idle := 0
	for _, t := range m.tasks {
		if t.Status == StatusRunning {
			running++
		} else {
			idle++
		}
	}
	right := StyleLogDim.Render(fmt.Sprintf("%d running · %d idle", running, idle))

	leftLen := lipgloss.Width(left)
	rightLen := lipgloss.Width(right)
	pad := m.width - leftLen - rightLen - 2
	if pad < 0 {
		pad = 0
	}

	return StyleStatusBar.Render(left + strings.Repeat(" ", pad) + right)
}

func (m Model) renderHelpOverlay(base string) string {
	help := `  Keybindings

  ↑ / k           navigate up
  ↓ / j           navigate down
  pgup / ctrl+u   scroll log up
  pgdn / ctrl+d   scroll log down
  r               run selected task
  x               kill selected task
  l               clear log view
  a               add new task
  enter / space   collapse/expand group
  tab             switch tab
  /               filter by name/tag
  esc             exit filter/add mode
  ?               toggle this help
  q / ctrl+c      quit`

	box := StyleHelp.Render(help)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

// renderScrollable renders a fixed header section followed by scrollable rows,
// clipped to exactly bodyHeight lines. altScroll is the row offset.
func renderScrollable(fixedLines, rows []string, altScroll, bodyHeight int) string {
	rowsAvail := bodyHeight - len(fixedLines)
	if rowsAvail < 0 {
		rowsAvail = 0
	}

	maxScroll := len(rows) - rowsAvail
	if maxScroll < 0 {
		maxScroll = 0
	}
	if altScroll > maxScroll {
		altScroll = maxScroll
	}

	end := altScroll + rowsAvail
	if end > len(rows) {
		end = len(rows)
	}
	visible := rows[altScroll:end]

	all := make([]string, 0, bodyHeight)
	all = append(all, fixedLines...)
	all = append(all, visible...)

	for len(all) < bodyHeight {
		all = append(all, "")
	}

	return strings.Join(all, "\n")
}
