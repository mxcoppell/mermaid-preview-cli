# mermaid-preview

Lightweight CLI to preview Mermaid diagrams in the browser.

## For LLM Agents

To display a mermaid diagram, pipe it to stdin. The CLI opens a browser
window and exits immediately (exit code 0). No server is left running.

```bash
echo "graph TD; A-->B-->C" | mermaid-preview
```

This is the recommended way for agents to show diagrams. The browser
window is self-contained and stays open after the CLI exits.

You can also preview an existing file (this starts a server with live reload):

```bash
mermaid-preview diagram.mmd
```

## Quick Usage

```bash
# Fire-and-forget preview (stdin — CLI exits immediately)
echo "graph LR; A-->B" | mermaid-preview

# Live preview with file watching (CLI stays running)
mermaid-preview diagram.mmd

# Multiple files — each gets its own window
mermaid-preview flow.mmd sequence.mmd arch.md

# Extracts ```mermaid blocks from markdown
mermaid-preview README.md

# Force fire-and-forget mode for files
mermaid-preview --once diagram.mmd

# Force server mode for stdin
echo "graph LR; A-->B" | mermaid-preview --serve
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-p, --port PORT` | auto | Server port |
| `-b, --no-browser` | false | Don't auto-open browser |
| `-t, --theme THEME` | system | dark, light, or system |
| `-w, --no-watch` | false | Disable file watching |
| `--poll INTERVAL` | - | Stat-based polling (e.g. 500ms) |
| `--once` | default for stdin | Render to self-contained HTML and exit |
| `--serve` | default for files | Force server mode (live reload) |
| `-v, --version` | - | Print version |
| `-h, --help` | - | Print help |

## Exit Codes

- `0` — success
- `1` — argument error (bad flags, missing file)
- `2` — runtime error (port in use, read failure)

## Stderr Format

All output is structured: `mermaid-preview: <message>`

```
mermaid-preview: wrote /tmp/mermaid-preview-12345.html           # --once mode
mermaid-preview: listening on http://127.0.0.1:52341 (flow.mmd)  # server mode
mermaid-preview: shutting down
mermaid-preview: no clients connected, shutting down             # auto-shutdown
mermaid-preview: error: <message>                                # exit code 1 or 2
```

## Architecture

Go binary with embedded mermaid.js (~2.5MB). Two modes:
- **stdin/--once**: writes a self-contained HTML file to /tmp, opens browser, exits
- **file/--serve**: starts HTTP server on 127.0.0.1 with WebSocket live reload

## Build & Test

```bash
go build -ldflags="-s -w" -o mermaid-preview .     # build
go test ./...                                        # unit tests
cd e2e && npm ci && npx playwright test              # E2E tests
```

## Agent Skill

A Claude Code skill is provided at `skills/mermaid-preview.md`. To install it
for use across projects, symlink or copy it into your Claude Code skills
directory:

```bash
# Symlink (recommended — stays up to date)
ln -s "$(pwd)/skills/mermaid-preview.md" ~/.claude/skills/mermaid-preview.md
```

Once installed, Claude Code will automatically activate the skill when you ask
to visualize, preview, or display a Mermaid diagram.

## Project Structure

```
main.go                          # entrypoint → cmd.Execute()
cmd/root.go                      # flag parsing, config, orchestration
cmd/once.go                      # --once mode: self-contained HTML generation
internal/server/server.go        # HTTP server, routes, lifecycle
internal/server/websocket.go     # WebSocket broadcast (mutex+slice)
internal/watcher/watcher.go      # fsnotify + poll fallback
internal/browser/open.go         # cross-platform browser open (Chrome --app mode)
internal/parser/markdown.go      # extract mermaid blocks from .md
internal/version/version.go      # build-time version injection
web/embed.go                     # //go:embed directives
web/templates/index.html         # single-page template
web/static/app.js                # frontend logic (vanilla JS)
web/static/style.css             # dark/light theme CSS
web/static/mermaid.min.js        # vendored mermaid IIFE build
testdata/                        # test fixtures (.mmd, .md)
e2e/                             # Playwright E2E tests
```
