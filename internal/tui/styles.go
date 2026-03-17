package tui

import "github.com/charmbracelet/lipgloss"

var (
	TitleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00d4ff")).MarginBottom(1)
	SelectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff88")).Bold(true)
	DimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))
	SizeStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffaa00")).Bold(true)
	DangerStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff4444")).Bold(true)
	HelpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).MarginTop(1)
	StatusStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#00d4ff"))
)
