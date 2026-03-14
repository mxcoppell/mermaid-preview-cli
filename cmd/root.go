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

	"github.com/mxie/mermaid-preview/internal/gui"
	"github.com/mxie/mermaid-preview/internal/parser"
	"github.com/mxie/mermaid-preview/internal/version"
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
}

func Execute() int {
	cfg, err := parseFlags(os.Args[1:], os.Stdin)
	if err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		fmt.Fprintf(os.Stderr, "mermaid-preview: error: %v\n", err)
		return 1
	}

	if err := run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "mermaid-preview: error: %v\n", err)
		return 2
	}
	return 0
}

func parseFlags(args []string, stdin *os.File) (Config, error) {
	var cfg Config
	var showVersion bool

	fs := flag.NewFlagSet("mermaid-preview", flag.ContinueOnError)
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
		fmt.Fprintf(os.Stdout, "mermaid-preview %s\n", version.Version)
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

		// Build watch file list
		var watchFiles []string
		if !cfg.NoWatch && !cfg.IsStdin {
			absPath, err := filepath.Abs(file)
			if err != nil {
				return fmt.Errorf("resolving path: %w", err)
			}
			watchFiles = []string{absPath}
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
		}

		if err := spawnGUI(guiCfg); err != nil {
			return err
		}
	}

	return nil
}

func spawnGUI(cfg gui.Config) error {
	tmpPath, err := gui.WriteConfig(cfg)
	if err != nil {
		return fmt.Errorf("writing GUI config: %w", err)
	}

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding executable: %w", err)
	}

	cmd := exec.Command(exePath, "--internal-gui="+tmpPath)
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("spawning GUI: %w", err)
	}

	return nil
}

func printHelp(w io.Writer) {
	fmt.Fprint(w, `USAGE:
    mermaid-preview [FLAGS] [FILE...]

ARGUMENTS:
    FILE    One or more .mmd, .mermaid, or .md files
            Each file opens in its own preview window
            (.md files: extracts `+"`"+`mermaid`+"`"+` code blocks)

FLAGS:
    -p, --port PORT       Server port (default: auto-select available)
    -t, --theme THEME     dark | light | system (default: system)
    -w, --no-watch        Disable file watching
        --poll INTERVAL   Stat-based polling fallback (e.g. 500ms)
    -v, --version         Print version
    -h, --help            Print help

STDIN:
    echo "graph LR; A-->B" | mermaid-preview
    cat diagram.mmd | mermaid-preview

AGENT TOOL USAGE:
    Pipe mermaid source to stdin. The CLI opens a native preview window
    and exits immediately (exit code 0). No server process is left
    running from the CLI's perspective.

    Example from an LLM agent:
        echo "graph TD; A-->B-->C" | mermaid-preview

    For file-based preview with live reload:
        mermaid-preview diagram.mmd

KEYBOARD SHORTCUTS (in preview window):
    Cmd/Ctrl+F  Search nodes      T  Toggle theme
    +/-         Zoom in/out       0  Reset zoom
    Esc         Close window      Space  Close window

EXIT CODES: 0 = success, 1 = argument error, 2 = runtime error
`)
}
