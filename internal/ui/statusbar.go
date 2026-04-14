package ui

import (
	"net/url"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type StatusBarModel struct {
	endpoint string
	hint     string
	width    int
}

func newStatusBar(endpoint string, width int) StatusBarModel {
	return StatusBarModel{endpoint: endpointHost(endpoint), width: width}
}

// endpointHost strips scheme and path from an endpoint URL, returning just the host.
func endpointHost(endpoint string) string {
	if endpoint == "" {
		return ""
	}
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" {
		return endpoint
	}
	return u.Host
}

var bgStyle = lipgloss.NewStyle().Background(lipgloss.Color("235"))

func fill(n int) string {
	if n <= 0 {
		return ""
	}
	return bgStyle.Render(strings.Repeat(" ", n))
}

// view renders a single-line status bar.
// When overrideLeft is set (delete confirm, upload prompt, etc.) it replaces the full line.
// Otherwise: endpoint on the left, key hints on the right.
func (s StatusBarModel) view(overrideLeft string) string {
	if overrideLeft != "" {
		rendered := styleStatusBar.Render(overrideLeft)
		used := lipgloss.Width(rendered)
		return rendered + fill(s.width-used)
	}

	hints := s.hint
	if hints == "" {
		hints = "↑↓ move  Enter/→ open  ← back  D del  U upload  Q quit"
	}
	hintsRendered := styleStatusBarRight.Render(hints)
	hintsWidth := lipgloss.Width(hintsRendered)

	if s.endpoint != "" {
		epRendered := styleStatusBarEndpoint.Render(s.endpoint)
		epWidth := lipgloss.Width(epRendered)
		gap := s.width - epWidth - hintsWidth
		return epRendered + fill(gap) + hintsRendered
	}

	gap := s.width - hintsWidth
	return fill(gap) + hintsRendered
}
