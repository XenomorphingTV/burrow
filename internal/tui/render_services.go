package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type serviceEntry struct {
	name        string
	status      string // "active(running)", "inactive(dead)", "failed(failed)"
	enabled     string // "enabled", "disabled", "static", "masked"
	description string
	user        bool // true = --user scope
}

func (m Model) filteredServices() []serviceEntry {
	if m.filterInput == "" {
		return m.services
	}
	filter := strings.ToLower(m.filterInput)
	var out []serviceEntry
	for _, s := range m.services {
		if strings.Contains(strings.ToLower(s.name), filter) ||
			strings.Contains(strings.ToLower(s.description), filter) {
			out = append(out, s)
		}
	}
	return out
}

func (m Model) selectedService() (serviceEntry, bool) {
	services := m.filteredServices()
	if len(services) == 0 || m.servicesSelected >= len(services) {
		return serviceEntry{}, false
	}
	return services[m.servicesSelected], true
}

func serviceStatusStyle(status string) lipgloss.Style {
	switch {
	case strings.HasPrefix(status, "active"):
		return StyleStatusOk
	case strings.HasPrefix(status, "failed"):
		return StyleStatusFailed
	case strings.HasPrefix(status, "activating"), strings.HasPrefix(status, "deactivating"):
		return StyleStatusRunning
	default:
		return StyleLogDim
	}
}

func (m Model) renderServicesTab() string {
	totalAvailableHeight := m.height - 5

	if m.servicesLogMode {
		return m.renderServiceLogs(totalAvailableHeight)
	}

	const colStatus = 18
	const colEnabled = 9
	const colScope = 7
	const colDesc = 28

	nameWidth := m.width - colStatus - colEnabled - colScope - colDesc - 2
	if nameWidth < 10 {
		nameWidth = 10
	}

	styleHdr := lipgloss.NewStyle().
		Foreground(lipgloss.Color(activeTheme.Dim)).
		Bold(true)

	renderRow := func(name, status, enabled, scope, desc string, nameStyle, rowStyle, statusStyle lipgloss.Style) string {
		n := nameStyle.Width(nameWidth).Render(truncateStr(name, nameWidth))
		c1 := statusStyle.Width(colStatus).Render(truncateStr(status, colStatus))
		c2 := rowStyle.Width(colEnabled).Render(truncateStr(enabled, colEnabled))
		c3 := rowStyle.Width(colScope).Render(truncateStr(scope, colScope))
		c4 := rowStyle.Width(colDesc).Render(truncateStr(desc, colDesc))
		return " " + lipgloss.JoinHorizontal(lipgloss.Top, n, c1, c2, c3, c4)
	}

	header := renderRow("SERVICE", "STATUS", "ENABLED", "SCOPE", "DESCRIPTION", styleHdr, styleHdr, styleHdr)
	sep := StyleLogDim.Render(strings.Repeat("─", m.width))

	services := m.filteredServices()

	var fixed []string
	fixed = append(fixed, header, sep)
	if m.servicesStatusMsg != "" {
		fixed = append(fixed, StyleLogDim.Render("  "+m.servicesStatusMsg))
		fixed = append(fixed, StyleLogDim.Render(strings.Repeat("─", m.width)))
	}

	var rows []string
	if len(services) == 0 {
		if len(m.services) == 0 {
			rows = append(rows, StyleLogDim.Render("  no services loaded — press r to refresh"))
		} else {
			rows = append(rows, StyleLogDim.Render("  no services match filter"))
		}
	} else {
		for i, svc := range services {
			scope := "system"
			if svc.user {
				scope = "user"
			}
			var nameStyle, rowStyle, statusStyle lipgloss.Style
			if i == m.servicesSelected {
				nameStyle = StyleTaskRowSelected
				rowStyle = StyleTaskRowSelected
				statusStyle = StyleTaskRowSelected
			} else {
				nameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(activeTheme.Blue))
				rowStyle = StyleLogDim
				statusStyle = serviceStatusStyle(svc.status)
			}
			rows = append(rows, renderRow(svc.name, svc.status, svc.enabled, scope, svc.description, nameStyle, rowStyle, statusStyle))
		}
	}

	return renderScrollable(fixed, rows, m.servicesScroll, totalAvailableHeight)
}

func (m Model) renderServiceLogs(bodyHeight int) string {
	svc, ok := m.selectedService()
	if !ok {
		return StyleLogDim.Render("  no service selected")
	}

	scope := "system"
	if svc.user {
		scope = "user"
	}

	header := StyleLogHead.Width(m.width).Render(
		"  " + StyleLogName.Render(svc.name) +
			"  " + StyleLogDim.Render("["+scope+"]") +
			"  " + StyleLogDim.Render("esc to return"),
	)
	sep := StyleLogDim.Render(strings.Repeat("─", m.width))

	return header + "\n" + sep + "\n" + m.servicesViewport.View()
}
