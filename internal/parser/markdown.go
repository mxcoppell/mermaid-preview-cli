package parser

import "regexp"

var mermaidBlockRe = regexp.MustCompile("(?m)^```mermaid\\s*\n([\\s\\S]*?)^```")

// ExtractMermaidBlocks extracts all fenced mermaid code blocks from markdown content.
// Returns a slice of mermaid diagram source strings.
func ExtractMermaidBlocks(content string) []string {
	matches := mermaidBlockRe.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return nil
	}

	blocks := make([]string, len(matches))
	for i, m := range matches {
		blocks[i] = m[1]
	}
	return blocks
}
