package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// lineCategory classifies how a log line should be colored.
type lineCategory int

const (
	catDefault lineCategory = iota
	catDim
	catOk
	catErr
	catWarn
	catInfo
)

// logPattern maps a set of lowercased prefixes to a line category.
type logPattern struct {
	prefixes []string
	cat      lineCategory
}

var logPatterns = []logPattern{
	// ok / pass / success
	{
		[]string{
			"[ok]", "[success]", "[pass]", "[passed]", "[done]", "[complete]",
			"ok ", "ok\t", "ok\n",
			"pass ", "pass\t",
			"✓", "✔",
			"--- pass", // go test
			"=== pass",
		},
		catOk,
	},
	// error / fatal / fail
	{
		[]string{
			"[error]", "[err]", "[fatal]", "[fail]", "[failed]", "[critical]",
			"error:", "err:", "fatal:", "failed:",
			"error ", "fatal ",
			"✗", "✘", "×",
			"--- fail", // go test
			"fail\t", "fail ",
		},
		catErr,
	},
	// warn / warning
	{
		[]string{
			"[warn]", "[warning]", "[caution]",
			"warn:", "warning:",
			"warn ", "warning ",
			"⚠", "⚠️",
		},
		catWarn,
	},
	// info / debug / trace
	{
		[]string{
			"[info]", "[debug]", "[trace]", "[notice]", "[log]",
			"info:", "debug:", "trace:",
			"info ", "debug ",
			"-->", "->", "▶",
		},
		catInfo,
	},
}

// classifyLogLine returns the category and the byte length of the matched
// prefix (0 if no pattern matched). The prefix length is used to split the
// tag from the rest of the line for inline coloring.
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

// catStyle maps a lineCategory to its lipgloss style.
func catStyle(c lineCategory) lipgloss.Style {
	switch c {
	case catDim:
		return StyleLogDim
	case catOk:
		return StyleLogOk
	case catErr:
		return StyleLogErr
	case catWarn:
		return StyleLogWarn
	case catInfo:
		return StyleLogInfo
	default:
		return StyleLogText
	}
}

// renderLogLine colorizes a log line: the matched tag uses the category color
// and the remainder of the line uses the normal text style.
func renderLogLine(line string) string {
	cat, prefixLen := classifyLogLine(line)
	style := catStyle(cat)

	if cat == catDim || prefixLen == 0 {
		return style.Render(line)
	}
	tag := line[:prefixLen]
	rest := line[prefixLen:]
	if rest == "" {
		return style.Render(tag)
	}
	return style.Render(tag) + StyleLogText.Render(rest)
}

// truncateStr truncates s to at most max runes, appending "..." if truncated.
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

// fmtDuration formats a millisecond duration as a human-readable string.
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

// formatAge returns a human-readable age string for a time.
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
