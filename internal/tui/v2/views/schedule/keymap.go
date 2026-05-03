package schedule

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Toggle key.Binding
	Edit   key.Binding
	Filter key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up:     key.NewBinding(key.WithKeys("up", "k")),
		Down:   key.NewBinding(key.WithKeys("down", "j")),
		Toggle: key.NewBinding(key.WithKeys("enter", " ")),
		Edit:   key.NewBinding(key.WithKeys("e")),
		Filter: key.NewBinding(key.WithKeys("/")),
	}
}
