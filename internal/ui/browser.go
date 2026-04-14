package ui

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	s3client "github.com/lofsgaard/s3browser/internal/s3"
)

type browserState int

const (
	stateNormal browserState = iota
	stateConfirmDelete
	stateUploadInput
	stateUploading
	stateOpening
)

// -- message types ------------------------------------------------------------

type entriesLoadedMsg struct {
	entries   []s3client.Entry
	nextToken string
	err       error
}

type actionDoneMsg struct {
	err error
}

type fileOpenedMsg struct {
	err error
}

type uploadProgressMsg struct {
	filename  string
	bytesRead int64
	total     int64
	elapsed   time.Duration
}

type noopMsg struct{}

// -- progress reader ----------------------------------------------------------

type progressReader struct {
	r         io.Reader
	filename  string
	total     int64
	read      int64
	startTime time.Time
	ch        chan<- uploadProgressMsg
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.r.Read(p)
	pr.read += int64(n)
	select {
	case pr.ch <- uploadProgressMsg{
		filename:  pr.filename,
		bytesRead: pr.read,
		total:     pr.total,
		elapsed:   time.Since(pr.startTime),
	}:
	default:
	}
	return n, err
}

// waitForProgress returns a Cmd that blocks until the next progress update.
func waitForProgress(ch <-chan uploadProgressMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return noopMsg{}
		}
		return msg
	}
}

// -- model --------------------------------------------------------------------

type BrowserModel struct {
	s3             *s3client.Client
	bucket         string
	currentPrefix  string
	prefixStack    []string
	pageHistory    []string
	entries        []s3client.Entry
	cursor         int
	loading        bool
	nextToken      string
	state          browserState
	statusMsg      string
	statusExpiry   time.Time
	uploadInput    textinput.Model
	uploadProgress uploadProgressMsg
	uploadCh       <-chan uploadProgressMsg
	height         int
	width          int
}

func newBrowser(client *s3client.Client, bucket string, width, height int) BrowserModel {
	ti := textinput.New()
	ti.Placeholder = "Local file path..."
	ti.CharLimit = 512

	return BrowserModel{
		s3:          client,
		bucket:      bucket,
		loading:     true,
		height:      height,
		width:       width,
		uploadInput: ti,
	}
}

func (b BrowserModel) Init() tea.Cmd {
	return b.fetchEntries("")
}

func (b BrowserModel) fetchEntries(contToken string) tea.Cmd {
	prefix := b.currentPrefix
	return func() tea.Msg {
		entries, next, err := b.s3.ListDir(context.Background(), prefix, contToken)
		return entriesLoadedMsg{entries: entries, nextToken: next, err: err}
	}
}

func (b BrowserModel) Update(msg tea.Msg) (BrowserModel, tea.Cmd) {
	switch msg := msg.(type) {
	case noopMsg:
		return b, nil

	case entriesLoadedMsg:
		b.loading = false
		if msg.err != nil {
			b.statusMsg = styleError.Render("Error: " + msg.err.Error())
			b.statusExpiry = time.Now().Add(5 * time.Second)
		} else {
			b.entries = msg.entries
			b.nextToken = msg.nextToken
			b.cursor = 0
		}
		return b, nil

	case uploadProgressMsg:
		b.uploadProgress = msg
		return b, waitForProgress(b.uploadCh)

	case actionDoneMsg:
		b.loading = false
		if msg.err != nil {
			b.statusMsg = styleError.Render("Error: " + msg.err.Error())
			b.statusExpiry = time.Now().Add(5 * time.Second)
		} else {
			b.statusMsg = ""
		}
		b.state = stateNormal
		b.loading = true
		return b, b.fetchEntries("")

	case fileOpenedMsg:
		b.state = stateNormal
		if msg.err != nil {
			b.statusMsg = styleError.Render("Error: " + msg.err.Error())
			b.statusExpiry = time.Now().Add(5 * time.Second)
		} else {
			b.statusMsg = ""
		}
		return b, nil

	case tea.KeyMsg:
		if !b.statusExpiry.IsZero() && time.Now().After(b.statusExpiry) {
			b.statusMsg = ""
			b.statusExpiry = time.Time{}
		}

		switch b.state {
		case stateConfirmDelete:
			return b.handleConfirmDelete(msg)
		case stateUploadInput:
			return b.handleUploadInput(msg)
		case stateOpening, stateUploading:
			return b, nil // block input during async operations
		default:
			return b.handleNormal(msg)
		}
	}

	if b.state == stateUploadInput {
		var cmd tea.Cmd
		b.uploadInput, cmd = b.uploadInput.Update(msg)
		return b, cmd
	}

	return b, nil
}

