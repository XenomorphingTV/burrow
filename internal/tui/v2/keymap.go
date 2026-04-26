package v2

import "github.com/charmbracelet/bubbles/key"

type GlobalKeyMap struct {
	Quit key.Binding
	Tab  key.Binding
	Help key.Binding
}

func DefaultKeyMap() GlobalKeyMap {
	return GlobalKeyMap{
		Quit: key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Tab:  key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next tab")),
		Help: key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	}
}
