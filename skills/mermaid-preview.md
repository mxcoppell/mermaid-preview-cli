---
name: mermaid-preview
description: Preview, visualize, render, or display Mermaid diagrams in a native frameless window using the mermaid-preview CLI
---

# Mermaid Preview

Use the `mermaid-preview` CLI to display Mermaid diagrams in a native frameless window.

## When to Use

Activate when the user asks to **visualize**, **preview**, **display**, **render**, or **show** a Mermaid diagram.

## Primary Usage (fire-and-forget)

Pipe diagram source to stdin. The CLI opens a native window and exits immediately — no cleanup needed.

```bash
echo 'graph TD
    A[Start] --> B{Decision}
    B -->|Yes| C[Do Thing]
    B -->|No| D[Skip]
    C --> E[End]
    D --> E' | mermaid-preview
```

This is the recommended approach for agents. The window stays open after the CLI exits (exit code 0).

## File Preview (live reload)

For iterative editing, preview a file directly. The CLI starts a server and watches for changes:

```bash
mermaid-preview diagram.mmd
```

## Key Flags

| Flag | Purpose |
|------|---------|
| `--theme dark\|light\|system` | Set color theme |
| `--no-watch` | Disable file watching |
| `--poll INTERVAL` | Stat-based polling fallback (e.g. `500ms`) |

## Markdown Support

The CLI extracts ` ```mermaid ` blocks from `.md` files automatically:

```bash
mermaid-preview README.md
```

## Diagram Styling

For guidance on writing well-styled Mermaid diagrams with bold colors and theme compatibility, see the `mermaid-diagram-guide` skill.
