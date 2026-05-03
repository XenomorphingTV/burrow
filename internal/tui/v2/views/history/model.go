package history

import (
	style "github.com/XenomorphingTV/burrow/internal/tui/v2/styles"
	"github.com/XenomorphingTV/burrow/internal/store"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	styles style.Styles
	theme  style.Theme
	keys   KeyMap

	Width  int
	Height int

	Records  []*store.RunRecord
	Selected int
	Scroll   int

	FilterMode  bool
	FilterInput string

	confirmClear bool

	viewport viewport.Model
}

func New(styles style.Styles, theme style.Theme) Model {
	vp := viewport.New(80, 20)
	return Model{
		styles:   styles,
		theme:    theme,
		keys:     DefaultKeyMap(),
		viewport: vp,
	}
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.FilterMode {
			return m.handleFilterKey(msg)
		}
		return m.handleKey(msg)
	}
	return m, nil
}
