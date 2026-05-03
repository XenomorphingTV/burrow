package history

import (
	"fmt"
	"strings"
	"time"

	"github.com/XenomorphingTV/burrow/internal/store"
	"github.com/charmbracelet/lipgloss"
)

const sidebarWidth = 24

func (m Model) filteredRecords() []*store.RunRecord {
	if m.FilterInput == "" {
		return m.Records
	}
	filter := strings.ToLower(m.FilterInput)
	var out []*store.RunRecord
	for _, r := range m.Records {
		if strings.Contains(strings.ToLower(r.TaskName), filter) {
			out = append(out, r)
		}
	}
	return out
}

func (m *Model) scrollToSelected() {
	records := m.filteredRecords()
	h := m.listHeight()
	if h < 1 {
		h = 1
	}

	if m.Selected < m.Scroll {
		m.Scroll = m.Selected
	}
	if m.Selected >= m.Scroll+h {
		m.Scroll = m.Selected - h + 1
	}

	// Nudge away from indicator slots.
	if m.Scroll > 0 && m.Selected == m.Scroll {
		m.Scroll--
	}
	end := m.Scroll + h
	if end < len(records) && m.Selected == end-1 {
		m.Scroll++
	}

	maxScroll := len(records) - h
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.Scroll > maxScroll {
		m.Scroll = maxScroll
	}
	if m.Scroll < 0 {
		m.Scroll = 0
	}
}

// listHeight is the number of sidebar rows available for records (excludes heading).
func (m Model) listHeight() int {
	h := m.Height - 1 // heading line
	if h < 0 {
		h = 0
	}
	return h
}

// RecalcViewport resizes and repopulates the main-pane viewport.
func (m *Model) RecalcViewport() {
	divW := lipgloss.Width(m.styles.LogDim.Render("│"))
	vpW := m.Width - sidebarWidth - divW
	if vpW < 10 {
		vpW = 10
	}
	// 3 header lines (name+badge, meta, separator)
	vpH := m.Height - 3
	if vpH < 1 {
		vpH = 1
	}
	m.viewport.Width = vpW
	m.viewport.Height = vpH
	m.updateViewport()
}

func (m *Model) updateViewport() {
	records := m.filteredRecords()
	if len(records) == 0 || m.Selected >= len(records) {
		m.viewport.SetContent("")
		return
	}
	r := records[m.Selected]
	lines := make([]string, len(r.LogTail))
	for i, line := range r.LogTail {
		lines[i] = m.renderLogLine(line)
	}
	m.viewport.SetContent(strings.Join(lines, "\n"))
	m.viewport.GotoTop()
}

// ── log line coloriser (ported from v1 render_log.go) ────────────────────────

type lineCategory int

const (
	catDefault lineCategory = iota
	catDim
	catOk
	catErr
	catWarn
	catInfo
)

type logPattern struct {
	prefixes []string
	cat      lineCategory
}

var logPatterns = []logPattern{
	{[]string{
		"[ok]", "[success]", "[pass]", "[passed]", "[done]", "[complete]",
		"ok ", "ok\t", "ok\n", "pass ", "pass\t", "✓", "✔",
		"--- pass", "=== pass",
	}, catOk},
	{[]string{
		"[error]", "[err]", "[fatal]", "[fail]", "[failed]", "[critical]",
		"error:", "err:", "fatal:", "failed:",
		"error ", "fatal ", "✗", "✘", "×",
		"--- fail", "fail\t", "fail ",
	}, catErr},
	{[]string{
		"[warn]", "[warning]", "[caution]",
		"warn:", "warning:", "warn ", "warning ", "⚠", "⚠️",
	}, catWarn},
	{[]string{
		"[info]", "[debug]", "[trace]", "[notice]", "[log]",
		"info:", "debug:", "trace:", "info ", "debug ",
		"-->", "->", "▶",
	}, catInfo},
}

func classifyLogLine(line string) (lineCategory, int) {
	if strings.HasPrefix(line, "$ ") {
		return catDim, 0
	}
	lower := strings.ToLower(line)
	for _, p := range logPatterns {
		for _, prefix := range p.prefixes {
			if strings.HasPrefix(lower, prefix) {
				return p.cat, len(prefix)
			}
		}
	}
	return catDefault, 0
}

func (m Model) catStyle(c lineCategory) lipgloss.Style {
	switch c {
	case catDim:
		return m.styles.LogDim
	case catOk:
		return m.styles.StatusOk
	case catErr:
		return m.styles.StatusFailed
	case catWarn:
		return m.styles.StatusWarn
	case catInfo:
		return m.styles.LogCmd
	default:
		return m.styles.LogText
	}
}

func (m Model) renderLogLine(line string) string {
	cat, prefixLen := classifyLogLine(line)
	st := m.catStyle(cat)
	if cat == catDim || prefixLen == 0 {
		return st.Render(line)
	}
	tag := line[:prefixLen]
	rest := line[prefixLen:]
	if rest == "" {
		return st.Render(tag)
	}
	return st.Render(tag) + m.styles.LogText.Render(rest)
}

// ── shared formatting helpers ─────────────────────────────────────────────────

func truncateStr(s string, max int) string {
	runes := []rune(s)
	if max <= 0 || len(runes) <= max {
		return s
	}
	if max <= 3 {
		return string(runes[:max])
	}
	return string(runes[:max-3]) + "..."
}

func fmtDuration(ms int64) string {
	switch {
	case ms < 1000:
		return fmt.Sprintf("%dms", ms)
	case ms < 60_000:
		return fmt.Sprintf("%.1fs", float64(ms)/1000)
	default:
		return fmt.Sprintf("%dm%ds", ms/60_000, (ms%60_000)/1000)
	}
}

func formatAge(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}
