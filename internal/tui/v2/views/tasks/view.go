package tasks

import (
	"fmt"
	"strings"

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
	sidebarWidth := 24
	divider := m.styles.LogDim.Render("│")
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
	lines = append(lines, m.styles.SidebarHead.Width(width).Render("TASKS"))

	for i, task := range m.Tasks {
		row := m.statusDot(task.Status) + " " + task.Name
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
