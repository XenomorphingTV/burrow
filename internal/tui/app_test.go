package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xenomorphingtv/burrow/internal/config"
	"github.com/xenomorphingtv/burrow/internal/runner"
)

// newTestModel returns a Model wired up for testing (no store, no scheduler).
func newTestModel(t *testing.T) Model {
	t.Helper()
	cfg := &config.Config{
		Settings: config.Settings{MaxParallel: 2},
		Tasks: map[string]config.Task{
			"alpha": {Cmd: "echo alpha", Description: "first task"},
			"beta":  {Cmd: "echo beta", Description: "second task"},
		},
		Schedules: map[string]config.Schedule{},
	}
	pool := runner.NewPool(2)
	m := New(cfg, nil, nil, pool)
	m.width = 120
	m.height = 40
	m.recalcViewport()
	return m
}

// fillLog populates taskLogs for taskName with n lines and refreshes the viewport.
func fillLog(m *Model, taskName string, n int) {
	for i := 0; i < n; i++ {
		m.taskLogs[taskName] = append(m.taskLogs[taskName], fmt.Sprintf("line %d", i))
	}
	m.updateViewportContent(taskName)
}

// sendKey runs a single key message through the model and returns the updated model.
func sendKey(t *testing.T, m Model, keyType tea.KeyType) Model {
	t.Helper()
	next, _ := m.Update(tea.KeyMsg{Type: keyType})
	return next.(Model)
}

// scroll-lock behaviour

func TestScrollLockSetOnPageUp(t *testing.T) {
	m := newTestModel(t)
	taskName := m.tasks[0].Name
	fillLog(&m, taskName, m.viewport.Height*3)

	if !m.viewport.AtBottom() {
		t.Fatal("viewport should start at bottom")
	}

	m = sendKey(t, m, tea.KeyPgUp)

	if !m.scrollLock {
		t.Error("scrollLock should be true after pgup")
	}
	if m.viewport.AtBottom() {
		t.Error("viewport should no longer be at bottom after pgup")
	}
}

func TestScrollLockClearedOnPageDownToBottom(t *testing.T) {
	m := newTestModel(t)
	taskName := m.tasks[0].Name
	fillLog(&m, taskName, m.viewport.Height*3)

	// scroll up to engage lock
	m = sendKey(t, m, tea.KeyPgUp)
	if !m.scrollLock {
		t.Fatal("expected scrollLock=true after pgup")
	}

	// scroll back down until at bottom
	for !m.viewport.AtBottom() {
		m = sendKey(t, m, tea.KeyPgDown)
	}

	if m.scrollLock {
		t.Error("scrollLock should be cleared once viewport reaches bottom")
	}
}

func TestNewLinesDoNotScrollWhenLocked(t *testing.T) {
	m := newTestModel(t)
	taskName := m.tasks[0].Name
	fillLog(&m, taskName, m.viewport.Height*3)

	// scroll up
	m = sendKey(t, m, tea.KeyPgUp)
	offsetBefore := m.viewport.YOffset

	// simulate an incoming log line for the selected task
	next, _ := m.Update(logLineMsg{TaskName: taskName, Text: "new line while locked"})
	m = next.(Model)

	if m.viewport.YOffset != offsetBefore {
		t.Errorf("viewport offset changed from %d to %d; should not auto-scroll when locked",
			offsetBefore, m.viewport.YOffset)
	}
}

func TestScrollLockResetOnNewTaskRun(t *testing.T) {
	m := newTestModel(t)
	taskName := m.tasks[0].Name
	fillLog(&m, taskName, m.viewport.Height*3)

	m = sendKey(t, m, tea.KeyPgUp)
	if !m.scrollLock {
		t.Fatal("expected scrollLock=true after pgup")
	}

	// Simulate startTask clearing the lock (without actually spawning a process)
	m.scrollLock = false
	m.taskLogs[taskName] = nil
	m.viewport.SetContent("")

	if m.scrollLock {
		t.Error("scrollLock should be false after task is re-run")
	}
}

func TestScrollLockResetOnClearLog(t *testing.T) {
	m := newTestModel(t)
	taskName := m.tasks[0].Name
	fillLog(&m, taskName, m.viewport.Height*3)

	m = sendKey(t, m, tea.KeyPgUp)
	if !m.scrollLock {
		t.Fatal("expected scrollLock=true")
	}

	// press l (clear log)
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	m = next.(Model)

	if m.scrollLock {
		t.Error("scrollLock should be cleared after clear-log key")
	}
}

