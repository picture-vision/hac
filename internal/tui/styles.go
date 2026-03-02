package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	colorPrimary   = lipgloss.Color("#7C3AED")
	colorSecondary = lipgloss.Color("#06B6D4")
	colorOn        = lipgloss.Color("#22C55E")
	colorOff       = lipgloss.Color("#6B7280")
	colorError     = lipgloss.Color("#EF4444")
	colorWarning   = lipgloss.Color("#F59E0B")
	colorBg        = lipgloss.Color("#1E1E2E")
	colorBorder    = lipgloss.Color("#45475A")
	colorMuted     = lipgloss.Color("#6C7086")
	colorText      = lipgloss.Color("#CDD6F4")

	// Layout
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary).
			Padding(0, 1)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Padding(0, 1)

	// List items
	itemStyle = lipgloss.NewStyle().
			Padding(0, 1)

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(colorPrimary).
				Bold(true).
				Padding(0, 1)

	// State colors
	stateOnStyle = lipgloss.NewStyle().
			Foreground(colorOn).
			Bold(true)

	stateOffStyle = lipgloss.NewStyle().
			Foreground(colorOff)

	stateUnavailableStyle = lipgloss.NewStyle().
				Foreground(colorError)

	// Borders and panels
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)

	activePanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorPrimary).
				Padding(0, 1)

	titleStyle = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Bold(true)

	labelStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	valueStyle = lipgloss.NewStyle().
			Foreground(colorText)

	errorStyle = lipgloss.NewStyle().
			Foreground(colorError).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(colorOn).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Padding(0, 1)

	areaHeaderStyle = lipgloss.NewStyle().
			Foreground(colorWarning).
			Bold(true).
			Padding(0, 1)
)

func stateStyle(state string) lipgloss.Style {
	switch state {
	case "on", "open", "playing", "home", "above_horizon":
		return stateOnStyle
	case "unavailable", "unknown":
		return stateUnavailableStyle
	default:
		return stateOffStyle
	}
}