func (b BrowserModel) handleNormal(msg tea.KeyMsg) (BrowserModel, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if b.cursor > 0 {
			b.cursor--
		}
	case "down", "j":
		if b.cursor < len(b.entries)-1 {
			b.cursor++
		}
	case "enter", "right", "l":
		if len(b.entries) == 0 {
			break
		}
		entry := b.entries[b.cursor]
		if entry.Kind == s3client.KindPrefix {
			b.prefixStack = append(b.prefixStack, b.currentPrefix)
			b.pageHistory = nil
			b.currentPrefix = entry.FullKey
			b.loading = true
			b.cursor = 0
			return b, b.fetchEntries("")
		}
		b.state = stateOpening
		b.statusMsg = fmt.Sprintf("Opening %s...", entry.Name)
		b.statusExpiry = time.Time{}
		return b, b.openFileCmd(entry)

	case "backspace", "left", "h":
		if len(b.prefixStack) > 0 {
			b.currentPrefix = b.prefixStack[len(b.prefixStack)-1]
			b.prefixStack = b.prefixStack[:len(b.prefixStack)-1]
			b.pageHistory = nil
			b.loading = true
			b.cursor = 0
			return b, b.fetchEntries("")
		}
	case "n":
		if b.nextToken != "" {
			b.pageHistory = append(b.pageHistory, b.nextToken)
			token := b.nextToken
			b.loading = true
			return b, b.fetchEntries(token)
		}
	case "p":
		if len(b.pageHistory) > 1 {
			b.pageHistory = b.pageHistory[:len(b.pageHistory)-1]
			token := b.pageHistory[len(b.pageHistory)-1]
			b.loading = true
			return b, b.fetchEntries(token)
		} else if len(b.pageHistory) == 1 {
			b.pageHistory = nil
			b.loading = true
			return b, b.fetchEntries("")
		}
	case "d":
		if len(b.entries) > 0 && b.entries[b.cursor].Kind == s3client.KindObject {
			b.state = stateConfirmDelete
		} else {
			b.statusMsg = "Select a file to delete"
			b.statusExpiry = time.Now().Add(3 * time.Second)
		}
	case "u":
		b.state = stateUploadInput
		b.uploadInput.SetValue("")
		b.uploadInput.Focus()
		return b, textinput.Blink
	}
	return b, nil
}

func (b BrowserModel) openFileCmd(entry s3client.Entry) tea.Cmd {
	key := entry.FullKey
	name := entry.Name
	return func() tea.Msg {
		body, err := b.s3.GetObject(context.Background(), key)
		if err != nil {
			return fileOpenedMsg{err: err}
		}
		defer body.Close()

		tmpPath := filepath.Join(os.TempDir(), name)
		f, err := os.Create(tmpPath)
		if err != nil {
			return fileOpenedMsg{err: fmt.Errorf("create temp file: %w", err)}
		}
		if _, err = io.Copy(f, body); err != nil {
			f.Close()
			return fileOpenedMsg{err: fmt.Errorf("download: %w", err)}
		}
		f.Close()

		if err = openWithOS(tmpPath); err != nil {
			return fileOpenedMsg{err: fmt.Errorf("open: %w", err)}
		}
		return fileOpenedMsg{}
	}
}

func openWithOS(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", path)
	case "darwin":
		cmd = exec.Command("open", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	return cmd.Start()
}

func (b BrowserModel) handleConfirmDelete(msg tea.KeyMsg) (BrowserModel, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		if len(b.entries) == 0 {
			b.state = stateNormal
			return b, nil
		}
		key := b.entries[b.cursor].FullKey
		b.loading = true
		b.state = stateNormal
		return b, func() tea.Msg {
			err := b.s3.Delete(context.Background(), key)
			return actionDoneMsg{err: err}
		}
	default:
		b.state = stateNormal
	}
	return b, nil
}

func (b BrowserModel) handleUploadInput(msg tea.KeyMsg) (BrowserModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		localPath := strings.TrimSpace(b.uploadInput.Value())
		localPath = strings.Trim(localPath, `"'`)
		if localPath == "" {
			b.state = stateNormal
			return b, nil
		}
		b.uploadInput.Blur()

		info, err := os.Stat(localPath)
		if err != nil {
			b.state = stateNormal
			b.statusMsg = styleError.Render("File not found: " + localPath)
			b.statusExpiry = time.Now().Add(4 * time.Second)
			return b, nil
		}

		ch := make(chan uploadProgressMsg, 32)
		b.uploadCh = ch
		b.state = stateUploading
		b.uploadProgress = uploadProgressMsg{
			filename: filepath.Base(localPath),
			total:    info.Size(),
		}

		prefix := b.currentPrefix
		s3c := b.s3
		size := info.Size()
		name := filepath.Base(localPath)

		uploadCmd := func() tea.Msg {
			f, err := os.Open(localPath)
			if err != nil {
				close(ch)
				return actionDoneMsg{err: fmt.Errorf("open file: %w", err)}
			}
			defer f.Close()
			pr := &progressReader{
				r:         f,
				filename:  name,
				total:     size,
				startTime: time.Now(),
				ch:        ch,
			}
			key := prefix + name
			err = s3c.Upload(context.Background(), key, pr, size)
			close(ch)
			return actionDoneMsg{err: err}
		}

		return b, tea.Batch(uploadCmd, waitForProgress(ch))

	case "esc":
		b.state = stateNormal
		b.uploadInput.Blur()
	default:
		var cmd tea.Cmd
		b.uploadInput, cmd = b.uploadInput.Update(msg)
		return b, cmd
	}
	return b, nil
}

