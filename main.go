package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lofsgaard/s3browser/internal/config"
	s3client "github.com/lofsgaard/s3browser/internal/s3"
	"github.com/lofsgaard/s3browser/internal/ui"
)

func main() {
	cfg := config.Parse()

	client, err := s3client.NewClient(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	m := ui.NewModel(cfg, client)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
