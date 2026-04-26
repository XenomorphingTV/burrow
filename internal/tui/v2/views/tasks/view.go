package tasks

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	sidebarWidth := 24
	divider := m.styles.LogDim.Render("│")

	sidebar := m.renderSidebar(sidebarWidth)
	mainPane := m.renderMainPane()

	return lipgloss.JoinHorizontal(lipgloss.Top,
		sidebar,
		divider,
		mainPane,
	)
}

func (m Model) renderSidebar(width int) string {
	var lines []string
	lines = append(lines, m.styles.SidebarHead.Width(width).Render("TASKS"))

	for i, t := range m.Tasks {
		row := m.statusDot(t.Status) + " " + t.Name
		if i == m.Selected {
			lines = append(lines, m.styles.TaskRowSelected.Width(width).Render(row))
		} else {
			lines = append(lines, m.styles.TaskRowNormal.Width(width).Render(row))
		}
	}

	for len(lines) < m.Height-3 {
		lines = append(lines, m.styles.TaskRowNormal.Width(width).Render(""))
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderMainPane() string {
	name := m.selectedTaskName()
	return m.styles.LogHead.Render(name)
}

func (m Model) statusDot(s TaskStatus) string {
	switch s {
	case StatusRunning:
		return m.styles.StatusRunning.Render("●")
	case StatusSuccess:
		return m.styles.StatusOk.Render("●")
	case StatusFailed:
		return m.styles.StatusFailed.Render("●")
	default:
		return m.styles.StatusIdle.Render("●")
	}
}
