# mermaid-preview-cli

Lightweight CLI to preview Mermaid diagrams in a native frameless window (macOS).

A single Go binary with embedded mermaid.js that opens a frameless webview window to render diagrams. Supports live reload on file changes. No browser dependency, no internet, no Node.js.

## Install

### Homebrew (macOS)

```bash
brew install mxcoppell/tap/mermaid-preview-cli
```

### Download binary

Grab the latest release from [GitHub Releases](https://github.com/mxcoppell/mermaid-preview-cli/releases).

### Build from source

```bash
go build -ldflags="-s -w" -o bin/mermaid-preview-cli .
```

## Quick Start

```bash
# Pipe from stdin (CLI exits immediately, window stays open)
echo "graph LR; A-->B-->C" | mermaid-preview-cli

# Preview a file (live reload on changes)
mermaid-preview-cli diagram.mmd

# Multiple files — each gets its own window
mermaid-preview-cli flow.mmd sequence.mmd

# Extracts ```mermaid blocks from markdown
mermaid-preview-cli README.md
```

## Use Cases

### Coding Agents — Inline Visualization

Fire-and-forget stdin mode for agents. Pipe diagram source, the CLI exits immediately, and the window stays open. Ideal for visualizing architecture, data flows, and sequences mid-conversation. For best results, run in a subagent to keep the main conversation context clean.

### Live Preview While Editing

Point at a `.mmd` or `.mermaid` file for live reload. Changes are reflected instantly (100ms debounce). Supports fsnotify (default) or stat-based polling (`--poll`).

### Markdown Documentation

Extracts ` ```mermaid ` fenced blocks from `.md` files automatically. Multiple blocks render stacked with labels.

### Side-by-Side Comparison

Pass multiple files — each opens in its own window. Useful for comparing diagram variants or reviewing changes.

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

`mermaid-preview-cli` includes a Claude Code skill that lets agents automatically discover and use it when you ask to visualize diagrams. The skill file provides full agent-facing documentation including use cases and recommended patterns.

```bash
# Install the skill (symlink stays up to date)
ln -s "$(pwd)/skills/mermaid-preview-cli.md" ~/.claude/skills/mermaid-preview-cli.md
```

Once installed, asking Claude Code to "show this as a diagram" or "visualize this flow" will automatically pipe Mermaid source through `mermaid-preview-cli`.

For best results, agents should run `mermaid-preview-cli` in a subagent — diagram rendering is a visual side-effect with no output to return, so delegating it keeps the main conversation context clean.

## Contributing

```bash
# Build
go build -ldflags="-s -w" -o bin/mermaid-preview-cli .

# Run tests
go test ./...

# E2E tests (requires Node.js)
cd e2e && npm ci && npx playwright install chromium && npx playwright test
```

## License

MIT