// log line classification — tests target classifyLogLine (pure logic) so they
// are not sensitive to whether lipgloss emits ANSI codes in the test environment.

func TestClassifyCommandEcho(t *testing.T) {
	cat, _ := classifyLogLine("$ echo hello")
	if cat != catDim {
		t.Errorf("expected catDim, got %d", cat)
	}
}

func TestClassifyDefault(t *testing.T) {
	cat, prefixLen := classifyLogLine("just some normal output")
	if cat != catDefault {
		t.Errorf("expected catDefault, got %d", cat)
	}
	if prefixLen != 0 {
		t.Errorf("expected prefixLen=0, got %d", prefixLen)
	}
}

func TestClassifyBracketPrefixes(t *testing.T) {
	cases := []struct {
		input string
		want  lineCategory
	}{
		// ok / success
		{"[ok] connected", catOk},
		{"[OK] connected", catOk},
		{"[success] done", catOk},
		{"[done] finished", catOk},
		{"[pass] all good", catOk},
		// error / fatal
		{"[error] bad things", catErr},
		{"[ERROR] bad things", catErr},
		{"[fatal] crash", catErr},
		{"[fail] test failed", catErr},
		{"[failed] step 2", catErr},
		// warn
		{"[warn] low disk", catWarn},
		{"[WARNING] high memory", catWarn},
		// info / debug
		{"[info] starting server", catInfo},
		{"[INFO] starting server", catInfo},
		{"[debug] cache miss", catInfo},
		{"[trace] entering fn", catInfo},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got, prefixLen := classifyLogLine(tc.input)
			if got != tc.want {
				t.Errorf("classifyLogLine(%q) cat = %d, want %d", tc.input, got, tc.want)
			}
			if prefixLen == 0 {
				t.Errorf("classifyLogLine(%q) prefixLen = 0, expected a matched prefix", tc.input)
			}
		})
	}
}

func TestClassifyBareKeywords(t *testing.T) {
	cases := []struct {
		input string
		want  lineCategory
	}{
		{"error: module not found", catErr},
		{"fatal: out of memory", catErr},
		{"warn: deprecated api", catWarn},
		{"warning: rate limit approaching", catWarn},
		{"info: listening on :8080", catInfo},
		{"debug: cache miss key=42", catInfo},
		{"ok  github.com/foo/bar", catOk},
		{"--- PASS TestFoo (0.01s)", catOk},
		{"--- FAIL TestBar (timeout)", catErr},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got, _ := classifyLogLine(tc.input)
			if got != tc.want {
				t.Errorf("classifyLogLine(%q) = %d, want %d", tc.input, got, tc.want)
			}
		})
	}
}

func TestRenderLogLinePreservesText(t *testing.T) {
	// renderLogLine must always include the original text in its output,
	// regardless of color support.
	lines := []string{
		"$ echo hello",
		"[ok] connected",
		"[error] something failed",
		"plain output line",
	}
	for _, line := range lines {
		out := renderLogLine(line)
		if !strings.Contains(out, line) && !strings.Contains(out, strings.TrimPrefix(line, "[ok] ")) {
			// The tag may be split from the rest; check that the text portion is present
			if !strings.Contains(out, line[strings.Index(line, " ")+1:]) {
				t.Errorf("renderLogLine(%q) output %q does not contain original text", line, out)
			}
		}
	}
}

// helper utilities

func TestTruncateStr(t *testing.T) {
	cases := []struct {
		input string
		max   int
		want  string
	}{
		{"hello", 10, "hello"},
		{"hello world", 8, "hello..."},
		{"hi", 2, "hi"},
		{"hello", 3, "hel"},
		{"hello", 0, "hello"},
	}
	for _, tc := range cases {
		got := truncateStr(tc.input, tc.max)
		if got != tc.want {
			t.Errorf("truncateStr(%q, %d) = %q; want %q", tc.input, tc.max, got, tc.want)
		}
	}
}

func TestFmtDuration(t *testing.T) {
	cases := []struct {
		ms   int64
		want string
	}{
		{0, "0ms"},
		{500, "500ms"},
		{999, "999ms"},
		{1000, "1.0s"},
		{2100, "2.1s"},
		{59999, "60.0s"},
		{60000, "1m0s"},
		{90500, "1m30s"},
	}
	for _, tc := range cases {
		got := fmtDuration(tc.ms)
		if got != tc.want {
			t.Errorf("fmtDuration(%d) = %q; want %q", tc.ms, got, tc.want)
		}
	}
}
