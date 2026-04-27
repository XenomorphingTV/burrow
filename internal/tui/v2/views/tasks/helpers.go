package tasks

import (
	"sort"
	"strings"
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

func (m Model) selectedTaskName() string {
	if len(m.Tasks) == 0 || m.Selected >= len(m.Tasks) {
		return ""
	}
	return m.Tasks[m.Selected].Name
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

func (m *Model) recalcViewport() {
	const (
		sidebarWidth   = 24
		dividerWidth   = 1
		headingPadding = 2
		logHeadLines   = 3
	)

	extraPanelHeight := 0
	if m.addTaskMode {
		extraPanelHeight = 9
	} else if m.promptMode {
		extraPanelHeight = m.promptPanelHeight()
	}

	vpWidth := m.Width - sidebarWidth - dividerWidth - headingPadding
	if vpWidth < 10 {
		vpWidth = 10
	}

	vpHeight := m.Height - logHeadLines - extraPanelHeight
	if vpHeight < 3 {
		vpHeight = 3
	}

	m.Viewport.Width = vpWidth
	m.Viewport.Height = vpHeight
}
