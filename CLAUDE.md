# mermaid-preview-cli

Lightweight CLI to preview Mermaid diagrams in a native frameless window (macOS only).

## For LLM Agents

To display a mermaid diagram, pipe it to stdin. The CLI opens a native
frameless window and exits immediately (exit code 0).

```bash
echo "graph TD; A-->B-->C" | mermaid-preview-cli
```

This is the recommended way for agents to show diagrams. The window
stays open after the CLI exits.

You can also preview an existing file (this starts a server with live reload):

```bash
mermaid-preview-cli diagram.mmd
```

## Quick Usage

```bash
# Fire-and-forget preview (stdin — CLI exits immediately)
echo "graph LR; A-->B" | mermaid-preview-cli

# Live preview with file watching
mermaid-preview-cli diagram.mmd

# Multiple files — each gets its own window
mermaid-preview-cli flow.mmd sequence.mmd arch.md

# Extracts ```mermaid blocks from markdown
mermaid-preview-cli README.md
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-p, --port PORT` | auto | Server port |
| `-t, --theme THEME` | system | dark, light, or system |
| `-w, --no-watch` | false | Disable file watching |
| `--poll INTERVAL` | - | Stat-based polling (e.g. 500ms) |
| `-v, --version` | - | Print version |
| `-h, --help` | - | Print help |

## Exit Codes

- `0` — success
- `1` — argument error (bad flags, missing file)
- `2` — runtime error (port in use, read failure)

## Stderr Format

All output is structured: `mermaid-preview-cli: <message>`

```
mermaid-preview-cli: listening on http://127.0.0.1:52341 (flow.mmd)  # server mode
mermaid-preview-cli: shutting down
mermaid-preview-cli: error: <message>                                # exit code 1 or 2
```

## Architecture

macOS-only Go binary with embedded mermaid.js and a native frameless webview
(`github.com/webview/webview_go`). Dual-mode binary: the CLI reads input, writes
a temp config, spawns itself with `--internal-gui`, and exits. The GUI process
runs an HTTP server + webview event loop.

## Build & Test

```bash
go build -ldflags="-s -w" -o bin/mermaid-preview-cli .   # build
go test ./...                                         # unit tests
cd e2e && npm ci && npx playwright test               # E2E tests
```

## Agent Skill

A Claude Code skill is provided at `skills/mermaid-preview-cli.md`. To install it
for use across projects, symlink or copy it into your Claude Code skills
directory:

```bash
# Symlink (recommended — stays up to date)
mkdir -p ~/.claude/skills/mermaid-preview-cli
ln -s "$(pwd)/skills/mermaid-preview-cli.md" ~/.claude/skills/mermaid-preview-cli/SKILL.md
```

Once installed, Claude Code will automatically activate the skill when you ask
to visualize, preview, or display a Mermaid diagram.

## Project Structure

```
main.go                              # entrypoint → cmd.Execute() or gui.Run()
cmd/root.go                          # flag parsing, config, spawn GUI process
internal/gui/gui.go                  # GUI entry point: server + webview lifecycle
internal/gui/window.go               # webview creation, JS bindings
internal/gui/config.go               # CLI→GUI IPC via temp JSON file
internal/gui/frameless_darwin.go     # macOS frameless window (Cocoa/CGO)
internal/server/server.go            # HTTP server, routes, lifecycle
internal/server/websocket.go         # WebSocket broadcast (mutex+slice)
internal/watcher/watcher.go          # fsnotify + poll fallback
internal/parser/markdown.go          # extract mermaid blocks from .md
internal/version/version.go          # build-time version injection
web/embed.go                         # //go:embed directives
web/templates/index.html             # single-page template
web/static/app.js                    # frontend logic (vanilla JS)
web/static/style.css                 # dark/light theme CSS
web/static/mermaid.min.js            # vendored mermaid IIFE build
testdata/                            # test fixtures (.mmd, .md)
e2e/                                 # Playwright E2E tests
```
