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
│       ├── statusbar.go         # Bottom status bar (path, key hints, prompts)
│       └── styles.go            # All lipgloss style constants
```

## Key dependencies

- **`github.com/aws/aws-sdk-go-v2`** — AWS SDK. Credentials resolved via default chain (env vars → `~/.aws/credentials` → instance metadata).
- **`github.com/charmbracelet/bubbletea`** — Elm-architecture TUI framework.
- **`github.com/charmbracelet/bubbles`** — Reusable TUI components (`textinput` for upload prompt).
- **`github.com/charmbracelet/lipgloss`** — Terminal styling and layout.

## Architecture

The app follows the [Bubble Tea](https://github.com/charmbracelet/bubbletea) model: a root `Model` (`ui/model.go`) owns a `BrowserModel` and a `StatusBarModel`. All state lives in these structs; updates flow through `Update()` returning new state + commands.

### S3 navigation

`ListObjectsV2` is always called with `Delimiter: "/"`. This makes S3 behave like a directory tree:
- `CommonPrefixes` → folder entries (`KindPrefix`)
- `Contents` → file entries (`KindObject`)

The current path is tracked as a `currentPrefix` string (e.g. `photos/2024/`). Navigating into a folder pushes to `prefixStack`; going back pops it.

### Browser state machine

`BrowserModel` has four states (`browserState`):

| State | Description |
|-------|-------------|
| `stateNormal` | Default — navigate, open files |
| `stateConfirmDelete` | Waiting for Y/N confirmation before deleting |
| `stateUploadInput` | Text input active for local file path |
| `stateOpening` | Downloading + opening a file, input blocked |

### File opening

`Enter` on a `KindObject` entry runs `openFileCmd`: downloads the object body to `os.TempDir()/filename`, then calls the OS default opener (`cmd /c start` on Windows, `open` on macOS, `xdg-open` on Linux).

### S3-compatible endpoints

When `--endpoint` is set, the S3 client uses `BaseEndpoint` + `UsePathStyle: true`. This covers MinIO, LocalStack, Intility, and any other S3-compatible service.

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
