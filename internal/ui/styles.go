package ui

import "github.com/charmbracelet/lipgloss"

const (
	colSizeWidth     = 10
	colModifiedWidth = 19
	colStorageWidth  = 8
)

var (
	styleSelected = lipgloss.NewStyle().
			Bold(true).
			Background(lipgloss.Color("237")).
			Foreground(lipgloss.Color("255"))

	stylePrefix = lipgloss.NewStyle().
			Foreground(lipgloss.Color("74"))

	styleObject = lipgloss.NewStyle()

	styleSize = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Width(colSizeWidth).
			Align(lipgloss.Right)

	styleModified = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Width(colModifiedWidth)

	styleStorage = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Width(colStorageWidth)

	styleHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("245")).
			BorderBottom(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("238"))

	styleStatusBar = lipgloss.NewStyle().
			Background(lipgloss.Color("235")).
			Foreground(lipgloss.Color("252")).
			Padding(0, 1)

	styleStatusBarRight = lipgloss.NewStyle().
				Background(lipgloss.Color("235")).
				Foreground(lipgloss.Color("243")).
				Padding(0, 1)

	styleError = lipgloss.NewStyle().
			Foreground(lipgloss.Color("167")).
			Bold(true)

	styleLoading = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))
)
