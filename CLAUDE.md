# s3browser

A terminal UI (TUI) application for browsing S3-compatible object storage, written in Go.

## Project structure

```
s3browser/
├── main.go                      # Entry point: parse flags, build client, launch TUI
├── internal/
│   ├── config/
│   │   └── config.go            # CLI flag parsing and AppConfig struct
│   ├── s3/
│   │   ├── types.go             # Entry and EntryKind types (shared domain model)
│   │   └── client.go            # S3 client wrapper (ListDir, GetObject, Delete, Upload)
│   └── ui/
│       ├── model.go             # Root Bubble Tea Model — wires browser + status bar
│       ├── browser.go           # Interactive table, navigation state machine, file actions
│       ├── statusbar.go         # Single-line bottom status bar (endpoint, hints, prompts)
│       └── styles.go            # All lipgloss style constants
```

## Key dependencies

- **`github.com/aws/aws-sdk-go-v2`** — AWS SDK. Credentials resolved via default chain (env vars → `~/.aws/credentials` → instance metadata).
- **`github.com/charmbracelet/bubbletea`** — Elm-architecture TUI framework.
- **`github.com/charmbracelet/bubbles`** — Reusable TUI components (`textinput` for upload prompt).
- **`github.com/charmbracelet/lipgloss`** — Terminal styling and layout.

## Layout

```
Bucket: my-bucket / prefix / subprefix     ← breadcrumb (styleBreadcrumb, no background)
────────────────────────────────────────── ← header with bottom border (styleHeader)
Name            Size       Modified         ← column headers
──────────────────────────────────────────
folder/                                    ← KindPrefix entries (stylePrefix, blue)
file.txt        1.2 KB   2026-01-01 ...    ← KindObject entries
s3.intility.com           ↑↓ move  ...    ← status bar (single line, dark background)
```

Height budget (terminal height H):
- `m.browser.height = H - 1` (1 line for status bar)
- `maxRows = b.height - 3` (breadcrumb + header text + border = 3 lines)

## Architecture

The app follows the [Bubble Tea](https://github.com/charmbracelet/bubbletea) model: a root `Model` (`ui/model.go`) owns a `BrowserModel` and a `StatusBarModel`. All state lives in these structs; updates flow through `Update()` returning new state + commands.

### S3 navigation

`ListObjectsV2` is always called with `Delimiter: "/"`. This makes S3 behave like a directory tree:
- `CommonPrefixes` → folder entries (`KindPrefix`)
- `Contents` → file entries (`KindObject`)

The current path is tracked as a `currentPrefix` string (e.g. `photos/2024/`). Navigating into a folder pushes to `prefixStack`; going back pops it. Pagination uses `NextContinuationToken`; `pageHistory []string` tracks past tokens for backward navigation.

### Browser state machine

`BrowserModel` has five states (`browserState`):

| State | Description |
|-------|-------------|
| `stateNormal` | Default — navigate, open files |
| `stateConfirmDelete` | Waiting for Y/N confirmation before deleting |
| `stateUploadInput` | Text input active for local file path |
| `stateUploading` | Upload in progress — input blocked, progress shown in status bar |
| `stateOpening` | Downloading + opening a file, input blocked |

### Upload progress

Uploading uses a `progressReader` that wraps `io.Reader` and sends `uploadProgressMsg` values to a buffered channel on every `Read()` call. A `waitForProgress` Cmd blocks on that channel and re-queues itself on each message, driving UI updates. The status bar shows filename, an ASCII progress bar, percentage, transferred/total, speed, and ETA.

Upload path input strips surrounding quotes (`"` and `'`) so Windows "Copy as path" works directly.

### File opening

`Enter` on a `KindObject` entry runs `openFileCmd`: downloads the object body to `os.TempDir()/filename`, then calls the OS default opener (`cmd /c start` on Windows, `open` on macOS, `xdg-open` on Linux).

### Status bar

`StatusBarModel` renders a single line:
- **Default**: endpoint hostname on the left (accent color), key hints on the right (dim).
- **Override** (delete confirm, upload prompt, upload progress, opening): full-width prompt replaces the line.
- Background is filled to the full terminal width using explicit space characters rendered with the background style (avoids lipgloss `Width()` padding issues).

### S3-compatible endpoints

When `--endpoint` is set, the S3 client uses `BaseEndpoint` + `UsePathStyle: true`. This covers MinIO, LocalStack, Intility, and any other S3-compatible service. The endpoint hostname (scheme and path stripped) is shown in the status bar.

## Building

```bash
go build -o s3browser.exe .
```

## Running

```bash
s3browser --bucket my-bucket --region eu-west-1
s3browser --bucket my-bucket --endpoint https://s3.intility.com
```

See README.md for full auth and usage documentation.
