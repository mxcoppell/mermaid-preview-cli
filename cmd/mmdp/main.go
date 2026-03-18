package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/mxcoppell/mmdp/cmd"
	"github.com/mxcoppell/mmdp/internal/gui"
)

func main() {
	// Check for internal flags before normal flag parsing.
	// This keeps subprocess startup fast — no cobra/flag overhead.
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "--internal-host=") {
			cfgPath := strings.TrimPrefix(arg, "--internal-host=")
			if err := gui.RunHost(cfgPath); err != nil {
				fmt.Fprintf(os.Stderr, "mmdp: error: %v\n", err)
				os.Exit(2)
			}
			return
		}
		// Legacy single-window mode
		if strings.HasPrefix(arg, "--internal-gui=") {
			cfgPath := strings.TrimPrefix(arg, "--internal-gui=")
			if err := gui.Run(cfgPath); err != nil {
				fmt.Fprintf(os.Stderr, "mmdp: error: %v\n", err)
				os.Exit(2)
			}
			return
		}
	}

	os.Exit(cmd.Execute())
}
