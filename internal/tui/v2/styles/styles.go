package style

import "github.com/charmbracelet/lipgloss"

type Styles struct {
	Base   lipgloss.Style
	Header lipgloss.Style

	SidebarHead lipgloss.Style

	TaskRowNormal   lipgloss.Style
	TaskRowSelected lipgloss.Style
	TaskName        lipgloss.Style

	GroupHeader lipgloss.Style
	ActivePanel lipgloss.Style

	Tag lipgloss.Style

	LogHead lipgloss.Style
	LogName lipgloss.Style
	LogCmd  lipgloss.Style

	StatusOk      lipgloss.Style
	StatusRunning lipgloss.Style
	StatusFailed  lipgloss.Style
	StatusIdle    lipgloss.Style
	StatusWarn    lipgloss.Style

	LogDim  lipgloss.Style
	LogOk   lipgloss.Style
	LogErr  lipgloss.Style
	LogInfo lipgloss.Style
	LogWarn lipgloss.Style
	LogText lipgloss.Style

	StatusBar    lipgloss.Style
	Key          lipgloss.Style
	TabActive    lipgloss.Style
	TabInactive  lipgloss.Style
	SectionTitle lipgloss.Style
	Help         lipgloss.Style
}

func NewStyle(t Theme) Styles {
	s := Styles{}

	s.Base = lipgloss.NewStyle().
		Background(lipgloss.Color(t.Bg)).
		Foreground(lipgloss.Color(t.Text))

	s.Header = lipgloss.NewStyle().
		Background(lipgloss.Color(t.BgHeader)).
		Foreground(lipgloss.Color(t.Purple)).
		Bold(true).
		Padding(0, 1)

	s.SidebarHead = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Dim)).
		Bold(true).
		Padding(0, 1)

	s.TaskRowNormal = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Text)).
		Padding(0, 1)

	s.TaskRowSelected = lipgloss.NewStyle().
		Background(lipgloss.Color(t.Selected)).
		Foreground(lipgloss.Color(t.Text)).
		Padding(0, 1)

	s.TaskName = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Blue))

	s.GroupHeader = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Blue)).
		Bold(true).
		Padding(0, 1)

	s.ActivePanel = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Purple)).
		Bold(true)

	s.Tag = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Dim)).
		Background(lipgloss.Color(t.Bg))

	s.LogHead = lipgloss.NewStyle().
		Background(lipgloss.Color(t.BgHeader)).
		Padding(0, 1)

	s.LogName = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Purple)).
		Bold(true)

	s.LogCmd = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Dim))

	s.StatusOk = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Green)).
		Bold(true)

	s.StatusRunning = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Blue)).
		Bold(true)

	s.StatusFailed = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Red)).
		Bold(true)

	s.StatusIdle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Dim))

	s.StatusWarn = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Yellow)).
		Bold(true)

	s.LogDim = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Dim))

	s.LogOk = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Green))

	s.LogErr = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Red))

	s.LogInfo = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Blue))

	s.LogWarn = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Yellow))

	s.LogText = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Text))

	s.StatusBar = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Dim)).
		Padding(0, 1)

	s.Key = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Purple)).
		Bold(true)

	s.TabActive = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Purple)).
		Bold(true).
		Padding(0, 1)

	s.TabInactive = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Dim)).
		Padding(0, 1)

	s.SectionTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Blue)).
		Bold(true).
		Padding(0, 1)

	s.Help = lipgloss.NewStyle().
		Background(lipgloss.Color(t.BgHeader)).
		Foreground(lipgloss.Color(t.Text)).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(t.Border)).
		Padding(1, 2)

	return s
}
