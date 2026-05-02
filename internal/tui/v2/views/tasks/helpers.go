package tasks

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/XenomorphingTV/burrow/internal/config"
	"github.com/XenomorphingTV/burrow/internal/tui/v2/messages"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type SidebarItem struct {
	IsGroup bool
	Name    string
}

func taskNamespace(name string) string {
	if i := strings.IndexByte(name, '.'); i >= 0 {
		return name[:i]
	}
	return ""
}

func taskLocalName(name string) string {
	if i := strings.IndexByte(name, '.'); i >= 0 {
		return name[i+1:]
	}
	return name
}

func (m Model) selectedTaskName() string {
	items := m.visibleItems()
	if m.Selected < 0 || m.Selected >= len(items) {
		return ""
	}
	item := items[m.Selected]
	if item.IsGroup {
		return ""
	}
	return item.Name
}

func (m Model) visibleItems() []SidebarItem {
	filtered := m.filteredTasks()

	type groupData struct{ tasks []string }
	groups := make(map[string]*groupData)
	var ungrouped []string

	for _, task := range filtered {
		if namespace := taskNamespace(task.Name); namespace != "" {
			if groups[namespace] == nil {
				groups[namespace] = &groupData{}
			}
			groups[namespace].tasks = append(groups[namespace].tasks, task.Name)
		} else {
			ungrouped = append(ungrouped, task.Name)
		}
	}

	type entry struct {
		primary  string
		item     SidebarItem
		children []SidebarItem
	}
	var entries []entry

	for _, name := range ungrouped {
		entries = append(entries, entry{primary: name, item: SidebarItem{Name: name}})
	}

	for namespace, group := range groups {
		sort.Strings(group.tasks)
		var children []SidebarItem
		for _, name := range group.tasks {
			children = append(children, SidebarItem{Name: name})
		}
		entries = append(entries, entry{
			primary:  namespace,
			item:     SidebarItem{IsGroup: true, Name: namespace},
			children: children,
		})
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].primary < entries[j].primary })

	var items []SidebarItem
	for _, entry := range entries {
		items = append(items, entry.item)
		if entry.item.IsGroup && !m.CollapsedGroups[entry.item.Name] {
			items = append(items, entry.children...)
		}
	}

	return items
}

func (m Model) filteredTasks() []TaskEntry {
	if m.FilterInput == "" {
		return m.Tasks
	}

	filter := strings.ToLower(m.FilterInput)
	var out []TaskEntry
	for _, task := range m.Tasks {
		if strings.Contains(strings.ToLower(task.Name), filter) {
			out = append(out, task)
			continue
		}
		for _, tag := range task.Cfg.Tags {
			if strings.Contains(strings.ToLower(tag), filter) {
				out = append(out, task)
				break
			}
		}
	}
	return out
}

func (m Model) promptPanelHeight() int {
	if !m.promptMode || m.promptTaskName == "" {
		return 0
	}
	task, ok := m.cfg.Tasks[m.promptTaskName]
	if !ok || m.promptStep >= len(task.Inputs) {
		return 4
	}
	inp := task.Inputs[m.promptStep]
	if len(inp.Options) > 0 {
		height := 4 + len(inp.Options)
		if height > 12 {
			height = 12
		}
		return height
	}
	return 4
}

func truncateStr(str string, max int) string {
	runes := []rune(str)
	if max <= 0 || len(runes) <= max {
		return str
	}
	if max <= 3 {
		return string(runes[:max])
	}
	return string(runes[:max-3]) + "..."
}

// headHeight returns the number of lines the main pane header occupies:
// task name + status badge, description, and separator.
func (m Model) headHeight() int {
	return 3
}

// dividerWidth returns the visual column width of the sidebar divider.
func (m Model) dividerWidth() int {
	return lipgloss.Width(m.styles.LogDim.Render("│"))
}

