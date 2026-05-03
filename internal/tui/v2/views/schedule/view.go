package schedule

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const editPanelHeight = 7

func (m Model) View() string {
	if m.editMode {
		listH := m.Height - editPanelHeight
		if listH < 0 {
			listH = 0
		}
		return m.renderList(listH) + "\n" + m.renderEditPanel()
	}
	return m.renderList(m.Height)
}

func (m Model) renderList(height int) string {
	var heading string
	switch {
	case m.FilterMode:
		heading = m.styles.SidebarHead.Width(m.Width).Render("/" + m.FilterInput + "_")
	case m.FilterInput != "":
		heading = m.styles.SidebarHead.Width(m.Width).Render("/" + m.FilterInput)
	default:
		heading = m.styles.SidebarHead.Width(m.Width).Render("SCHEDULES")
	}

	fixedLines := []string{heading}

	entries := m.filteredEntries()
	var rows []string
	if len(entries) == 0 {
		rows = append(rows, m.styles.LogDim.Render("  No schedules or watch tasks configured."))
	} else {
		nameStyle := lipgloss.NewStyle().Width(22).Foreground(lipgloss.Color(m.theme.Text))
		cronBadge := lipgloss.NewStyle().Width(9).Foreground(lipgloss.Color(m.theme.Text))
		watchBadge := lipgloss.NewStyle().Width(9).Foreground(lipgloss.Color(m.theme.Yellow))
		cronExprStyle := lipgloss.NewStyle().Width(20).Foreground(lipgloss.Color(m.theme.Blue))
		selBg := lipgloss.Color(m.theme.Selected)

		for i, e := range entries {
			isSelected := i == m.Selected

			var dot, name, kindBadge, detail string
			if isSelected {
				if e.Enabled {
					dot = m.styles.StatusOk.Background(selBg).Render("●")
				} else {
					dot = m.styles.LogText.Background(selBg).Render("●")
				}
				name = nameStyle.Background(selBg).Render(e.Name)
				if e.Kind == "cron" {
					kindBadge = m.styles.LogText.Background(selBg).Width(9).Render("[cron]")
					detail = cronExprStyle.Background(selBg).Render(e.Cron) +
						m.styles.LogText.Background(selBg).Render(describeCron(e.Cron))
				} else {
					kindBadge = watchBadge.Background(selBg).Render("[watch]")
					detail = cronExprStyle.Background(selBg).Render(strings.Join(e.Patterns, ", "))
				}
			} else {
				if e.Enabled {
					dot = m.styles.StatusOk.Render("●")
				} else {
					dot = m.styles.LogDim.Render("●")
				}
				name = nameStyle.Render(e.Name)
				if e.Kind == "cron" {
					kindBadge = cronBadge.Render("[cron]")
					detail = cronExprStyle.Render(e.Cron) + m.styles.LogDim.Render(describeCron(e.Cron))
				} else {
					kindBadge = watchBadge.Render("[watch]")
					detail = cronExprStyle.Render(strings.Join(e.Patterns, ", "))
				}
			}

			inner := dot + " " + name + kindBadge + detail
			var row string
			if isSelected {
				row = m.styles.TaskRowSelected.Width(m.Width).Render(inner)
			} else {
				row = m.styles.TaskRowNormal.Width(m.Width).Render(inner)
			}
			rows = append(rows, row)
		}
	}

	rowsAvail := height - len(fixedLines)
	if rowsAvail < 0 {
		rowsAvail = 0
	}
	maxScroll := len(rows) - rowsAvail
	if maxScroll < 0 {
		maxScroll = 0
	}
	scroll := m.Scroll
	if scroll > maxScroll {
		scroll = maxScroll
	}
	if scroll < 0 {
		scroll = 0
	}
	end := scroll + rowsAvail
	if end > len(rows) {
		end = len(rows)
	}

	all := append(fixedLines, rows[scroll:end]...)
	for len(all) < height {
		all = append(all, m.styles.TaskRowNormal.Width(m.Width).Render(""))
	}

	if scroll > 0 && len(all) > 1 {
		all[1] = m.styles.LogDim.Width(m.Width).Render(fmt.Sprintf("  ↑ %d more", scroll))
	}
	if end < len(rows) && len(all) == height {
		all[height-1] = m.styles.LogDim.Width(m.Width).Render(fmt.Sprintf("  ↓ %d more", len(rows)-end))
	}

	return strings.Join(all[:height], "\n")
}

func (m Model) renderEditPanel() string {
	sep := m.styles.LogDim.Render("  " + strings.Repeat("─", m.Width-4))
	nameLabel := m.styles.LogName.Render("  edit: ") + m.styles.LogDim.Render(m.editName)
	inputLine := m.styles.LogDim.Render("  cron expression: ") + m.editInput.View()

	var previewLine string
	spec := m.editInput.Value()
	if m.editErr != "" {
		previewLine = "  " + m.styles.LogErr.Render(m.editErr)
	} else if spec != "" {
		previewLine = "  " + m.styles.LogDim.Render(describeCron(spec))
		if next := nextThreeRuns(spec); next != "" {
			previewLine += "  " + m.styles.LogDim.Render("next: "+next)
		}
	}

	saveHint := m.styles.LogDim.Render("  ") +
		m.styles.Key.Render("enter") + m.styles.LogDim.Render(" save  ·  ") +
		m.styles.Key.Render("esc") + m.styles.LogDim.Render(" cancel")

	lines := []string{sep, nameLabel, inputLine, previewLine, "", saveHint}
	for len(lines) < editPanelHeight {
		lines = append(lines, "")
	}
	return strings.Join(lines[:editPanelHeight], "\n")
}

