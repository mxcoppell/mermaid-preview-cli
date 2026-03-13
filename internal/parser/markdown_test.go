package parser

import "testing"

func TestExtractMermaidBlocks(t *testing.T) {
	content := "# Title\n\n```mermaid\ngraph LR; A-->B\n```\n\nSome text.\n\n```mermaid\nsequenceDiagram\n    A->>B: Hello\n```\n"

	blocks := ExtractMermaidBlocks(content)
	if len(blocks) != 2 {
		t.Fatalf("got %d blocks, want 2", len(blocks))
	}
	if blocks[0] != "graph LR; A-->B\n" {
		t.Errorf("block[0] = %q", blocks[0])
	}
	if blocks[1] != "sequenceDiagram\n    A->>B: Hello\n" {
		t.Errorf("block[1] = %q", blocks[1])
	}
}

func TestExtractMermaidBlocks_NoBlocks(t *testing.T) {
	content := "# Just markdown\n\nNo mermaid here.\n\n```go\nfmt.Println()\n```\n"

	blocks := ExtractMermaidBlocks(content)
	if blocks != nil {
		t.Errorf("got %v, want nil", blocks)
	}
}

func TestExtractMermaidBlocks_Empty(t *testing.T) {
	blocks := ExtractMermaidBlocks("")
	if blocks != nil {
		t.Errorf("got %v, want nil", blocks)
	}
}

func TestExtractMermaidBlocks_SingleBlock(t *testing.T) {
	content := "```mermaid\npie\n    \"A\" : 50\n    \"B\" : 50\n```\n"

	blocks := ExtractMermaidBlocks(content)
	if len(blocks) != 1 {
		t.Fatalf("got %d blocks, want 1", len(blocks))
	}
}
