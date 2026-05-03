package history

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	sidebar := m.renderSidebar()
	divider := m.styles.LogDim.Render("│")
	main := m.renderMainPane()
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, divider, main)
}

func (m Model) renderSidebar() string {
	var heading string
	switch {
	case m.FilterMode:
		heading = m.styles.SidebarHead.Width(sidebarWidth).Render("/" + m.FilterInput + "_")
	case m.FilterInput != "":
		heading = m.styles.SidebarHead.Width(sidebarWidth).Render("/" + m.FilterInput)
	default:
		heading = m.styles.SidebarHead.Width(sidebarWidth).Render("HISTORY")
	}

	records := m.filteredRecords()
	selBg := m.styles.TaskRowSelected.GetBackground()

	var rows []string
	if len(records) == 0 {
		rows = append(rows, m.styles.LogDim.Render("  no runs yet"))
	} else {
		nameW := lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.Text))

		for i, r := range records {
			isSelected := i == m.Selected

			var dot, prefix string
			switch r.Trigger {
			case "on_failure":
				if isSelected {
					dot = m.styles.StatusWarn.Background(selBg).Render("●")
					prefix = m.styles.StatusWarn.Background(selBg).Render("↳")
				} else {
					dot = m.styles.StatusWarn.Render("●")
					prefix = m.styles.StatusWarn.Render("↳")
				}
			case "on_success":
				if isSelected {
					dot = m.styles.StatusOk.Background(selBg).Render("●")
					prefix = m.styles.StatusOk.Background(selBg).Render("↳")
				} else {
					dot = m.styles.StatusOk.Render("●")
					prefix = m.styles.StatusOk.Render("↳")
				}
			default:
				if isSelected {
					prefix = m.styles.LogText.Background(selBg).Render(" ")
					if r.ExitCode == 0 {
						dot = m.styles.StatusOk.Background(selBg).Render("●")
					} else {
						dot = m.styles.StatusFailed.Background(selBg).Render("●")
					}
				} else {
					prefix = " "
					if r.ExitCode == 0 {
						dot = m.styles.StatusOk.Render("●")
					} else {
						dot = m.styles.StatusFailed.Render("●")
					}
				}
			}

			ageStr := formatAge(r.StartTime)
			// sidebarWidth - 2 padding - 3 fixed chars (prefix+dot+space) - age - 1 space
			nameMax := sidebarWidth - 2 - 3 - len(ageStr) - 1
			if nameMax < 0 {
				nameMax = 0
			}

			var name, age string
			if isSelected {
				name = nameW.Background(selBg).Render(truncateStr(r.TaskName, nameMax))
				age = m.styles.LogDim.Background(selBg).Render(ageStr)
			} else {
				name = nameW.Render(truncateStr(r.TaskName, nameMax))
				age = m.styles.LogDim.Render(ageStr)
			}

			inner := prefix + dot + " " + name + " " + age
			var row string
			if isSelected {
				row = m.styles.TaskRowSelected.Width(sidebarWidth).Render(inner)
			} else {
				row = m.styles.TaskRowNormal.Width(sidebarWidth).Render(inner)
			}
			rows = append(rows, row)
		}
	}

	h := m.listHeight()
	scroll := m.Scroll
	maxScroll := len(rows) - h
	if maxScroll < 0 {
		maxScroll = 0
	}
	if scroll > maxScroll {
		scroll = maxScroll
	}
	if scroll < 0 {
		scroll = 0
	}
	end := scroll + h
	if end > len(rows) {
		end = len(rows)
	}

	all := []string{heading}
	all = append(all, rows[scroll:end]...)

	for len(all) < m.Height {
		all = append(all, m.styles.TaskRowNormal.Width(sidebarWidth).Render(""))
	}

	// Scroll indicators overwrite the first/last content rows.
	if scroll > 0 && len(all) > 1 {
		all[1] = m.styles.LogDim.Width(sidebarWidth).Render(fmt.Sprintf("  ↑ %d more", scroll))
	}
	if end < len(rows) && len(all) == m.Height {
		all[m.Height-1] = m.styles.LogDim.Width(sidebarWidth).Render(fmt.Sprintf("  ↓ %d more", len(rows)-end))
	}

	// Confirm-clear prompt overlays the last line.
	if m.confirmClear && len(all) > 0 {
		all[len(all)-1] = m.styles.StatusWarn.Width(sidebarWidth).Render("  clear history? [y/n]")
	}

	return strings.Join(all[:m.Height], "\n")
}

func (m Model) renderMainPane() string {
	divW := lipgloss.Width(m.styles.LogDim.Render("│"))
	mainWidth := m.Width - sidebarWidth - divW

	records := m.filteredRecords()
	headerBg := m.styles.LogHead.GetBackground()
	bgStyle := lipgloss.NewStyle().Background(headerBg)

	if len(records) == 0 {
		empty := m.styles.LogDim.Render("  No history yet. Run a task!")
		return lipgloss.NewStyle().Width(mainWidth).Height(m.Height).Render(empty)
	}

	if m.Selected >= len(records) {
		return lipgloss.NewStyle().Width(mainWidth).Height(m.Height).Render("")
	}
	r := records[m.Selected]

	var resultBadge string
	if r.ExitCode == 0 {
		resultBadge = m.styles.StatusOk.Background(headerBg).Render("[ok]")
	} else {
		resultBadge = m.styles.StatusFailed.Background(headerBg).Render(fmt.Sprintf("[exit %d]", r.ExitCode))
	}

	var triggerStr string
	switch r.Trigger {
	case "on_failure":
		triggerStr = m.styles.StatusWarn.Render("↳ triggered by failure")
	case "on_success":
		triggerStr = m.styles.StatusOk.Render("↳ triggered by success")
	default:
		triggerStr = m.styles.LogDim.Render(r.Trigger)
	}

	meta := m.styles.LogDim.Render(formatAge(r.StartTime)+" · "+fmtDuration(r.DurationMs)+" · ") + triggerStr

	contentWidth := mainWidth - 2 // LogHead has Padding(0,1)
	taskName := truncateStr(r.TaskName, contentWidth-lipgloss.Width(resultBadge)-2)
	headLine1 := m.styles.LogHead.Width(mainWidth).Render(
		m.styles.LogName.Background(headerBg).Render(taskName) +
			bgStyle.Render("  ") +
			resultBadge,
	)
	headLine2 := m.styles.LogHead.Width(mainWidth).Render(meta)
	headLine3 := m.styles.LogDim.Width(mainWidth).Render(strings.Repeat("─", mainWidth))

	head := lipgloss.JoinVertical(lipgloss.Left, headLine1, headLine2, headLine3)
	return lipgloss.JoinVertical(lipgloss.Left, head, m.viewport.View())
}
