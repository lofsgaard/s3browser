package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lofsgaard/s3browser/internal/config"
	s3client "github.com/lofsgaard/s3browser/internal/s3"
)

type Model struct {
	browser   BrowserModel
	statusBar StatusBarModel
	width     int
	height    int
}

func NewModel(cfg config.AppConfig, client *s3client.Client) Model {
	return Model{
		browser:   newBrowser(client, cfg.Bucket, 80, 24),
		statusBar: newStatusBar(cfg.Endpoint, 80),
	}
}

func (m Model) Init() tea.Cmd {
	return m.browser.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.browser.width = msg.Width
		m.browser.height = msg.Height - 1
		m.statusBar.width = msg.Width
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.browser, cmd = m.browser.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	browserView := m.browser.View()
	leftOverride := m.browser.statusBarLeft()
	statusView := m.statusBar.view(leftOverride)
	return browserView + "\n" + statusView
}
