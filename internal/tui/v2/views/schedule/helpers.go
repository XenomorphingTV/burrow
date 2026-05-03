package schedule

import (
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

func (m Model) filteredEntries() []ScheduleEntry {
	if m.FilterInput == "" {
		return m.Entries
	}
	filter := strings.ToLower(m.FilterInput)
	var out []ScheduleEntry
	for _, e := range m.Entries {
		if strings.Contains(strings.ToLower(e.Name), filter) {
			out = append(out, e)
		}
	}
	return out
}

func (m *Model) scrollToSelected() {
	entries := m.filteredEntries()
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

	// Nudge away from indicator slots (same logic as tasks sidebar).
	if m.Scroll > 0 && m.Selected == m.Scroll {
		m.Scroll--
	}
	end := m.Scroll + h
	if end < len(entries) && m.Selected == end-1 {
		m.Scroll++
	}

	maxScroll := len(entries) - h
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

func (m Model) listHeight() int {
	h := m.Height - 1 // heading line
	if m.editMode {
		h -= editPanelHeight
	}
	if h < 0 {
		h = 0
	}
	return h
}

func validateCron(spec string) error {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	_, err := parser.Parse(spec)
	return err
}

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