func (m *Model) RecalcViewport() {
	extraPanelHeight := 0
	if m.promptMode {
		extraPanelHeight = m.promptPanelHeight()
	}

	vpWidth := m.Width - m.SidebarWidth - m.dividerWidth()
	if vpWidth < 10 {
		vpWidth = 10
	}

	vpHeight := m.Height - m.headHeight() - extraPanelHeight
	if vpHeight < 3 {
		vpHeight = 3
	}

	m.Viewport.Width = vpWidth
	m.Viewport.Height = vpHeight
}

func (m Model) initPromptStep() Model {
	task, ok := m.cfg.Tasks[m.promptTaskName]
	if !ok || m.promptStep >= len(task.Inputs) {
		return m
	}

	input := task.Inputs[m.promptStep]
	if len(input.Options) > 0 {
		m.promptOptCursor = 0
		m.promptTextInput.Blur()
	} else {
		textInput := textinput.New()
		textInput.Placeholder = input.Prompt
		textInput.CharLimit = 256
		textInput.Width = 40
		textInput.Focus()
		m.promptTextInput = textInput
		m.promptOptCursor = 0
	}
	return m
}

func (m Model) advancePromptStep() (Model, tea.Cmd) {
	task, ok := m.cfg.Tasks[m.promptTaskName]
	if !ok {
		m.promptMode = false
		return m, nil
	}
	if m.promptStep >= len(task.Inputs) {
		m.promptMode = false
		return m, nil
	}

	input := task.Inputs[m.promptStep]
	var value string
	if len(input.Options) > 0 {
		if m.promptOptCursor < len(input.Options) {
			value = input.Options[m.promptOptCursor]
		}
	} else {
		value = strings.TrimSpace(m.promptTextInput.Value())
	}
	m.promptValues[input.Name] = value
	m.promptStep++

	if m.promptStep >= len(task.Inputs) {
		collected := make(map[string]string, len(m.promptValues))
		for key, value := range m.promptValues {
			collected[key] = value
		}
		m.promptMode = false
		m.promptValues = make(map[string]string)
		m.RecalcViewport()
		return m, func() tea.Msg {
			return messages.RunTaskMessage{
				Name:    m.promptTaskName,
				Trigger: "manual",
				Env:     collected,
			}
		}
	}

	m = m.initPromptStep()
	m.RecalcViewport()
	return m, textinput.Blink
}

func configFileForTask(name string) string {
	if local, err := config.LoadLocal(); err == nil {
		if _, ok := local.Tasks[name]; ok {
			if abs, err := filepath.Abs("burrow.toml"); err == nil {
				return abs
			}
			return "burrow.toml"
		}
	}
	return filepath.Join(config.DefaultConfigDir(), "tasks.toml")
}

func taskLineInFile(path, taskName string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}

	header := "[tasks." + taskName + "]"
	for i, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == header {
			return i + 1
		}
	}
	return 0
}

// openInEditor builds the exec.Cmd that opens path (at lineN, if > 0) in the
// editor named by $EDITOR (defaulting to vi).
func openInEditor(path string, lineN int) *exec.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	if lineN <= 0 {
		return exec.Command(editor, path)
	}

	base := filepath.Base(editor)
	switch base {
	case "code":
		// VS Code requires --goto for file:line syntax; without it the colon is treated as part of the filename
		return exec.Command(editor, "--goto", fmt.Sprintf("%s:%d", path, lineN))
	case "hx", "helix":
		// file:N
		return exec.Command(editor, fmt.Sprintf("%s:%d", path, lineN))
	default:
		// +N file — vi, vim, nvim, nano, emacs, micro, kak
		return exec.Command(editor, fmt.Sprintf("+%d", lineN), path)
	}
}

func (m *Model) UpdateViewportForSelected() {
	name := m.selectedTaskName()
	lines := m.TaskLogs[name]
	m.Viewport.SetContent(strings.Join(lines, "\n"))
	if !m.ScrollLock {
		m.Viewport.GotoBottom()
	}
}

func (m *Model) UpdateViewportContent(name string) {
	lines := m.TaskLogs[name]
	m.Viewport.SetContent(strings.Join(lines, "\n"))
	if !m.ScrollLock {
		m.Viewport.GotoBottom()
	}
}
