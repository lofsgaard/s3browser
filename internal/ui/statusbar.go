package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type StatusBarModel struct {
	bucket        string
	currentPrefix string
	hint          string
	width         int
}

func newStatusBar(bucket string, width int) StatusBarModel {
	return StatusBarModel{bucket: bucket, width: width}
}

func (s StatusBarModel) view(overrideLeft string) string {
	left := overrideLeft
	if left == "" {
		left = s.pathStr()
	}

	right := s.hint
	if right == "" {
		right = "↑↓ move  Enter/→ open  ← back  D del  U upload  Q quit"
	}

	leftStyled := styleStatusBar.Render(left)
	rightStyled := styleStatusBarRight.Render(right)

	leftWidth := lipgloss.Width(leftStyled)
	rightWidth := lipgloss.Width(rightStyled)
	gap := s.width - leftWidth - rightWidth
	if gap < 0 {
		gap = 0
	}

	spacer := styleStatusBar.Copy().Width(gap).Render("")
	return leftStyled + spacer + rightStyled
}

func (s StatusBarModel) pathStr() string {
	parts := []string{s.bucket}
	if s.currentPrefix != "" {
		segs := strings.Split(strings.TrimSuffix(s.currentPrefix, "/"), "/")
		parts = append(parts, segs...)
	}
	return strings.Join(parts, " / ")
}
