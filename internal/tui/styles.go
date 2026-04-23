package tui

import "github.com/charmbracelet/lipgloss"

// activeTheme is the currently applied color palette.
var activeTheme = ThemeCatppuccinMocha

// initStyles rebuilds all Lipgloss style variables from the given theme.
// Call this once during startup before creating the TUI model.
func initStyles(t Theme) {
	activeTheme = t

	StyleBase = lipgloss.NewStyle().
		Background(lipgloss.Color(t.Bg)).
		Foreground(lipgloss.Color(t.Text))

	StyleHeader = lipgloss.NewStyle().
		Background(lipgloss.Color(t.BgHeader)).
		Foreground(lipgloss.Color(t.Purple)).
		Bold(true).
		Padding(0, 1)

	StyleSidebarHead = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Dim)).
		Bold(true).
		Padding(0, 1)

	StyleTaskRowNormal = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Text)).
		Padding(0, 1)

	StyleTaskRowSelected = lipgloss.NewStyle().
		Background(lipgloss.Color(t.Selected)).
		Foreground(lipgloss.Color(t.Text)).
		Padding(0, 1)

	StyleTaskName = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Blue))

	StyleTag = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Dim)).
		Background(lipgloss.Color(t.Bg))

	StyleLogHead = lipgloss.NewStyle().
		Background(lipgloss.Color(t.BgHeader)).
		Padding(0, 1)

	StyleLogName = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Purple)).
		Bold(true)

	StyleLogCmd = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Dim))

	StyleStatusOk = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Green)).
		Bold(true)

	StyleStatusRunning = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Blue)).
		Bold(true)

	StyleStatusFailed = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Red)).
		Bold(true)

	StyleStatusIdle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Dim))

	StyleStatusWarn = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Yellow)).
		Bold(true)

	StyleLogDim = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Dim))

	StyleLogOk = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Green))

	StyleLogErr = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Red))

	StyleLogInfo = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Blue))

	StyleLogWarn = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Yellow))

	StyleLogText = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Text))

	StyleStatusBar = lipgloss.NewStyle().
		Background(lipgloss.Color(t.BgHeader)).
		Foreground(lipgloss.Color(t.Dim)).
		Padding(0, 1)

	StyleKey = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Purple)).
		Bold(true)

	StyleTabActive = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Purple)).
		Bold(true).
		Padding(0, 1)

	StyleTabInactive = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Dim)).
		Padding(0, 1)

	StyleSectionTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Blue)).
		Bold(true).
		Padding(0, 1)

	StyleHelp = lipgloss.NewStyle().
		Background(lipgloss.Color(t.BgHeader)).
		Foreground(lipgloss.Color(t.Text)).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(t.Border)).
		Padding(1, 2)
}

// Style variables — populated by initStyles on startup.
var (
	StyleBase           lipgloss.Style
	StyleHeader         lipgloss.Style
	StyleSidebarHead    lipgloss.Style
	StyleTaskRowNormal  lipgloss.Style
	StyleTaskRowSelected lipgloss.Style
	StyleTaskName       lipgloss.Style
	StyleTag            lipgloss.Style
	StyleLogHead        lipgloss.Style
	StyleLogName        lipgloss.Style
	StyleLogCmd         lipgloss.Style
	StyleStatusOk       lipgloss.Style
	StyleStatusRunning  lipgloss.Style
	StyleStatusFailed   lipgloss.Style
	StyleStatusIdle     lipgloss.Style
	StyleStatusWarn     lipgloss.Style
	StyleLogDim         lipgloss.Style
	StyleLogOk          lipgloss.Style
	StyleLogErr         lipgloss.Style
	StyleLogInfo        lipgloss.Style
	StyleLogWarn        lipgloss.Style
	StyleLogText        lipgloss.Style
	StyleStatusBar      lipgloss.Style
	StyleKey            lipgloss.Style
	StyleTabActive      lipgloss.Style
	StyleTabInactive    lipgloss.Style
	StyleSectionTitle   lipgloss.Style
	StyleHelp           lipgloss.Style
)

func init() {
	initStyles(ThemeCatppuccinMocha)
}
