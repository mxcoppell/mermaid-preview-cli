---
name: mermaid-preview-cli
description: Preview, visualize, render, display, or show Mermaid diagrams in a native frameless window — supports stdin fire-and-forget, live file preview, markdown extraction, and multi-file comparison
---

# Mermaid Preview

Display Mermaid diagrams in a native frameless window using the `mermaid-preview-cli` CLI. Single binary with embedded mermaid.js — no browser, no Node.js.

## Important: Always Pipe, Never Create Files

**Do NOT create `.mmd` files just to preview them.** Always pipe diagram source directly to stdin:

```bash
echo 'graph TD
    A[Start] --> B{Decision}
    B -->|Yes| C[Do Thing]
    B -->|No| D[Skip]' | mermaid-preview-cli
```

The CLI exits immediately (exit code 0), the window stays open. No temp files, no cleanup.

## Recommended: Run in a Subagent

Diagram rendering is a visual side-effect — no output needs to return to the conversation. Running in a subagent keeps the main context clean.

```
Agent tool call:
  prompt: |
    Render this Mermaid diagram using mermaid-preview-cli:

    echo 'graph TD
        A[User Request] --> B[Auth Service]
        B --> C{Valid?}
        C -->|Yes| D[Process]
        C -->|No| E[Reject]' | mermaid-preview-cli
  subagent_type: general-purpose
```

## Use Cases

### Previewing Existing Files

For files that already exist in the project, use file mode with live reload:

```bash
mermaid-preview-cli diagram.mmd           # single file
mermaid-preview-cli README.md             # extracts ```mermaid blocks
mermaid-preview-cli before.mmd after.mmd  # side-by-side comparison
```

## Quick Reference

### Flags

| Flag | Purpose |
|------|---------|
| `--theme dark\|light\|system` | Set color theme (default: system) |
| `--no-watch` | Disable file watching |
| `--poll INTERVAL` | Stat-based polling fallback (e.g. `500ms`) |
| `--port PORT` | Set server port (default: auto) |

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `Cmd+F` | Search nodes |
| `T` | Toggle theme (system → light → dark) |
| `+` / `-` | Zoom in / out |
| `0` | Reset zoom (fit to viewport) |
| `Esc` | Close search, or close window |
| `Space` | Close window |

### Export

The toolbar provides SVG and PNG export via a native save dialog.

## Diagram Styling

For guidance on writing well-styled Mermaid diagrams with bold colors and theme compatibility, see the `mermaid-diagram-guide` skill.
