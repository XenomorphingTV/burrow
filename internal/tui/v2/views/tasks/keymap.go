package tasks

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines key bindings for the tasks tab.
type KeyMap struct {
	Up         key.Binding
	Down       key.Binding
	ScrollUp   key.Binding
	ScrollDown key.Binding
	Run        key.Binding
	Kill       key.Binding
	Clear      key.Binding
	Edit       key.Binding
	Filter     key.Binding
	Toggle     key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up:         key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:       key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		ScrollUp:   key.NewBinding(key.WithKeys("pgup", "ctrl+u"), key.WithHelp("pgup/ctrl+u", "scroll up")),
		ScrollDown: key.NewBinding(key.WithKeys("pgdown", "ctrl+d"), key.WithHelp("pgdn/ctrl+d", "scroll down")),
		Run:        key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "run")),
		Kill:       key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "kill")),
		Clear:      key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "clear log")),
		Edit:       key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
		Filter:     key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		Toggle:     key.NewBinding(key.WithKeys("enter", " "), key.WithHelp("enter", "collapse/expand")),
	}
}
