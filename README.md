# mermaid-preview

Lightweight CLI to preview Mermaid diagrams in the browser with live reload.

A single Go binary (~10MB) that starts a local HTTP server, opens your browser, and renders Mermaid diagrams with live reload on file changes. No internet required, no editor plugin needed.

## Install

### Homebrew (macOS/Linux)

```bash
brew install mxie/tap/mermaid-preview
```

### Scoop (Windows)

```bash
scoop bucket add mxie https://github.com/mxie/scoop-bucket
scoop install mermaid-preview
```

### Download binary

Grab the latest release from [GitHub Releases](https://github.com/mxie/mermaid-preview/releases).

### Build from source

```bash
go install github.com/mxie/mermaid-preview@latest
```

## Quick Start

```bash
# Preview a .mmd file
mermaid-preview diagram.mmd

# Pipe from stdin
echo "graph LR; A-->B-->C" | mermaid-preview

# Extract and preview mermaid blocks from markdown
mermaid-preview README.md
```

## CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-p, --port PORT` | auto | Server port |
| `-b, --no-browser` | false | Don't auto-open browser |
| `-t, --theme THEME` | system | `dark`, `light`, or `system` |
| `-w, --no-watch` | false | Disable file watching |
| `--poll INTERVAL` | — | Polling fallback for WSL/Docker/NFS (e.g. `500ms`) |
| `--once` | default for stdin | Render to self-contained HTML and exit (no server) |
| `--serve` | default for files | Force server mode (override `--once` for stdin) |
| `-v, --version` | — | Print version |
| `-h, --help` | — | Print help |

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `Cmd/Ctrl+F` | Search nodes |
| `T` | Toggle theme (system → light → dark) |
| `+` / `-` | Zoom in / out |
| `0` | Reset zoom |
| `Esc` | Close search, or quit server |

## How It Works

`mermaid-preview` is a Go binary with mermaid.js embedded via `//go:embed`. It starts an HTTP server on `127.0.0.1`, serves a single-page app that renders diagrams client-side using mermaid.js, and pushes file changes to the browser via WebSocket. No external dependencies, no internet, no Node.js.

## Contributing

```bash
# Build
go build -ldflags="-s -w" -o mermaid-preview .

# Run tests
go test ./...

# E2E tests (requires Node.js)
cd e2e && npm ci && npx playwright install chromium && npx playwright test
```

## License

MIT
