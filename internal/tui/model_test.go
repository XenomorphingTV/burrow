package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xenomorphingtv/burrow/internal/config"
	"github.com/xenomorphingtv/burrow/internal/runner"
)

func newGroupedModel(t *testing.T) Model {
	t.Helper()
	cfg := &config.Config{
		Settings: config.Settings{MaxParallel: 2},
		Tasks: map[string]config.Task{
			"db.seed":    {Cmd: "node seed.js"},
			"db.migrate": {Cmd: "node migrate.js"},
			"deploy":     {Cmd: "deploy.sh"},
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

// navigation

func TestNavigateDownIncreasesSelected(t *testing.T) {
	m := newTestModel(t)
	before := m.selected
	m = sendKey(t, m, tea.KeyDown)
	if m.selected != before+1 {
		t.Errorf("expected selected=%d after down, got %d", before+1, m.selected)
	}
}

func TestNavigateUpDoesNotGoBelowZero(t *testing.T) {
	m := newTestModel(t)
	m.selected = 0
	m = sendKey(t, m, tea.KeyUp)
	if m.selected != 0 {
		t.Error("selected should not go below 0")
	}
}

func TestNavigateDownDoesNotExceedTaskCount(t *testing.T) {
	m := newTestModel(t)
	// Press down many more times than there are tasks.
	for i := 0; i < 20; i++ {
		m = sendKey(t, m, tea.KeyDown)
	}
	items := m.visibleItems()
	if m.selected >= len(items) {
		t.Errorf("selected %d out of bounds (len=%d)", m.selected, len(items))
	}
}

func TestJKNavigation(t *testing.T) {
	m := newTestModel(t)
	m.selected = 0

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = next.(Model)
	if m.selected != 1 {
		t.Errorf("j should move down: expected selected=1, got %d", m.selected)
	}

	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	m = next.(Model)
	if m.selected != 0 {
		t.Errorf("k should move up: expected selected=0, got %d", m.selected)
	}
}

// tab switching

func TestTabSwitching(t *testing.T) {
	m := newTestModel(t)
	if m.tab != TabTasks {
		t.Fatal("should start on TabTasks")
	}
	m = sendKey(t, m, tea.KeyTab)
	if m.tab != TabSchedule {
		t.Errorf("expected TabSchedule after one tab, got %d", m.tab)
	}
	m = sendKey(t, m, tea.KeyTab)
	if m.tab != TabHistory {
		t.Errorf("expected TabHistory after two tabs, got %d", m.tab)
	}
	m = sendKey(t, m, tea.KeyTab)
	if m.tab != TabStats {
		t.Errorf("expected TabStats after three tabs, got %d", m.tab)
	}
	m = sendKey(t, m, tea.KeyTab)
	if m.tab != TabTasks {
		t.Errorf("expected TabTasks after wrapping, got %d", m.tab)
	}
}

// filter mode

func TestFilterModeEntry(t *testing.T) {
	m := newTestModel(t)
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m = next.(Model)
	if !m.filterMode {
		t.Error("'/' should enter filter mode")
	}
}

func TestFilterModeTyping(t *testing.T) {
	m := newTestModel(t)
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m = next.(Model)

	for _, r := range "alp" {
		next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = next.(Model)
	}

	if m.filterInput != "alp" {
		t.Errorf("expected filterInput='alp', got %q", m.filterInput)
	}
}

func TestFilterModeFiltersTaskList(t *testing.T) {
	m := newTestModel(t) // has 'alpha' and 'beta'
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m = next.(Model)
	for _, r := range "alpha" {
		next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = next.(Model)
	}

	filtered := m.filteredTasks()
	if len(filtered) != 1 {
		t.Errorf("expected 1 result for 'alpha', got %d: %v", len(filtered), filtered)
	}
	if filtered[0].Name != "alpha" {
		t.Errorf("expected task 'alpha', got %q", filtered[0].Name)
	}
}

func TestFilterModeEscapeExits(t *testing.T) {
	m := newTestModel(t)
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m = next.(Model)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(Model)
	if m.filterMode {
		t.Error("esc should exit filter mode")
	}
}

func TestFilterModeBackspace(t *testing.T) {
	m := newTestModel(t)
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m = next.(Model)
	for _, r := range "abc" {
		next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = next.(Model)
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = next.(Model)
	if m.filterInput != "ab" {
		t.Errorf("backspace should remove last char; got %q", m.filterInput)
	}
}

// help overlay

func TestHelpOverlayToggle(t *testing.T) {
	m := newTestModel(t)
	if m.showHelp {
		t.Fatal("help should be hidden initially")
	}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	m = next.(Model)
	if !m.showHelp {
		t.Error("'?' should show help overlay")
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	m = next.(Model)
	if m.showHelp {
		t.Error("second '?' should hide help overlay")
	}
}

// namespace grouping

func TestVisibleItemsGroupsNamespacedTasks(t *testing.T) {
	m := newGroupedModel(t)
	items := m.visibleItems()

	groupCount := 0
	for _, item := range items {
		if item.isGroup {
			groupCount++
		}
	}
	if groupCount == 0 {
		t.Error("expected at least one group header for 'db.*' tasks")
	}
}

func TestTaskNamespaceExtraction(t *testing.T) {
	cases := []struct{ name, wantNS, wantLocal string }{
		{"db.seed", "db", "seed"},
		{"db.migrate", "db", "migrate"},
		{"deploy", "", "deploy"},
	}
	for _, tc := range cases {
		ns := taskNamespace(tc.name)
		local := taskLocalName(tc.name)
		if ns != tc.wantNS {
			t.Errorf("taskNamespace(%q) = %q, want %q", tc.name, ns, tc.wantNS)
		}
		if local != tc.wantLocal {
			t.Errorf("taskLocalName(%q) = %q, want %q", tc.name, local, tc.wantLocal)
		}
	}
}

func TestGroupCollapseExpand(t *testing.T) {
	m := newGroupedModel(t)

	// Find the group header index.
	items := m.visibleItems()
	groupIdx := -1
	for i, item := range items {
		if item.isGroup && item.name == "db" {
			groupIdx = i
			break
		}
	}
	if groupIdx < 0 {
		t.Fatal("could not find 'db' group header")
	}

	countBefore := len(m.visibleItems())
	m.selected = groupIdx

	// Collapse with space.
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = next.(Model)
	countAfter := len(m.visibleItems())

	if countAfter >= countBefore {
		t.Errorf("collapsing group should reduce visible items (%d -> %d)", countBefore, countAfter)
	}
}

// selectedTaskName

func TestSelectedTaskNameReturnsCorrectTask(t *testing.T) {
	m := newTestModel(t)
	m.selected = 0
	name := m.selectedTaskName()
	if name == "" {
		t.Error("selectedTaskName should return a non-empty name")
	}
}

func TestSelectedTaskNameEmptyOnGroup(t *testing.T) {
	m := newGroupedModel(t)
	// Find and select the group header.
	for i, item := range m.visibleItems() {
		if item.isGroup {
			m.selected = i
			break
		}
	}
	if name := m.selectedTaskName(); name != "" {
		t.Errorf("selectedTaskName on group should return '', got %q", name)
	}
}

// View smoke test

func TestViewRendersWithoutPanic(t *testing.T) {
	m := newTestModel(t)
	out := m.View()
	if !strings.Contains(out, "burrow") {
		t.Error("View() output should contain 'burrow'")
	}
}

func TestViewRendersAllTabs(t *testing.T) {
	m := newTestModel(t)
	for _, tab := range []Tab{TabTasks, TabSchedule, TabHistory, TabStats} {
		m.tab = tab
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("View() panicked on tab %d: %v", tab, r)
				}
			}()
			m.View()
		}()
	}
}
