package tasks

import (
	"fmt"
	"strings"

	"github.com/XenomorphingTV/burrow/internal/config"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderAddTaskPanel() string {
	labels := []string{"name", "cmd", "description", "tags", "cwd"}

	separator := m.styles.LogDim.Render("  " + strings.Repeat("─", m.Width-4))
	title := m.styles.SectionTitle.Render("  add task") +
		m.styles.LogDim.Render("  (tab=next · shift+tab=prev · enter=save · esc=cancel · saves to global config)")

	var fieldLines []string
	for i, input := range m.addTaskInputs {
		label := fmt.Sprintf("  %-14s", labels[i]+":")
		var labelStr string
		if i == m.addTaskStep {
			labelStr = m.styles.ActivePanel.Render(label)
		} else {
			labelStr = m.styles.LogDim.Render(label)
		}
		fieldLines = append(fieldLines, labelStr+input.View())
	}

	var errLine string
	if m.addTaskErr != "" {
		errLine = "  " + m.styles.LogErr.Render(m.addTaskErr)
	}

	lines := []string{separator, title}
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

	separator := m.styles.LogDim.Render("  " + strings.Repeat("─", m.Width-4))
	title := m.styles.SectionTitle.Render(fmt.Sprintf("  %s inputs (%d/%d)", m.promptTaskName, m.promptStep+1, len(task.Inputs))) +
		m.styles.LogDim.Render("  (enter=confirm · esc=cancel)")

	var lines []string
	lines = append(lines, separator, title, "")

	if m.promptStep < len(task.Inputs) {
		input := task.Inputs[m.promptStep]
		if len(input.Options) > 0 {
			label := m.styles.LogDim.Render("  " + input.Prompt)
			lines = append(lines, label)
			maxVisible := m.promptPanelHeight() - 4
			if maxVisible < 1 {
				maxVisible = 1
			}
			start := 0
			if m.promptOptCursor >= maxVisible {
				start = m.promptOptCursor - maxVisible + 1
			}
			end := start + maxVisible
			if end > len(input.Options) {
				end = len(input.Options)
			}
			for i := start; i < end; i++ {
				option := input.Options[i]
				if i == m.promptOptCursor {
					lines = append(lines, m.styles.TaskRowSelected.Width(m.Width-4).Render("  > "+option))
				} else {
					lines = append(lines, m.styles.TaskRowNormal.Width(m.Width-4).Render("    "+option))
				}
			}
		} else {
			label := m.styles.ActivePanel.Render(fmt.Sprintf("  %-14s", input.Prompt+":"))
			lines = append(lines, label+m.promptTextInput.View())
		}
	}

	height := m.promptPanelHeight()
	for len(lines) < height {
		lines = append(lines, "")
	}
	return strings.Join(lines[:height], "\n")
}

func (m Model) View() string {
	divider := m.styles.LogDim.Render("│")
	sidebar := m.renderSidebar(m.SidebarWidth)
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
	switch {
	case m.FilterMode:
		lines = append(lines, m.styles.SidebarHead.Width(width).Render("/"+m.FilterInput+"_"))
	case m.FilterInput != "":
		lines = append(lines, m.styles.SidebarHead.Width(width).Render("/"+m.FilterInput))
	default:
		lines = append(lines, m.styles.SidebarHead.Width(width).Render("TASKS"))
	}

	groupCount := make(map[string]int)
	for _, t := range m.filteredTasks() {
		if ns := taskNamespace(t.Name); ns != "" {
			groupCount[ns]++
		}
	}

	entryByName := make(map[string]TaskEntry, len(m.Tasks))
	for _, t := range m.Tasks {
		entryByName[t.Name] = t
	}

	for i, item := range m.visibleItems() {
		isSelected := i == m.Selected

		if item.IsGroup {
			arrow := "▼ "
			label := item.Name
			if m.CollapsedGroups[item.Name] {
				arrow = "▶ "
				label = fmt.Sprintf("%s (%d)", item.Name, groupCount[item.Name])
			}
			if isSelected {
				lines = append(lines, m.styles.TaskRowSelected.Width(width).Render(arrow+label))
			} else {
				lines = append(lines, m.styles.GroupHeader.Width(width).Render(arrow+label))
			}
			continue
		}

		entry := entryByName[item.Name]
		prefix := ""
		if taskNamespace(item.Name) != "" {
			prefix = "  "
		}

		var suffix string
		switch entry.Status {
		case StatusRunning:
			frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
			suffix = m.styles.StatusRunning.Render(frames[m.TickCount%len(frames)])
		case StatusSuccess:
			suffix = m.styles.StatusOk.Render(fmt.Sprintf("%dms", entry.DurationMs))
		case StatusFailed:
			suffix = m.styles.StatusFailed.Render(fmt.Sprintf("x%d", entry.ExitCode))
		case StatusWatching:
			suffix = m.styles.StatusWarn.Render("[watching]")
		}

		row := prefix + m.statusDot(entry.Status) + " " + taskLocalName(item.Name)
		if suffix != "" {
			row += " " + suffix
		}

		if isSelected {
			lines = append(lines, m.styles.TaskRowSelected.Width(width).Render(row))
		} else {
			lines = append(lines, m.styles.TaskRowNormal.Width(width).Render(row))
		}
	}

	extraPanelHeight := 0
	if m.addTaskMode {
		extraPanelHeight = 9
	} else if m.promptMode {
		extraPanelHeight = m.promptPanelHeight()
	}
	sidebarHeight := m.Height - 3 - extraPanelHeight
	for len(lines) < sidebarHeight {
		lines = append(lines, m.styles.TaskRowNormal.Width(width).Render(""))
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderMainPane() string {
	name := m.selectedTaskName()

	var task config.Task
	var status TaskStatus
	for _, t := range m.Tasks {
		if t.Name == name {
			task = t.Cfg
			status = t.Status
			break
		}
	}

	// Bake the header background into badge styles so they don't bleed
	// the terminal default when nested inside LogHead's render.
	headerBg := m.styles.LogHead.GetBackground()
	var statusBadge string
	switch status {
	case StatusRunning:
		statusBadge = m.styles.StatusRunning.Background(headerBg).Render("[running]")
	case StatusSuccess:
		statusBadge = m.styles.StatusOk.Background(headerBg).Render("[ok]")
	case StatusFailed:
		statusBadge = m.styles.StatusFailed.Background(headerBg).Render("[failed]")
	case StatusWatching:
		statusBadge = m.styles.StatusWarn.Background(headerBg).Render("[watching]")
	default:
		statusBadge = m.styles.StatusIdle.Background(headerBg).Render("[idle]")
	}

	mainWidth := m.Viewport.Width
	// LogHead has Padding(0,1) so its text area is 2 chars narrower
	contentWidth := mainWidth - 2

	bgStyle := lipgloss.NewStyle().Background(headerBg)

	var scrollHint string
	if m.ScrollLock {
		scrollHint = bgStyle.Render("  ") + m.styles.LogDim.Background(headerBg).Render("↑ scrolled · pgdn to follow")
	}

	headLine1 := m.styles.LogHead.Width(mainWidth).Render(
		m.styles.LogName.Background(headerBg).Render(name) +
			bgStyle.Render("  ") +
			statusBadge +
			scrollHint,
	)
	// Truncate so description never wraps — extra lines push the tab bar off screen
	headLine2 := m.styles.LogHead.Width(mainWidth).Render(
		m.styles.LogCmd.Background(headerBg).Render(truncateStr(task.Description, contentWidth)),
	)
	headLine3 := m.styles.LogDim.Width(mainWidth).Render(strings.Repeat("─", mainWidth))

	head := lipgloss.JoinVertical(lipgloss.Left, headLine1, headLine2, headLine3)
	return lipgloss.JoinVertical(lipgloss.Left, head, m.Viewport.View())
}

func (m Model) statusDot(status TaskStatus) string {
	switch status {
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
