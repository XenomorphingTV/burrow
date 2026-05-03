package history

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	Up            key.Binding
	Down          key.Binding
	ScrollUp      key.Binding
	ScrollDown    key.Binding
	Filter        key.Binding
	DeleteHistory key.Binding
	Confirm       key.Binding
	Cancel        key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up:            key.NewBinding(key.WithKeys("up", "k")),
		Down:          key.NewBinding(key.WithKeys("down", "j")),
		ScrollUp:      key.NewBinding(key.WithKeys("pgup", "ctrl+u")),
		ScrollDown:    key.NewBinding(key.WithKeys("pgdn", "ctrl+d")),
		Filter:        key.NewBinding(key.WithKeys("/")),
		DeleteHistory: key.NewBinding(key.WithKeys("D")),
		Confirm:       key.NewBinding(key.WithKeys("y")),
		Cancel:        key.NewBinding(key.WithKeys("esc")),
	}
}