// -- view ---------------------------------------------------------------------

func (b BrowserModel) breadcrumb() string {
	parts := []string{"Bucket: " + b.bucket}
	if b.currentPrefix != "" {
		segs := strings.Split(strings.TrimSuffix(b.currentPrefix, "/"), "/")
		parts = append(parts, segs...)
	}
	return styleBreadcrumb.Render(strings.Join(parts, " / "))
}

func (b BrowserModel) View() string {
	crumb := b.breadcrumb()

	if b.loading {
		return crumb + "\n" + styleLoading.Render("  Loading...")
	}
	if len(b.entries) == 0 {
		return crumb + "\n" + styleLoading.Render("  (empty)")
	}

	tableWidth := b.width
	nameWidth := tableWidth - colSizeWidth - colModifiedWidth - colStorageWidth - 6
	if nameWidth < 10 {
		nameWidth = 10
	}

	header := lipgloss.JoinHorizontal(lipgloss.Top,
		styleHeader.Copy().Width(nameWidth).Render("Name"),
		"  ",
		styleHeader.Copy().Width(colSizeWidth).Align(lipgloss.Right).Render("Size"),
		"  ",
		styleHeader.Copy().Width(colModifiedWidth).Render("Modified"),
		"  ",
		styleHeader.Copy().Width(colStorageWidth).Render("Class"),
	)

	var rows []string
	rows = append(rows, crumb, header)

	maxRows := b.height - 3
	if maxRows < 1 {
		maxRows = 1
	}

	start := 0
	if b.cursor >= maxRows {
		start = b.cursor - maxRows + 1
	}
	end := start + maxRows
	if end > len(b.entries) {
		end = len(b.entries)
	}

	for i := start; i < end; i++ {
		entry := b.entries[i]
		selected := i == b.cursor

		nameStyle := styleObject
		if entry.Kind == s3client.KindPrefix {
			nameStyle = stylePrefix
		}

		name := entry.Name
		if entry.Kind == s3client.KindPrefix {
			name = name + "/"
		}
		if len(name) > nameWidth {
			name = name[:nameWidth-1] + "…"
		}

		sizeStr := ""
		modStr := ""
		scStr := entry.StorageClass

		if entry.Kind == s3client.KindObject {
			sizeStr = formatSize(entry.Size)
			modStr = entry.LastModified.Local().Format("2006-01-02 15:04:05")
		}

		nameCol := nameStyle.Copy().Width(nameWidth).Render(name)
		sizeCol := styleSize.Render(sizeStr)
		modCol := styleModified.Render(modStr)
		scCol := styleStorage.Render(scStr)

		row := lipgloss.JoinHorizontal(lipgloss.Top,
			nameCol, "  ", sizeCol, "  ", modCol, "  ", scCol,
		)

		if selected {
			row = styleSelected.Copy().Width(tableWidth).Render(row)
		}
		rows = append(rows, row)
	}

	return strings.Join(rows, "\n")
}

func (b BrowserModel) statusBarLeft() string {
	switch b.state {
	case stateConfirmDelete:
		if len(b.entries) > 0 {
			return fmt.Sprintf("Delete %q? [Y]es / [N]o", b.entries[b.cursor].Name)
		}
	case stateUploadInput:
		return "Upload: " + b.uploadInput.View()
	case stateUploading:
		return renderUploadProgress(b.uploadProgress)
	case stateOpening:
		return b.statusMsg
	}
	if !b.statusExpiry.IsZero() && time.Now().Before(b.statusExpiry) {
		return b.statusMsg
	}
	return ""
}

func renderUploadProgress(p uploadProgressMsg) string {
	const barWidth = 16

	transferred := formatSize(p.bytesRead)

	var speedStr string
	if p.elapsed.Seconds() > 0.5 {
		bps := float64(p.bytesRead) / p.elapsed.Seconds()
		speedStr = formatSize(int64(bps)) + "/s"
	}

	if p.total <= 0 {
		if speedStr != "" {
			return fmt.Sprintf("Uploading %s  %s  %s", p.filename, transferred, speedStr)
		}
		return fmt.Sprintf("Uploading %s  %s", p.filename, transferred)
	}

	pct := float64(p.bytesRead) / float64(p.total)
	if pct > 1 {
		pct = 1
	}
	filled := int(pct * barWidth)
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
	total := formatSize(p.total)

	var eta string
	if speedStr != "" && p.bytesRead < p.total {
		bps := float64(p.bytesRead) / p.elapsed.Seconds()
		remaining := time.Duration(float64(p.total-p.bytesRead)/bps) * time.Second
		eta = "  ETA " + remaining.Round(time.Second).String()
	}

	return fmt.Sprintf("Uploading %s  %s  %d%%  %s / %s  %s%s",
		p.filename, bar, int(pct*100), transferred, total, speedStr, eta)
}

// -- helpers ------------------------------------------------------------------

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
