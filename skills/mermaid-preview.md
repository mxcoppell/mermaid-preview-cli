---
name: mermaid-preview
description: Preview, visualize, render, or display Mermaid diagrams in the browser using the mermaid-preview CLI
---

# Mermaid Preview

Use the `mermaid-preview` CLI to display Mermaid diagrams in the user's browser.

## When to Use

Activate when the user asks to **visualize**, **preview**, **display**, **render**, or **show** a Mermaid diagram.

## Primary Usage (fire-and-forget)

Pipe diagram source to stdin. The CLI opens a browser window and exits immediately — no server, no cleanup.

```bash
echo 'graph TD
    A[Start] --> B{Decision}
    B -->|Yes| C[Do Thing]
    B -->|No| D[Skip]
    C --> E[End]
    D --> E' | mermaid-preview
```

This is the recommended approach for agents. The browser window is self-contained and stays open after the CLI exits (exit code 0).

## File Preview (live reload)

For iterative editing, preview a file directly. The CLI starts a server and watches for changes:

```bash
mermaid-preview diagram.mmd
```

## Key Flags

| Flag | Purpose |
|------|---------|
| `--theme dark\|light\|system` | Set color theme |
| `--once` | Force fire-and-forget mode (default for stdin) |
| `--no-browser` | Don't auto-open browser (useful for testing) |

## Markdown Support

The CLI extracts ` ```mermaid ` blocks from `.md` files automatically:

```bash
mermaid-preview README.md
```

## Diagram Styling

For guidance on writing well-styled Mermaid diagrams with bold colors and theme compatibility, see the `mermaid-diagram-guide` skill.
