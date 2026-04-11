package tui

import "github.com/charmbracelet/lipgloss"

// Catppuccin Mocha color palette
const (
	colorBg       = "#1a1b1e"
	colorBgHeader = "#181820"
	colorBorder   = "#2a2b35"
	colorSelected = "#2a2b3d"
	colorText     = "#cdd6f4"
	colorDim      = "#45475a"
	colorPurple   = "#cba6f7"
	colorBlue     = "#89b4fa"
	colorGreen    = "#a6e3a1"
	colorRed      = "#f38ba8"
	colorYellow   = "#f9e2af"
)

var (
	StyleBase = lipgloss.NewStyle().
		Background(lipgloss.Color(colorBg)).
		Foreground(lipgloss.Color(colorText))

	StyleHeader = lipgloss.NewStyle().
		Background(lipgloss.Color(colorBgHeader)).
		Foreground(lipgloss.Color(colorPurple)).
		Bold(true).
		Padding(0, 1)

	StyleSidebarHead = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorDim)).
		Bold(true).
		Padding(0, 1)

	StyleTaskRowNormal = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorText)).
		Padding(0, 1)

	StyleTaskRowSelected = lipgloss.NewStyle().
		Background(lipgloss.Color(colorSelected)).
		Foreground(lipgloss.Color(colorText)).
		Padding(0, 1)

	StyleTaskName = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorBlue))

	StyleTag = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorDim)).
		Background(lipgloss.Color(colorBg))

	StyleLogHead = lipgloss.NewStyle().
		Background(lipgloss.Color(colorBgHeader)).
		Padding(0, 1)

	StyleLogName = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorPurple)).
		Bold(true)

	StyleLogCmd = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorDim))

	StyleStatusOk = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorGreen)).
		Bold(true)

	StyleStatusRunning = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorBlue)).
		Bold(true)

	StyleStatusFailed = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorRed)).
		Bold(true)

	StyleStatusIdle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorDim))

	StyleStatusWarn = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorYellow)).
		Bold(true)

	StyleLogDim = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorDim))

	StyleLogOk = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorGreen))

	StyleLogErr = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorRed))

	StyleLogInfo = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorBlue))

	StyleLogWarn = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorYellow))

	StyleLogText = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorText))

	StyleStatusBar = lipgloss.NewStyle().
		Background(lipgloss.Color(colorBgHeader)).
		Foreground(lipgloss.Color(colorDim)).
		Padding(0, 1)

	StyleKey = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorPurple)).
		Bold(true)

	StyleTabActive = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorPurple)).
		Bold(true).
		Padding(0, 1)

	StyleTabInactive = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorDim)).
		Padding(0, 1)

	StyleSectionTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorBlue)).
		Bold(true).
		Padding(0, 1)

	StyleHelp = lipgloss.NewStyle().
		Background(lipgloss.Color(colorBgHeader)).
		Foreground(lipgloss.Color(colorText)).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(colorBorder)).
		Padding(1, 2)
)
