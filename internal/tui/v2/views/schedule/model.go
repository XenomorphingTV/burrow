package schedule

import (
	"github.com/XenomorphingTV/burrow/internal/config"
	style "github.com/XenomorphingTV/burrow/internal/tui/v2/styles"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type ScheduleEntry struct {
	Name     string
	Kind     string // "cron" or "watch"
	Cron     string
	Patterns []string
	Enabled  bool
}

type Model struct {
	cfg    *config.Config
	styles style.Styles
	theme  style.Theme
	keys   KeyMap

	Width  int
	Height int

	Entries  []ScheduleEntry
	Selected int
	Scroll   int

	FilterMode  bool
	FilterInput string

	editMode  bool
	editName  string
	editInput textinput.Model
	editErr   string
}

func New(cfg *config.Config, styles style.Styles, theme style.Theme) Model {
	ti := textinput.New()
	ti.Placeholder = "e.g. */5 * * * *"
	ti.CharLimit = 64
	ti.Width = 40
	return Model{
		cfg:       cfg,
		styles:    styles,
		theme:     theme,
		keys:      DefaultKeyMap(),
		editInput: ti,
	}
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.editMode {
			return m.handleEditKey(msg)
		}
		if m.FilterMode {
			return m.handleFilterKey(msg)
		}
		return m.handleKey(msg)
	}
	return m, nil
}
