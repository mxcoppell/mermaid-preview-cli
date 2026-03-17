# mermaid-preview-cli

Lightweight CLI to preview Mermaid diagrams in a native frameless window (macOS only).

## For LLM Agents

To display a mermaid diagram, pipe it to stdin. The CLI opens a native
frameless window and exits immediately (exit code 0).

```bash
echo "graph TD; A-->B-->C" | mermaid-preview-cli
```

On success, the CLI prints `Previewing <name>` to **stdout** so agents can
confirm the window opened. If a file is already open, it prints
`Previewing <name> (reused)` and activates the existing window (no duplicate).

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
| `--verbose` | false | Show informational messages on stderr |
| `-v, --version` | - | Print version |
| `-h, --help` | - | Print help |

## Exit Codes

- `0` — success
- `1` — argument error (bad flags, missing file)
- `2` — runtime error (port in use, read failure)

## Output Format

**Stdout** (agent-consumable confirmation):
```
Previewing flowchart.mmd              # new window opened
Previewing flowchart.mmd (reused)     # existing window activated
Previewing stdin                      # stdin input
```

**Stderr** (structured diagnostics, `--verbose` for info messages):
```
mermaid-preview-cli: listening on http://127.0.0.1:52341 (flow.mmd)  # --verbose only
mermaid-preview-cli: shutting down
mermaid-preview-cli: error: <message>                                # exit code 1 or 2
```

## Architecture

macOS-only Go binary with embedded mermaid.js and a native frameless webview
(`github.com/webview/webview_go`). Two-process model: CLI spawns GUI subprocess
via `--internal-host=<config.json>`, exits immediately. Multi-window host: First
invocation spawns a persistent host process. Subsequent invocations join via IPC
socket, opening new windows in the same process. The host manages all windows,
the dock icon, and the NSApp event loop. Duplicate file detection prevents
opening the same file twice (activates the existing window instead). Windows
cascade automatically so multi-file opens don't stack on top of each other.

## Build & Test

```bash
go build -ldflags="-s -w" -o bin/mermaid-preview-cli ./cmd/mermaid-preview-cli  # build
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
cmd/mermaid-preview-cli/main.go      # entrypoint → cmd.Execute(), gui.RunHost(), or gui.Run()
cmd/root.go                          # flag parsing, config, IPC-first spawn
internal/gui/host.go                 # multi-window host: IPC, window management, lifecycle
internal/gui/gui.go                  # legacy single-window GUI entry point
internal/gui/window.go               # webview creation, JS bindings (legacy path)
internal/gui/config.go               # CLI→GUI IPC via temp JSON file
internal/gui/frameless_darwin.go     # macOS frameless window, save panel (Cocoa/CGO)
internal/gui/dockicon_darwin.go      # programmatic dock icon rendering (CoreGraphics/CGO)
internal/gui/dockmenu_darwin.go      # HostDelegate, dock right-click menu, NSApp init
internal/gui/dockmenu_callbacks_darwin.go  # CGO export callbacks for dock menu
internal/ipc/ipc.go                  # Unix socket IPC client (Dial, SendOpen)
internal/ipc/server.go               # Unix socket IPC server
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
scripts/gen-icon.m                   # standalone dock icon PNG generator
assets/dock-icon.png                 # reference dock icon image
testdata/                            # test fixtures (.mmd, .md)
e2e/                                 # Playwright E2E tests
```
