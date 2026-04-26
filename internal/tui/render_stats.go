package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/XenomorphingTV/burrow/internal/store"
	"github.com/charmbracelet/lipgloss"
)

type taskStat struct {
	name       string
	total      int
	successes  int
	runs7d     int
	totalDurMs int64
	lastRun    time.Time
}

func computeStats(records []*store.RunRecord) []*taskStat {
	cutoff := time.Now().AddDate(0, 0, -7)
	statsMap := make(map[string]*taskStat)

	for _, r := range records {
		s := statsMap[r.TaskName]
		if s == nil {
			s = &taskStat{name: r.TaskName}
			statsMap[r.TaskName] = s
		}
		s.total++
		if r.ExitCode == 0 {
			s.successes++
		}
		if r.StartTime.After(cutoff) {
			s.runs7d++
		}
		s.totalDurMs += r.DurationMs
		if r.StartTime.After(s.lastRun) {
			s.lastRun = r.StartTime
		}
	}

	out := make([]*taskStat, 0, len(statsMap))
	for _, s := range statsMap {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}

func (m Model) renderStatsTab() string {
	bodyHeight := m.height - 3

	// Column widths (fixed)
	const col7d = 5
	const colRate = 8
	const colDur = 9
	const colLast = 10
	const padding = 2 // left+right padding on name

	nameWidth := m.width - col7d - colRate - colDur - colLast - padding
	if nameWidth < 10 {
		nameWidth = 10
	}

	styleHdr := lipgloss.NewStyle().
		Foreground(lipgloss.Color(activeTheme.Dim)).
		Bold(true)

	renderRow := func(name, d7, rate, dur, last string, nameStyle lipgloss.Style) string {
		n := nameStyle.Width(nameWidth).Render(truncateStr(name, nameWidth))
		c1 := StyleLogDim.Width(col7d).Align(lipgloss.Right).Render(d7)
		c2 := lipgloss.NewStyle().Width(colRate).Align(lipgloss.Right).Render(rate)
		c3 := StyleLogDim.Width(colDur).Align(lipgloss.Right).Render(dur)
		c4 := StyleLogDim.Width(colLast).Align(lipgloss.Right).Render(last)
		return " " + lipgloss.JoinHorizontal(lipgloss.Top, n, c1, c2, c3, c4)
	}

	header := renderRow("TASK", "7d", "success", "avg dur", "last run", styleHdr)
	sep := StyleLogDim.Render(strings.Repeat("─", m.width))

	var rows []string

	if len(m.statsRecords) == 0 {
		rows = append(rows, StyleLogDim.Render("  no history yet"))
	} else {
		stats := computeStats(m.statsRecords)
		for _, s := range stats {
			nameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(activeTheme.Blue))

			d7 := fmt.Sprintf("%d", s.runs7d)

			var rateStr string
			var rateStyle lipgloss.Style
			if s.total == 0 {
				rateStr = "—"
				rateStyle = StyleLogDim
			} else {
				pct := 100 * s.successes / s.total
				rateStr = fmt.Sprintf("%d%%", pct)
				switch {
				case pct == 100:
					rateStyle = StyleStatusOk
				case pct >= 75:
					rateStyle = StyleLogOk
				case pct >= 50:
					rateStyle = StyleStatusWarn
				default:
					rateStyle = StyleStatusFailed
				}
			}

			var durStr string
			if s.total > 0 {
				durStr = fmtDuration(s.totalDurMs / int64(s.total))
			} else {
				durStr = "—"
			}

			var lastStr string
			if s.lastRun.IsZero() {
				lastStr = "—"
			} else {
				lastStr = formatAge(s.lastRun)
			}

			// Build row manually so the rate cell gets its own color
			nStr := nameStyle.Width(nameWidth).Render(truncateStr(s.name, nameWidth))
			c1 := StyleLogDim.Width(col7d).Align(lipgloss.Right).Render(d7)
			c2 := rateStyle.Width(colRate).Align(lipgloss.Right).Render(rateStr)
			c3 := StyleLogDim.Width(colDur).Align(lipgloss.Right).Render(durStr)
			c4 := StyleLogDim.Width(colLast).Align(lipgloss.Right).Render(lastStr)
			rows = append(rows, " "+lipgloss.JoinHorizontal(lipgloss.Top, nStr, c1, c2, c3, c4))
		}
	}

	lines := renderScrollable([]string{header, sep}, rows, m.altScroll, bodyHeight)
	return lines
}
