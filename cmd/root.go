package cmd

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/term"

	"github.com/mxcoppell/mmdp/internal/gui"
	"github.com/mxcoppell/mmdp/internal/ipc"
	"github.com/mxcoppell/mmdp/internal/parser"
	"github.com/mxcoppell/mmdp/internal/version"
)

const maxStdinSize = 10 * 1024 * 1024 // 10MB

// Config holds all CLI configuration.
type Config struct {
	Port    int
	Theme   string
	NoWatch bool
	Poll    time.Duration
	Files   []string
	IsStdin bool
	Verbose bool
}

func Execute() int {
	cfg, err := parseFlags(os.Args[1:], os.Stdin)
	if err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		fmt.Fprintf(os.Stderr, "mmdp: error: %v\n", err)
		return 1
	}

	if err := run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "mmdp: error: %v\n", err)
		return 2
	}
	return 0
}

func parseFlags(args []string, stdin *os.File) (Config, error) {
	var cfg Config
	var showVersion bool

	fs := flag.NewFlagSet("mmdp", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	fs.IntVar(&cfg.Port, "port", 0, "")
	fs.IntVar(&cfg.Port, "p", 0, "")
	fs.StringVar(&cfg.Theme, "theme", "system", "")
	fs.StringVar(&cfg.Theme, "t", "system", "")
	fs.BoolVar(&cfg.NoWatch, "no-watch", false, "")
	fs.BoolVar(&cfg.NoWatch, "w", false, "")
	fs.DurationVar(&cfg.Poll, "poll", 0, "")
	fs.BoolVar(&showVersion, "version", false, "")
	fs.BoolVar(&showVersion, "v", false, "")
	fs.BoolVar(&cfg.Verbose, "verbose", false, "")

	// Custom help handling
	var showHelp bool
	fs.BoolVar(&showHelp, "help", false, "")
	fs.BoolVar(&showHelp, "h", false, "")

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	if showHelp {
		printHelp(os.Stdout)
		return Config{}, flag.ErrHelp
	}

	if showVersion {
		fmt.Fprintf(os.Stdout, "mmdp %s\n", version.Version)
		return Config{}, flag.ErrHelp
	}

	// Validate theme
	switch cfg.Theme {
	case "dark", "light", "system":
	default:
		return Config{}, fmt.Errorf("invalid theme %q: must be dark, light, or system", cfg.Theme)
	}

	// Check for file arguments first, then fall back to stdin detection
	remaining := fs.Args()
	if len(remaining) >= 1 {
		cfg.Files = remaining
	} else if !term.IsTerminal(int(stdin.Fd())) {
		cfg.IsStdin = true
		cfg.NoWatch = true
		cfg.Files = []string{""}
	} else {
		printHelp(os.Stderr)
		return Config{}, fmt.Errorf("no input file specified")
	}

	return cfg, nil
}

func run(cfg Config) error {
	for _, file := range cfg.Files {
		// Read content
		var content string
		if cfg.IsStdin {
			data, err := io.ReadAll(io.LimitReader(os.Stdin, maxStdinSize))
			if err != nil {
				return fmt.Errorf("reading stdin: %w", err)
			}
			content = string(data)
		} else {
			data, err := os.ReadFile(file)
			if err != nil {
				return fmt.Errorf("reading %s: %w", file, err)
			}
			content = string(data)
		}

		isMarkdown := strings.HasSuffix(file, ".md") || strings.HasSuffix(file, ".markdown")

		// For markdown, validate that there are mermaid blocks
		if isMarkdown {
			blocks := parser.ExtractMermaidBlocks(content)
			if len(blocks) == 0 {
				return fmt.Errorf("no mermaid diagram blocks found in %s", file)
			}
		}

		// Build watch file list and normalize path for dedup
		var watchFiles []string
		var filePath string
		if !cfg.IsStdin {
			absPath, err := filepath.Abs(file)
			if err != nil {
				return fmt.Errorf("resolving path: %w", err)
			}
			resolved, err := filepath.EvalSymlinks(absPath)
			if err != nil {
				resolved = absPath
			}
			filePath = resolved
			if !cfg.NoWatch {
				watchFiles = []string{absPath}
			}
		}

		// Spawn GUI process
		guiCfg := gui.Config{
			Port:       cfg.Port,
			Theme:      cfg.Theme,
			Content:    content,
			Filename:   filepath.Base(file),
			IsMarkdown: isMarkdown,
			WatchFiles: watchFiles,
			Poll:       cfg.Poll,
			NoWatch:    cfg.NoWatch,
			Verbose:    cfg.Verbose,
			FilePath:   filePath,
		}

		resp, err := spawnGUI(guiCfg)
		if err != nil {
			return err
		}

		// Print confirmation to stdout for agent consumption.
		displayName := filepath.Base(file)
		if cfg.IsStdin {
			displayName = "stdin"
		}
		if resp.Reused {
			fmt.Fprintf(os.Stdout, "Previewing %s (reused)\n", displayName)
		} else {
			fmt.Fprintf(os.Stdout, "Previewing %s\n", displayName)
		}
	}

	return nil
}

func spawnGUI(cfg gui.Config) (ipc.OpenResponse, error) {
	// Try IPC to existing host first
	if resp, ok := trySendIPC(cfg); ok {
		return resp, nil
	}

	// No host running — spawn one
	err := spawnHostProcess(cfg)
	if err != nil {
		// Lost race — another process spawned the host, retry IPC.
		if resp, ok := trySendIPC(cfg); ok {
			return resp, nil
		}
		return ipc.OpenResponse{}, err
	}

	waitForHost()
	return ipc.OpenResponse{OK: true}, nil
}

// trySendIPC attempts to connect to an existing host and send the config.
func trySendIPC(cfg gui.Config) (ipc.OpenResponse, bool) {
	conn, err := ipc.Dial()
	if err != nil {
		return ipc.OpenResponse{}, false
	}
	defer conn.Close()

	tmpPath, err := gui.WriteConfig(cfg)
	if err != nil {
		return ipc.OpenResponse{}, false
	}

	resp, err := ipc.SendOpen(conn, tmpPath)
	if err != nil {
		os.Remove(tmpPath)
		return ipc.OpenResponse{}, false
	}
	if !resp.OK {
		fmt.Fprintf(os.Stderr, "mmdp: host error: %s\n", resp.Error)
		return ipc.OpenResponse{}, false
	}
	return resp, true
}

// waitForHost polls until the IPC socket accepts connections (up to ~2.5s).
func waitForHost() {
	for range 50 {
		if ipc.IsHostRunning() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// spawnHostProcess starts a new host process with the initial config.
func spawnHostProcess(cfg gui.Config) error {
	tmpPath, err := gui.WriteConfig(cfg)
	if err != nil {
		return fmt.Errorf("writing GUI config: %w", err)
	}

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding executable: %w", err)
	}

	cmd := exec.Command(exePath, "--internal-host="+tmpPath)
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("spawning host: %w", err)
	}

	return nil
}

func printHelp(w io.Writer) {
	fmt.Fprint(w, `USAGE:
    mmdp [FLAGS] [FILE...]

ARGUMENTS:
    FILE    One or more .mmd, .mermaid, or .md files
            Each file opens in its own preview window
            (.md files: extracts `+"`"+`mermaid`+"`"+` code blocks)

FLAGS:
    -p, --port PORT       Server port (default: auto-select available)
    -t, --theme THEME     dark | light | system (default: system)
    -w, --no-watch        Disable file watching
        --poll INTERVAL   Stat-based polling fallback (e.g. 500ms)
        --verbose         Show informational messages on stderr
    -v, --version         Print version
    -h, --help            Print help

STDIN:
    echo "graph LR; A-->B" | mmdp
    cat diagram.mmd | mmdp

AGENT TOOL USAGE:
    Pipe mermaid source to stdin. The CLI opens a native preview window
    and exits immediately (exit code 0). No server process is left
    running from the CLI's perspective.

    On success, prints "Previewing <name>" to stdout. If a file is
    already open, prints "Previewing <name> (reused)" and activates
    the existing window instead of opening a duplicate.

    Example from an LLM agent:
        echo "graph TD; A-->B-->C" | mmdp

    For file-based preview with live reload:
        mmdp diagram.mmd

KEYBOARD SHORTCUTS (in preview window):
    Cmd/Ctrl+F  Search nodes      T  Toggle theme
    +/-         Zoom in/out       0  Reset zoom
    Esc         Close window      Space  Close window

EXIT CODES: 0 = success, 1 = argument error, 2 = runtime error
`)
}
