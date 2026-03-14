package gui

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/mxie/mermaid-preview-cli/internal/parser"
	"github.com/mxie/mermaid-preview-cli/internal/server"
	"github.com/mxie/mermaid-preview-cli/internal/watcher"
)

// Run is the GUI process entry point. It reads the config from the temp file,
// starts an HTTP server, optionally starts file watchers, creates a frameless
// webview window, and runs the event loop until the window is closed.
func Run(cfgPath string) error {
	cfg, err := ReadConfig(cfgPath)
	if err != nil {
		return fmt.Errorf("reading config: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv := server.New(server.Config{
		Port:       cfg.Port,
		Theme:      cfg.Theme,
		Content:    cfg.Content,
		Filename:   cfg.Filename,
		IsMarkdown: cfg.IsMarkdown,
	})

	addr, err := srv.Start(ctx)
	if err != nil {
		return fmt.Errorf("starting server: %w", err)
	}
	url := fmt.Sprintf("http://%s", addr)
	fmt.Fprintf(os.Stderr, "mermaid-preview-cli: listening on %s (%s)\n", url, cfg.Filename)

	// Start file watchers
	if !cfg.NoWatch && len(cfg.WatchFiles) > 0 {
		for _, file := range cfg.WatchFiles {
			isMarkdown := strings.HasSuffix(file, ".md") || strings.HasSuffix(file, ".markdown")
			absPath, err := filepath.Abs(file)
			if err != nil {
				fmt.Fprintf(os.Stderr, "mermaid-preview-cli: resolve path error (%s): %v\n", file, err)
				continue
			}

			var w watcher.Watcher
			if cfg.Poll > 0 {
				w = watcher.NewPollWatcher(absPath, cfg.Poll)
			} else {
				w, err = watcher.NewFileWatcher(absPath)
				if err != nil {
					fmt.Fprintf(os.Stderr, "mermaid-preview-cli: watcher error (%s): %v\n", file, err)
					continue
				}
			}

			go func() {
				if err := w.Start(ctx); err != nil {
					fmt.Fprintf(os.Stderr, "mermaid-preview-cli: watcher error: %v\n", err)
				}
			}()

			go func(isMD bool) {
				for newContent := range w.Content() {
					if isMD {
						blocks := parser.ExtractMermaidBlocks(newContent)
						srv.UpdateContent(newContent, blocks)
					} else {
						srv.UpdateContent(newContent, nil)
					}
				}
			}(isMarkdown)
		}
	}

	// Create webview window
	w := createWindow(url)
	defer w.Destroy()

	// Wire server shutdown → webview terminate
	srv.OnShutdown = func() {
		w.Terminate()
	}

	// Handle SIGINT and SIGTERM gracefully
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		w.Terminate()
	}()

	// If the server exits for any reason (e.g. auto-shutdown after 30s
	// with no WebSocket clients), terminate the webview so the process
	// doesn't linger invisibly (no dock icon).
	go func() {
		srv.Wait()
		w.Terminate()
	}()

	// Schedule frameless styling before starting the event loop.
	// Registers a CFRunLoopTimer that fires once the run loop is active.
	scheduleFrameless(w.Window())

	// Run webview event loop (blocks until window is closed)
	w.Run()

	// Clean up
	fmt.Fprintf(os.Stderr, "mermaid-preview-cli: shutting down\n")
	srv.Shutdown()
	srv.Wait()
	return nil
}
