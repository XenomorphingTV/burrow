package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderHistoryTab() string {
	sidebarWidth := 24
	sidebar := m.renderHistorySidebar(sidebarWidth)
	mainPane := m.renderHistoryMainPane()
	divider := StyleLogDim.Render("│")
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, divider, mainPane)
}

func (m Model) renderHistorySidebar(width int) string {
	bodyHeight := m.height - 4
	rowsAvail := bodyHeight - 2 // header + hint line

	hintLine := StyleLogDim.Render("  ") +
		StyleKey.Render("D") + StyleLogDim.Render("  clear history")

	var heading string
	if m.filterMode {
		heading = StyleSidebarHead.Width(width).Render("/" + m.filterInput + "_")
	} else if m.filterInput != "" {
		heading = StyleSidebarHead.Width(width).Render("/" + m.filterInput)
	} else {
		heading = StyleSidebarHead.Width(width).Render("HISTORY")
	}

	history := m.filteredHistory()

	var lines []string
	lines = append(lines, heading)
	lines = append(lines, StyleTaskRowNormal.Width(width).Render(hintLine))

	if len(history) == 0 {
		lines = append(lines, StyleLogDim.Width(width).Render("  no runs yet"))
	} else {
		start := 0
		if m.historySelected >= rowsAvail {
			start = m.historySelected - rowsAvail + 1
		}

		for i := start; i < len(history) && len(lines) < bodyHeight; i++ {
			r := history[i]

			var dot, prefix string
			if r.Trigger == "on_failure" {
				dot = StyleStatusWarn.Render("●")
				prefix = StyleStatusWarn.Render("↳")
			} else if r.Trigger == "on_success" {
				dot = StyleStatusOk.Render("●")
				prefix = StyleStatusOk.Render("↳")
			} else {
				prefix = " "
				if r.ExitCode == 0 {
					dot = StyleStatusOk.Render("●")
				} else {
					dot = StyleStatusFailed.Render("●")
				}
			}

			ageStr := formatAge(r.StartTime)
			// Subtract 2 for Padding(0,1) on the row style, plus 4 for the
			// fixed characters (prefix + dot + two spaces).
			nameMax := width - 6 - len(ageStr)
			if nameMax < 0 {
				nameMax = 0
			}
			name := truncateStr(r.TaskName, nameMax)
			age := StyleLogDim.Render(ageStr)
			row := prefix + dot + " " + name + " " + age

			if i == m.historySelected {
				lines = append(lines, StyleTaskRowSelected.Width(width).Render(row))
			} else {
				lines = append(lines, StyleTaskRowNormal.Width(width).Render(row))
			}
		}
	}

	for len(lines) < bodyHeight {
		lines = append(lines, StyleTaskRowNormal.Width(width).Render(""))
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderHistoryMainPane() string {
	mainWidth := m.historyViewport.Width + 2

	history := m.filteredHistory()

	if len(history) == 0 {
		empty := StyleLogDim.Render("  No history yet. Run a task!")
		return lipgloss.NewStyle().Width(mainWidth).Height(m.height - 4).Render(empty)
	}

	if m.historySelected >= len(history) {
		return lipgloss.NewStyle().Width(mainWidth).Height(m.height - 4).Render("")
	}
	r := history[m.historySelected]

	var resultBadge string
	if r.ExitCode == 0 {
		resultBadge = StyleStatusOk.Render("[ok]")
	} else {
		resultBadge = StyleStatusFailed.Render(fmt.Sprintf("[exit %d]", r.ExitCode))
	}

	var triggerStr string
	if r.Trigger == "on_failure" {
		triggerStr = StyleStatusWarn.Render("↳ triggered by failure")
	} else if r.Trigger == "on_success" {
		triggerStr = StyleStatusOk.Render("↳ triggered by success")
	} else {
		triggerStr = StyleLogDim.Render(r.Trigger)
	}
	meta := StyleLogDim.Render(formatAge(r.StartTime)+" · "+fmtDuration(r.DurationMs)+" · ") + triggerStr
	// Truncate task name so it never wraps — wrapping adds lines and pushes the tab bar off screen.
	contentWidth := mainWidth - 2 // StyleLogHead has Padding(0,1)
	taskName := truncateStr(r.TaskName, contentWidth-lipgloss.Width(resultBadge)-2)
	headLine1 := StyleLogHead.Width(mainWidth).Render(StyleLogName.Render(taskName) + "  " + resultBadge)
	headLine2 := StyleLogHead.Width(mainWidth).Render(meta)
	headLine3 := StyleLogDim.Width(mainWidth).Render(strings.Repeat("─", mainWidth))

	head := lipgloss.JoinVertical(lipgloss.Left, headLine1, headLine2, headLine3)
	return lipgloss.JoinVertical(lipgloss.Left, head, m.historyViewport.View())
}
