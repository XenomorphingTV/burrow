package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type serviceEntry struct {
	name        string
	status      string
	enabled     string
	description string
}

func (m Model) renderServicesTab() string {
	bodyHeight := m.height - 4

	const colStatus = 10
	const colEnabled = 9
	const colDesc = 30

	nameWidth := m.width - colStatus - colEnabled - colDesc - 2
	if nameWidth < 10 {
		nameWidth = 10
	}

	styleHdr := lipgloss.NewStyle().
		Foreground(lipgloss.Color(activeTheme.Dim)).
		Bold(true)

	renderRow := func(name, status, enabled, desc string, nameStyle lipgloss.Style, rowStyle lipgloss.Style) string {
		n := nameStyle.Width(nameWidth).Render(truncateStr(name, nameWidth))
		c1 := rowStyle.Width(colStatus).Render(truncateStr(status, colStatus))
		c2 := rowStyle.Width(colEnabled).Render(truncateStr(enabled, colEnabled))
		c3 := rowStyle.Width(colDesc).Render(truncateStr(desc, colDesc))
		return " " + lipgloss.JoinHorizontal(lipgloss.Top, n, c1, c2, c3)
	}

	header := renderRow("SERVICE", "STATUS", "ENABLED", "DESCRIPTION", styleHdr, styleHdr)
	sep := StyleLogDim.Render(strings.Repeat("─", m.width))

	// Placeholder rows — real data will be populated once systemd integration is added.
	services := []serviceEntry{}

	var rows []string
	if len(services) == 0 {
		rows = append(rows, StyleLogDim.Render("  no services loaded"))
	} else {
		for i, svc := range services {
			var nameStyle, rowStyle lipgloss.Style
			if i == m.servicesSelected {
				nameStyle = StyleTaskRowSelected
				rowStyle = StyleTaskRowSelected
			} else {
				nameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(activeTheme.Blue))
				rowStyle = StyleLogDim
			}
			rows = append(rows, renderRow(svc.name, svc.status, svc.enabled, svc.description, nameStyle, rowStyle))
		}
	}

	lines := renderScrollable([]string{header, sep}, rows, m.servicesScroll, bodyHeight)
	return lines
}
