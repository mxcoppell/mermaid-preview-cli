package cmd

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"

	"github.com/mxie/mermaid-preview/internal/browser"
	"github.com/mxie/mermaid-preview/internal/parser"
	"github.com/mxie/mermaid-preview/internal/server"
	"github.com/mxie/mermaid-preview/internal/version"
	"github.com/mxie/mermaid-preview/internal/watcher"
)

const maxStdinSize = 10 * 1024 * 1024 // 10MB

// Config holds all CLI configuration.
type Config struct {
	Port      int
	NoBrowser bool
	Theme     string
	NoWatch   bool
	Poll      time.Duration
	Once      bool
	Serve     bool
	Files     []string
	IsStdin   bool
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
	fs.BoolVar(&cfg.NoBrowser, "no-browser", false, "")
	fs.BoolVar(&cfg.NoBrowser, "b", false, "")
	fs.StringVar(&cfg.Theme, "theme", "system", "")
	fs.StringVar(&cfg.Theme, "t", "system", "")
	fs.BoolVar(&cfg.NoWatch, "no-watch", false, "")
	fs.BoolVar(&cfg.NoWatch, "w", false, "")
	fs.DurationVar(&cfg.Poll, "poll", 0, "")
	fs.BoolVar(&cfg.Once, "once", false, "")
	fs.BoolVar(&cfg.Serve, "serve", false, "")
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
		cfg.Files = []string{"<stdin>"}
		// Stdin defaults to --once (fire-and-forget, no server).
		// --serve overrides this to keep the server running.
		if !cfg.Serve {
			cfg.Once = true
		}
	} else {
		printHelp(os.Stderr)
		return Config{}, fmt.Errorf("no input file specified")
	}

	return cfg, nil
}

// startInstance starts a server + watcher + browser for a single file.
func startInstance(ctx context.Context, cfg Config, file string) (*server.Server, error) {
	// Read initial content
	var content string
	if cfg.IsStdin {
		data, err := io.ReadAll(io.LimitReader(os.Stdin, maxStdinSize))
		if err != nil {
			return nil, fmt.Errorf("reading stdin: %w", err)
		}
		content = string(data)
	} else {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", file, err)
		}
		content = string(data)
	}

	isMarkdown := strings.HasSuffix(file, ".md") || strings.HasSuffix(file, ".markdown")

	srv := server.New(server.Config{
		Port:       cfg.Port,
		Theme:      cfg.Theme,
		Content:    content,
		Filename:   filepath.Base(file),
		IsMarkdown: isMarkdown,
	})

	addr, err := srv.Start(ctx)
	if err != nil {
		return nil, fmt.Errorf("starting server for %s: %w", file, err)
	}
	url := fmt.Sprintf("http://%s", addr)
	fmt.Fprintf(os.Stderr, "mermaid-preview: listening on %s (%s)\n", url, filepath.Base(file))

	// Start file watcher
	if !cfg.NoWatch && !cfg.IsStdin {
		absPath, err := filepath.Abs(file)
		if err != nil {
			return nil, fmt.Errorf("resolving path: %w", err)
		}

		var w watcher.Watcher
		if cfg.Poll > 0 {
			w = watcher.NewPollWatcher(absPath, cfg.Poll)
		} else {
			w, err = watcher.NewFileWatcher(absPath)
			if err != nil {
				return nil, fmt.Errorf("setting up watcher for %s: %w", file, err)
			}
		}

		go func() {
			if err := w.Start(ctx); err != nil {
				fmt.Fprintf(os.Stderr, "mermaid-preview: watcher error (%s): %v\n", file, err)
			}
		}()

		go func() {
			for newContent := range w.Content() {
				if isMarkdown {
					blocks := parser.ExtractMermaidBlocks(newContent)
					srv.UpdateContent(newContent, blocks)
				} else {
					srv.UpdateContent(newContent, nil)
				}
			}
		}()
	}

	if !cfg.NoBrowser {
		if err := browser.Open(url); err != nil {
			fmt.Fprintf(os.Stderr, "mermaid-preview: could not open browser for %s: %v\n", file, err)
		}
	}

	return srv, nil
}

func run(cfg Config) error {
	// --once mode: write self-contained HTML, open browser, exit immediately
	if cfg.Once {
		return runOnce(cfg)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	servers := make([]*server.Server, 0, len(cfg.Files))

	for _, file := range cfg.Files {
		srv, err := startInstance(ctx, cfg, file)
		if err != nil {
			// Shut down any servers we already started
			for _, s := range servers {
				s.Shutdown()
			}
			return err
		}
		servers = append(servers, srv)
	}

	// Wait for all servers to shut down.
	// Any one shutting down (Esc, tab close) cancels the shared context,
	// which triggers the rest to shut down too.
	var wg sync.WaitGroup
	wg.Add(len(servers))
	for _, srv := range servers {
		go func(s *server.Server) {
			defer wg.Done()
			s.Wait()
		}(srv)
	}
	wg.Wait()

	fmt.Fprintf(os.Stderr, "mermaid-preview: shutting down\n")
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
    -b, --no-browser      Don't auto-open browser
    -t, --theme THEME     dark | light | system (default: system)
    -w, --no-watch        Disable file watching
        --poll INTERVAL   Polling fallback for WSL/Docker/NFS (e.g. 500ms)
        --once            Render to a self-contained HTML file and exit
                          (no server, no live reload — the default for stdin)
        --serve           Force server mode for stdin (override --once default)
    -v, --version         Print version
    -h, --help            Print help

STDIN (fire-and-forget by default — CLI exits immediately):
    echo "graph LR; A-->B" | mermaid-preview
    cat diagram.mmd | mermaid-preview

AGENT TOOL USAGE:
    Pipe mermaid source to stdin. The CLI renders a self-contained HTML
    preview, opens the browser, and exits immediately (exit code 0).
    No server process is left running. No temp files need cleanup.

    Example from an LLM agent:
        echo "graph TD; A-->B-->C" | mermaid-preview

    For file-based preview with live reload (stays running):
        mermaid-preview diagram.mmd

KEYBOARD SHORTCUTS (in browser):
    Cmd/Ctrl+F  Search nodes      T  Toggle theme
    +/-         Zoom in/out       0  Reset zoom
    Esc         Close search or quit server (server mode only)

EXIT CODES: 0 = success, 1 = argument error, 2 = runtime error
`)
}
