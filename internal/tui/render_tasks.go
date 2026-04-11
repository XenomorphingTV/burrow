package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/xenomorphingtv/burrow/internal/config"
)

func (m Model) renderTasksTab() string {
	sidebarWidth := 24
	divider := StyleLogDim.Render("│")

	sidebar := m.renderSidebar(sidebarWidth)
	mainPane := m.renderMainPane()

	body := lipgloss.JoinHorizontal(lipgloss.Top,
		sidebar,
		divider,
		mainPane,
	)
	if m.addTaskMode {
		return body + "\n" + m.renderAddTaskPanel()
	}
	return body
}

func (m Model) renderSidebar(width int) string {
	var lines []string

	// Header
	if m.filterMode {
		lines = append(lines, StyleSidebarHead.Width(width).Render("/"+m.filterInput+"_"))
	} else if m.filterInput != "" {
		lines = append(lines, StyleSidebarHead.Width(width).Render("/"+m.filterInput))
	} else {
		lines = append(lines, StyleSidebarHead.Width(width).Render("TASKS"))
	}

	// Count tasks per group (for collapsed label)
	groupCount := make(map[string]int)
	for _, t := range m.filteredTasks() {
		if ns := taskNamespace(t.Name); ns != "" {
			groupCount[ns]++
		}
	}

	// Build a lookup for task entries by name
	entryByName := make(map[string]TaskEntry, len(m.tasks))
	for _, t := range m.tasks {
		entryByName[t.Name] = t
	}

	items := m.visibleItems()
	styleGroup := lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorBlue)).Bold(true).Padding(0, 1)

	for i, item := range items {
		isSelected := i == m.selected

		if item.isGroup {
			collapsed := m.collapsedGroups[item.name]
			arrow := "▼ "
			label := item.name
			if collapsed {
				arrow = "▶ "
				label = fmt.Sprintf("%s (%d)", item.name, groupCount[item.name])
			}
			if isSelected {
				lines = append(lines, StyleTaskRowSelected.Width(width).Render(arrow+label))
			} else {
				lines = append(lines, styleGroup.Width(width).Render(arrow+label))
			}
			continue
		}

		entry := entryByName[item.name]
		ns := taskNamespace(item.name)
		localName := taskLocalName(item.name)
		prefix := ""
		if ns != "" {
			prefix = "  " // indent grouped tasks
		}

		dot := m.statusDot(entry.Status)

		var suffix string
		switch entry.Status {
		case StatusRunning:
			frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
			suffix = StyleStatusRunning.Render(frames[m.tickCount%len(frames)])
		case StatusSuccess:
			suffix = StyleStatusOk.Render(fmt.Sprintf("%dms", entry.DurationMs))
		case StatusFailed:
			suffix = StyleStatusFailed.Render(fmt.Sprintf("x%d", entry.ExitCode))
		}

		row := prefix + dot + " " + localName
		if suffix != "" {
			row += " " + suffix
		}

		if isSelected {
			lines = append(lines, StyleTaskRowSelected.Width(width).Render(row))
		} else {
			lines = append(lines, StyleTaskRowNormal.Width(width).Render(row))
		}
	}

	// Fill remaining height
	addPanelHeight := 0
	if m.addTaskMode {
		addPanelHeight = 9
	}
	sidebarHeight := m.height - 4 - addPanelHeight
	for len(lines) < sidebarHeight {
		lines = append(lines, StyleTaskRowNormal.Width(width).Render(""))
	}

	return strings.Join(lines, "\n")
}

func (m Model) statusDot(s TaskStatus) string {
	switch s {
	case StatusRunning:
		return StyleStatusRunning.Render("●")
	case StatusSuccess:
		return StyleStatusOk.Render("●")
	case StatusFailed:
		return StyleStatusFailed.Render("●")
	default:
		return StyleStatusIdle.Render("●")
	}
}

func (m Model) renderMainPane() string {
	name := m.selectedTaskName()
	var task config.Task
	var status TaskStatus
	if name != "" {
		for _, t := range m.tasks {
			if t.Name == name {
				task = t.Cfg
				status = t.Status
				break
			}
		}
	}

	var statusBadge string
	switch status {
	case StatusRunning:
		statusBadge = StyleStatusRunning.Render("[running]")
	case StatusSuccess:
		statusBadge = StyleStatusOk.Render("[ok]")
	case StatusFailed:
		statusBadge = StyleStatusFailed.Render("[failed]")
	default:
		statusBadge = StyleStatusIdle.Render("[idle]")
	}

	mainWidth := m.viewport.Width + 2
	// contentWidth is the usable area inside StyleLogHead's Padding(0,1)
	contentWidth := mainWidth - 2

	nameStr := StyleLogName.Render(name)
	// Truncate cmd so it never wraps — wrapping adds lines and pushes the tab bar off screen
	cmd := truncateStr(task.Cmd, contentWidth-lipgloss.Width(nameStr)-3)
	cmdStr := StyleLogCmd.Render(cmd)

	var scrollHint string
	if m.scrollLock {
		scrollHint = "  " + StyleLogDim.Render("↑ scrolled · pgdn to follow")
	}

	headLine1 := StyleLogHead.Width(mainWidth).Render(nameStr + "  " + statusBadge + scrollHint)
	headLine2 := StyleLogHead.Width(mainWidth).Render(cmdStr)
	headLine3 := StyleLogDim.Width(mainWidth).Render(strings.Repeat("─", mainWidth))

	head := lipgloss.JoinVertical(lipgloss.Left, headLine1, headLine2, headLine3)

	return lipgloss.JoinVertical(lipgloss.Left, head, m.viewport.View())
}

func (m Model) renderAddTaskPanel() string {
	labels := []string{"name", "cmd", "description", "tags", "cwd"}

	sep := StyleLogDim.Render("  " + strings.Repeat("─", m.width-4))
	title := StyleSectionTitle.Render("  add task") +
		StyleLogDim.Render("  (tab=next · enter=save · esc=cancel · saves to global config)")

	var fieldLines []string
	for i, input := range m.addTaskInputs {
		label := fmt.Sprintf("  %-14s", labels[i]+":")
		var labelStr string
		if i == m.addTaskStep {
			labelStr = lipgloss.NewStyle().Foreground(lipgloss.Color(colorPurple)).Bold(true).Render(label)
		} else {
			labelStr = StyleLogDim.Render(label)
		}
		fieldLines = append(fieldLines, labelStr+input.View())
	}

	var errLine string
	if m.addTaskErr != "" {
		errLine = "  " + StyleLogErr.Render(m.addTaskErr)
	}

	lines := []string{sep, title}
	lines = append(lines, fieldLines...)
	lines = append(lines, errLine)
	for len(lines) < 9 {
		lines = append(lines, "")
	}
	return strings.Join(lines[:9], "\n")
}
