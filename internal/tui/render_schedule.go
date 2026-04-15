package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/robfig/cron/v3"
)

func (m Model) renderScheduleTab() string {
	bodyHeight := m.height - 4

	hintLine := StyleLogDim.Render("  ") +
		StyleKey.Render("enter") + StyleLogDim.Render("/") + StyleKey.Render("space") +
		StyleLogDim.Render("  toggle  ·  ") +
		StyleKey.Render("e") + StyleLogDim.Render("  edit")

	var heading string
	if m.filterMode {
		heading = StyleSectionTitle.Render("/" + m.filterInput + "_")
	} else if m.filterInput != "" {
		heading = StyleSectionTitle.Render("/" + m.filterInput)
	} else {
		heading = StyleSectionTitle.Render("Schedules")
	}

	fixedLines := []string{
		heading,
		hintLine,
		StyleLogDim.Render("  " + strings.Repeat("─", m.width-4)),
	}

	entries := m.scheduleTabEntries()

	var rows []string
	if len(entries) == 0 {
		rows = append(rows, StyleLogDim.Render("  No schedules or watch tasks configured."))
	} else {
		for i, e := range entries {
			var enabled bool
			if e.kind == "cron" {
				enabled = m.sched != nil && m.sched.IsEnabled(e.name)
			} else {
				enabled = !m.disabledSchedules["watch:"+e.name]
			}

			var dot string
			if enabled {
				dot = StyleStatusOk.Render("●")
			} else {
				dot = StyleLogDim.Render("●")
			}

			nameStr := lipgloss.NewStyle().Width(22).Foreground(lipgloss.Color(colorPurple)).Render(e.name)

			var kindBadge, detail string
			if e.kind == "cron" {
				kindBadge = lipgloss.NewStyle().Width(9).Foreground(lipgloss.Color(colorDim)).Render("[cron]")
				cronStr := lipgloss.NewStyle().Width(20).Foreground(lipgloss.Color(colorBlue)).Render(e.cron)
				detail = cronStr + StyleLogDim.Render(describeCron(e.cron))
			} else {
				kindBadge = lipgloss.NewStyle().Width(9).Foreground(lipgloss.Color(colorYellow)).Render("[watch]")
				detail = StyleLogDim.Render(strings.Join(e.patterns, ", "))
			}

			inner := dot + " " + nameStr + kindBadge + detail
			var row string
			if i == m.scheduleSelected {
				row = "  " + StyleTaskRowSelected.Width(m.width-2).Render(inner)
			} else {
				row = "  " + inner
			}
			rows = append(rows, row)
		}
	}

	if m.scheduleEditMode {
		editPanelHeight := 7
		listHeight := bodyHeight - editPanelHeight
		if listHeight < 0 {
			listHeight = 0
		}

		listPart := renderScrollable(fixedLines, rows, m.altScroll, listHeight)

		sep := StyleLogDim.Render("  " + strings.Repeat("─", m.width-4))
		nameLabel := StyleLogName.Render("  edit: ") + StyleLogDim.Render(m.scheduleEditName)
		inputLine := StyleLogDim.Render("  cron expression: ") + m.scheduleEditInput.View()

		var previewLine string
		spec := m.scheduleEditInput.Value()
		if m.scheduleEditErr != "" {
			previewLine = "  " + StyleLogErr.Render(m.scheduleEditErr)
		} else if spec != "" {
			desc := describeCron(spec)
			nextTimes := nextThreeRuns(spec)
			previewLine = "  " + StyleLogDim.Render(desc)
			if nextTimes != "" {
				previewLine += "  " + StyleLogDim.Render("next: "+nextTimes)
			}
		}

		saveHint := StyleLogDim.Render("  ") + StyleKey.Render("enter") +
			StyleLogDim.Render(" save  ·  ") + StyleKey.Render("esc") + StyleLogDim.Render(" cancel")

		editLines := []string{sep, nameLabel, inputLine, previewLine, "", saveHint}
		for len(editLines) < editPanelHeight {
			editLines = append(editLines, "")
		}
		editLines = editLines[:editPanelHeight]

		return listPart + "\n" + strings.Join(editLines, "\n")
	}

	return renderScrollable(fixedLines, rows, m.altScroll, bodyHeight)
}

// describeCron returns a human-readable description of a cron expression.
func describeCron(spec string) string {
	known := map[string]string{
		"* * * * *":    "every minute",
		"*/2 * * * *":  "every 2 minutes",
		"*/5 * * * *":  "every 5 minutes",
		"*/10 * * * *": "every 10 minutes",
		"*/15 * * * *": "every 15 minutes",
		"*/30 * * * *": "every 30 minutes",
		"0 * * * *":    "every hour",
		"0 */2 * * *":  "every 2 hours",
		"0 */6 * * *":  "every 6 hours",
		"0 0 * * *":    "daily at midnight",
		"0 6 * * *":    "daily at 6:00",
		"0 8 * * *":    "daily at 8:00",
		"0 9 * * *":    "daily at 9:00",
		"0 12 * * *":   "daily at noon",
		"0 18 * * *":   "daily at 18:00",
		"0 0 * * 1":    "weekly on Monday",
		"0 0 * * 0":    "weekly on Sunday",
		"0 0 1 * *":    "monthly on the 1st",
	}
	if desc, ok := known[spec]; ok {
		return desc
	}
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	s, err := parser.Parse(spec)
	if err != nil {
		return spec
	}
	t := s.Next(time.Now())
	return "next: " + t.Format("Mon Jan 2 15:04")
}

// nextThreeRuns returns a string of the next 3 run times for a cron spec.
func nextThreeRuns(spec string) string {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	s, err := parser.Parse(spec)
	if err != nil {
		return ""
	}
	var times []string
	t := time.Now()
	for i := 0; i < 3; i++ {
		t = s.Next(t)
		times = append(times, t.Format("Mon Jan 2 15:04"))
	}
	return strings.Join(times, ", ")
}
