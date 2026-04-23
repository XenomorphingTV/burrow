package tui

import (
	"fmt"
	"strings"

	"github.com/XenomorphingTV/burrow/internal/config"
	"github.com/charmbracelet/lipgloss"
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
	if m.promptMode {
		return body + "\n" + m.renderPromptPanel()
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
		Foreground(lipgloss.Color(activeTheme.Blue)).Bold(true).Padding(0, 1)

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
		case StatusWatching:
			suffix = StyleStatusWarn.Render("[watching]")
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
	extraPanelHeight := 0
	if m.addTaskMode {
		extraPanelHeight = 9
	} else if m.promptMode {
		extraPanelHeight = m.promptPanelHeight()
	}
	sidebarHeight := m.height - 4 - extraPanelHeight
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
	case StatusWatching:
		return StyleStatusWarn.Render("●")
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
	case StatusWatching:
		statusBadge = StyleStatusWarn.Render("[watching]")
	default:
		statusBadge = StyleStatusIdle.Render("[idle]")
	}

	mainWidth := m.viewport.Width + 2
	// contentWidth is the usable area inside StyleLogHead's Padding(0,1)
	contentWidth := mainWidth - 2

	nameStr := StyleLogName.Render(name)

	var scrollHint string
	if m.scrollLock {
		scrollHint = "  " + StyleLogDim.Render("↑ scrolled · pgdn to follow")
	}

	headLine1 := StyleLogHead.Width(mainWidth).Render(nameStr + "  " + statusBadge + scrollHint)
	// Truncate description so it never wraps — wrapping adds lines and pushes the tab bar off screen
	desc := truncateStr(task.Description, contentWidth)
	headLine2 := StyleLogHead.Width(mainWidth).Render(StyleLogCmd.Render(desc))
	headLine3 := StyleLogDim.Width(mainWidth).Render(strings.Repeat("─", mainWidth))

	head := lipgloss.JoinVertical(lipgloss.Left, headLine1, headLine2, headLine3)

	return lipgloss.JoinVertical(lipgloss.Left, head, m.viewport.View())
}

func (m Model) renderAddTaskPanel() string {
	labels := []string{"name", "cmd", "description", "tags", "cwd"}

	sep := StyleLogDim.Render("  " + strings.Repeat("─", m.width-4))
	title := StyleSectionTitle.Render("  add task") +
		StyleLogDim.Render("  (tab=next · shift+tab=prev · enter=save · esc=cancel · saves to global config)")

	var fieldLines []string
	for i, input := range m.addTaskInputs {
		label := fmt.Sprintf("  %-14s", labels[i]+":")
		var labelStr string
		if i == m.addTaskStep {
			labelStr = lipgloss.NewStyle().Foreground(lipgloss.Color(activeTheme.Purple)).Bold(true).Render(label)
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

func (m Model) renderPromptPanel() string {
	task, ok := m.cfg.Tasks[m.promptTaskName]
	if !ok {
		return ""
	}

	sep := StyleLogDim.Render("  " + strings.Repeat("─", m.width-4))
	title := StyleSectionTitle.Render(fmt.Sprintf("  %s  inputs  (%d/%d)", m.promptTaskName, m.promptStep+1, len(task.Inputs))) +
		StyleLogDim.Render("  (enter=confirm · esc=cancel)")

	var lines []string
	lines = append(lines, sep, title, "")

	if m.promptStep < len(task.Inputs) {
		inp := task.Inputs[m.promptStep]
		if len(inp.Options) > 0 {
			label := StyleLogDim.Render("  " + inp.Prompt)
			lines = append(lines, label)
			// Cap visible options to avoid exceeding panel height
			maxVisible := m.promptPanelHeight() - 4
			if maxVisible < 1 {
				maxVisible = 1
			}
			start := 0
			if m.promptOptCursor >= maxVisible {
				start = m.promptOptCursor - maxVisible + 1
			}
			end := start + maxVisible
			if end > len(inp.Options) {
				end = len(inp.Options)
			}
			for i := start; i < end; i++ {
				opt := inp.Options[i]
				if i == m.promptOptCursor {
					lines = append(lines, StyleTaskRowSelected.Width(m.width-4).Render("  > "+opt))
				} else {
					lines = append(lines, StyleTaskRowNormal.Width(m.width-4).Render("    "+opt))
				}
			}
		} else {
			label := lipgloss.NewStyle().
				Foreground(lipgloss.Color(activeTheme.Purple)).Bold(true).
				Render(fmt.Sprintf("  %-14s", inp.Prompt+":"))
			lines = append(lines, label+m.promptTextInput.View())
		}
	}

	height := m.promptPanelHeight()
	for len(lines) < height {
		lines = append(lines, "")
	}
	return strings.Join(lines[:height], "\n")
}
