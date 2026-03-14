package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/mxie/mermaid-preview-cli/cmd"
	"github.com/mxie/mermaid-preview-cli/internal/gui"
)

func main() {
	// Check for internal GUI mode before normal flag parsing.
	// This keeps GUI startup fast — no cobra/flag overhead.
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "--internal-gui=") {
			cfgPath := strings.TrimPrefix(arg, "--internal-gui=")
			if err := gui.Run(cfgPath); err != nil {
				fmt.Fprintf(os.Stderr, "mermaid-preview-cli: error: %v\n", err)
				os.Exit(2)
			}
			return
		}
	}

	os.Exit(cmd.Execute())
}
