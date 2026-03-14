---
name: mermaid-preview-cli
description: Preview, visualize, render, display, or show Mermaid diagrams in a native frameless window — supports stdin fire-and-forget, live file preview, markdown extraction, and multi-file comparison
---

# Mermaid Preview

Display Mermaid diagrams in a native frameless window using the `mermaid-preview-cli` CLI. No browser, no internet, no Node.js.

## Recommended: Run in a Subagent

Diagram rendering is a visual side-effect — no output needs to return to the conversation. Running `mermaid-preview-cli` in a subagent keeps the main context clean and avoids wasting tokens on tool output from a fire-and-forget command.

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

The subagent runs the command, the window opens, and the main conversation continues uninterrupted.

## Use Cases

### 1. Inline Visualization (Agents)

Pipe diagram source to stdin. The CLI opens a native window and exits immediately (exit code 0) — no cleanup needed.

Best for: visualizing architecture, data flows, sequences, or state machines during conversations.

```bash
echo 'graph TD
    A[Start] --> B{Decision}
    B -->|Yes| C[Do Thing]
    B -->|No| D[Skip]
    C --> E[End]
    D --> E' | mermaid-preview-cli
```

### 2. Live Preview While Editing

Point at a `.mmd` or `.mermaid` file for live reload. Changes are reflected instantly (100ms debounce). Uses fsnotify by default, with stat-based polling as a fallback.

Best for: iterating on diagram files in an editor.

```bash
mermaid-preview-cli diagram.mmd
```

### 3. Markdown Documentation

The CLI extracts ` ```mermaid ` fenced blocks from `.md` files automatically. Multiple blocks render stacked with labels.

Best for: previewing diagrams embedded in documentation.

```bash
mermaid-preview-cli README.md
```

### 4. Side-by-Side Comparison

Pass multiple files — each opens in its own window.

Best for: comparing diagram variants or reviewing changes.

```bash
mermaid-preview-cli before.mmd after.mmd
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
