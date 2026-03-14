# mermaid-preview

Lightweight CLI to preview Mermaid diagrams in a native frameless window (macOS).

A single Go binary with embedded mermaid.js that opens a frameless webview window to render diagrams. Supports live reload on file changes. No browser dependency, no internet, no Node.js.

## Install

### Homebrew (macOS)

```bash
brew install mxie/tap/mermaid-preview
```

### Download binary

Grab the latest release from [GitHub Releases](https://github.com/mxie/mermaid-preview/releases).

### Build from source

```bash
go build -ldflags="-s -w" -o bin/mermaid-preview .
```

## Quick Start

```bash
# Pipe from stdin (CLI exits immediately, window stays open)
echo "graph LR; A-->B-->C" | mermaid-preview

# Preview a file (live reload on changes)
mermaid-preview diagram.mmd

# Multiple files — each gets its own window
mermaid-preview flow.mmd sequence.mmd

# Extracts ```mermaid blocks from markdown
mermaid-preview README.md
```

## Supported Files

| Extension | Behavior |
|-----------|----------|
| `.mmd`, `.mermaid` | Mermaid diagram files |
| `.md`, `.markdown` | Extracts ` ```mermaid ` fenced blocks |

## CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-p, --port PORT` | auto | Server port |
| `-t, --theme THEME` | system | `dark`, `light`, or `system` |
| `-w, --no-watch` | false | Disable file watching |
| `--poll INTERVAL` | — | Stat-based polling fallback (e.g. `500ms`) |
| `-v, --version` | — | Print version |
| `-h, --help` | — | Print help |

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `Cmd+F` | Search nodes |
| `T` | Toggle theme (system → light → dark) |
| `+` / `-` | Zoom in / out |
| `0` | Reset zoom (fit to viewport) |
| `Esc` | Close search, or close window |
| `Space` | Close window |

## Stdin

Pipe any mermaid source to stdin. The CLI renders it and exits immediately — the window stays open independently. Max input: 10MB.

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | Argument error (bad flags, missing file) |
| `2` | Runtime error (port in use, read failure) |

## Agent Integration

`mermaid-preview` includes a Claude Code skill that lets agents automatically discover and use it when you ask to visualize diagrams.

```bash
# Install the skill (symlink stays up to date)
ln -s "$(pwd)/skills/mermaid-preview.md" ~/.claude/skills/mermaid-preview.md
```

Once installed, asking Claude Code to "show this as a diagram" or "visualize this flow" will automatically pipe Mermaid source through `mermaid-preview`.

## Contributing

```bash
# Build
go build -ldflags="-s -w" -o bin/mermaid-preview .

# Run tests
go test ./...

# E2E tests (requires Node.js)
cd e2e && npm ci && npx playwright install chromium && npx playwright test
```

## License

MIT
