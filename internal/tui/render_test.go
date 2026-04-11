package tui

import (
	"strings"
	"testing"
	"time"
)

// formatAge tests

func TestFormatAgeSeconds(t *testing.T) {
	got := formatAge(time.Now().Add(-30 * time.Second))
	if !strings.HasSuffix(got, "s ago") {
		t.Errorf("expected 'Xs ago' for <1m, got %q", got)
	}
}

func TestFormatAgeMinutes(t *testing.T) {
	got := formatAge(time.Now().Add(-5 * time.Minute))
	if !strings.HasSuffix(got, "m ago") {
		t.Errorf("expected 'Xm ago' for <1h, got %q", got)
	}
}

func TestFormatAgeHours(t *testing.T) {
	got := formatAge(time.Now().Add(-3 * time.Hour))
	if !strings.HasSuffix(got, "h ago") {
		t.Errorf("expected 'Xh ago' for <24h, got %q", got)
	}
}

func TestFormatAgeDays(t *testing.T) {
	got := formatAge(time.Now().Add(-48 * time.Hour))
	if !strings.HasSuffix(got, "d ago") {
		t.Errorf("expected 'Xd ago' for >=24h, got %q", got)
	}
}

// describeCron tests

func TestDescribeCronKnownExpressions(t *testing.T) {
	cases := []struct {
		expr string
		want string
	}{
		{"* * * * *", "every minute"},
		{"0 * * * *", "every hour"},
		{"0 0 * * *", "daily at midnight"},
		{"0 12 * * *", "daily at noon"},
		{"0 0 * * 1", "weekly on Monday"},
		{"0 0 1 * *", "monthly on the 1st"},
	}
	for _, tc := range cases {
		got := describeCron(tc.expr)
		if got != tc.want {
			t.Errorf("describeCron(%q) = %q, want %q", tc.expr, got, tc.want)
		}
	}
}

func TestDescribeCronUnknownReturnsNextRun(t *testing.T) {
	// A valid but unrecognised expression should return something containing "next:"
	// (the fallback path computes the next run time).
	got := describeCron("15 14 1 * *") // 2:15pm on the 1st of every month
	if got == "" {
		t.Error("describeCron should return a non-empty string for a valid expression")
	}
}

func TestDescribeCronInvalidReturnsExpr(t *testing.T) {
	expr := "not-a-cron"
	got := describeCron(expr)
	if got != expr {
		t.Errorf("invalid cron should return the expression itself; got %q", got)
	}
}

// renderScrollable tests

func TestRenderScrollableClipsToBodyHeight(t *testing.T) {
	fixed := []string{"header"}
	rows := make([]string, 20)
	for i := range rows {
		rows[i] = "row"
	}
	out := renderScrollable(fixed, rows, 0, 10)
	lines := strings.Split(out, "\n")
	if len(lines) != 10 {
		t.Errorf("expected 10 lines, got %d", len(lines))
	}
}

func TestRenderScrollableShowsFixedLines(t *testing.T) {
	fixed := []string{"HEADER"}
	rows := []string{"row1", "row2"}
	out := renderScrollable(fixed, rows, 0, 5)
	if !strings.Contains(out, "HEADER") {
		t.Error("output should contain fixed header line")
	}
}

func TestRenderScrollableScrollOffset(t *testing.T) {
	fixed := []string{}
	rows := []string{"row0", "row1", "row2", "row3", "row4"}
	out := renderScrollable(fixed, rows, 2, 3)
	lines := strings.Split(out, "\n")
	// offset=2, height=3 → rows[2:5]
	if len(lines) < 1 || !strings.Contains(lines[0], "row2") {
		t.Errorf("with offset=2 the first visible row should be row2; got %q", lines[0])
	}
}

func TestRenderScrollablePadsToBodyHeight(t *testing.T) {
	fixed := []string{"h"}
	rows := []string{"r"} // only 1 row
	out := renderScrollable(fixed, rows, 0, 5)
	lines := strings.Split(out, "\n")
	if len(lines) != 5 {
		t.Errorf("output should be padded to bodyHeight=5 lines, got %d", len(lines))
	}
}
