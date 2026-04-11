package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all keybindings for the TUI.
type KeyMap struct {
	Up             key.Binding
	Down           key.Binding
	ScrollUp       key.Binding
	ScrollDown     key.Binding
	Run            key.Binding
	Kill           key.Binding
	Clear          key.Binding
	Tab            key.Binding
	Filter         key.Binding
	Help           key.Binding
	Quit           key.Binding
	DeleteHistory  key.Binding
	ToggleSchedule key.Binding
	EditSchedule   key.Binding
	AddTask        key.Binding
}

// DefaultKeyMap returns the default keybindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		ScrollUp: key.NewBinding(
			key.WithKeys("pgup", "ctrl+u"),
			key.WithHelp("pgup/ctrl+u", "scroll log up"),
		),
		ScrollDown: key.NewBinding(
			key.WithKeys("pgdown", "ctrl+d"),
			key.WithHelp("pgdn/ctrl+d", "scroll log down"),
		),
		Run: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "run"),
		),
		Kill: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "kill"),
		),
		Clear: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "clear log"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "switch tab"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		DeleteHistory: key.NewBinding(
			key.WithKeys("D"),
			key.WithHelp("D", "clear history"),
		),
		ToggleSchedule: key.NewBinding(
			key.WithKeys("enter", " "),
			key.WithHelp("enter/space", "toggle schedule"),
		),
		EditSchedule: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit schedule"),
		),
		AddTask: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "add task"),
		),
	}
}
