package panes

import (
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

// LogView is a thin wrapper around viewport.Model for displaying log output.
type LogView struct {
	vp viewport.Model
}

// NewLogView creates a new LogView with the given dimensions.
func NewLogView(width, height int) LogView {
	vp := viewport.New(width, height)
	vp.Style = lipgloss.NewStyle()
	return LogView{vp: vp}
}

// SetContent updates the log content and scrolls to the bottom.
func (l *LogView) SetContent(content string) {
	l.vp.SetContent(content)
	l.vp.GotoBottom()
}

// View renders the log viewport.
func (l *LogView) View() string {
	return l.vp.View()
}

// SetSize resizes the viewport.
func (l *LogView) SetSize(width, height int) {
	l.vp.Width = width
	l.vp.Height = height
}
