package gui

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	webview "github.com/webview/webview_go"

	"github.com/mxcoppell/mermaid-preview-cli/internal/ipc"
	"github.com/mxcoppell/mermaid-preview-cli/internal/parser"
	"github.com/mxcoppell/mermaid-preview-cli/internal/server"
	"github.com/mxcoppell/mermaid-preview-cli/internal/watcher"
)

// activeHost is the package-level host reference needed by CGO callbacks.
var activeHost *Host

// Host is the single process that manages all preview windows.
type Host struct {
	mu         sync.Mutex
	windows    map[string]*WindowEntry
	nextID     int
	stdinCount int
	primaryWV  webview.WebView
	ipcSrv     *ipc.Server
	ctx        context.Context
	cancel     context.CancelFunc
	verbose    bool
}

// WindowEntry tracks a single preview window and its resources.
type WindowEntry struct {
	ID         string
	Filename   string
	Label      string // display name: filename or "Diagram N"
	ColorIndex int    // 0-7 palette index
	Webview    webview.WebView
	Server     *server.Server
	Cancel     context.CancelFunc
}

// RunHost is the host process entry point. It reads the initial config,
// sets up the dock icon, starts IPC, creates the first window, and runs
// the NSApp event loop.
func RunHost(cfgPath string) error {
	cfg, err := ReadConfig(cfgPath)
	if err != nil {
		return fmt.Errorf("reading config: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	h := &Host{
		windows: make(map[string]*WindowEntry),
		ctx:     ctx,
		cancel:  cancel,
		verbose: cfg.Verbose,
	}
	activeHost = h

	// Initialize as regular app (dock icon visible)
	initHostMode()

	// Start IPC server
	ipcSrv, err := ipc.NewServer(h.handleIPC)
	if err != nil {
		return fmt.Errorf("starting IPC server: %w", err)
	}
	h.ipcSrv = ipcSrv
	go ipcSrv.Serve()

	// Create the first window (also sets primaryWV)
	if err := h.openWindowFromConfig(cfg); err != nil {
		ipcSrv.Close()
		return fmt.Errorf("creating first window: %w", err)
	}

	// Signal handler
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		h.primaryWV.Dispatch(func() {
			h.closeAllWindows()
		})
	}()

	// Run the NSApp event loop (blocks until Terminate)
	h.primaryWV.Run()

	// Cleanup
	if h.verbose {
		fmt.Fprintf(os.Stderr, "mermaid-preview-cli: shutting down\n")
	}
	ipcSrv.Close()
	h.shutdownAllServers()
	return nil
}

// handleIPC processes an incoming IPC request from a CLI process.
func (h *Host) handleIPC(req ipc.OpenRequest) ipc.OpenResponse {
	cfg, err := ReadConfig(req.ConfigPath)
	if err != nil {
		return ipc.OpenResponse{Error: fmt.Sprintf("read config: %v", err)}
	}

	done := make(chan string, 1)
	errCh := make(chan error, 1)
	h.primaryWV.Dispatch(func() {
		id, err := h.createWindow(cfg)
		if err != nil {
			errCh <- err
			return
		}
		done <- id
	})

	select {
	case id := <-done:
		return ipc.OpenResponse{OK: true, WindowID: id}
	case err := <-errCh:
		return ipc.OpenResponse{Error: err.Error()}
	}
}

// openWindowFromConfig creates a window from a Config.
func (h *Host) openWindowFromConfig(cfg Config) error {
	_, err := h.createWindow(cfg)
	return err
}

// createWindow creates a webview + server + watchers for a config.
// Must be called on the main thread (or before Run() for the first window).
func (h *Host) createWindow(cfg Config) (string, error) {
	h.mu.Lock()
	h.nextID++
	id := fmt.Sprintf("w-%d", h.nextID)
	colorIndex := (h.nextID - 1) % PaletteSize

	// Derive display label
	label := cfg.Filename
	if label == "" || label == "." {
		h.stdinCount++
		label = fmt.Sprintf("Diagram %d", h.stdinCount)
	}
	h.mu.Unlock()

	wCtx, wCancel := context.WithCancel(h.ctx)

	srv := server.New(server.Config{
		Port:       cfg.Port,
		Theme:      cfg.Theme,
		Content:    cfg.Content,
		Filename:   cfg.Filename,
		IsMarkdown: cfg.IsMarkdown,
		Verbose:    h.verbose,
		Label:      label,
		ColorHex:   Palette[colorIndex].Hex,
	})

	addr, err := srv.Start(wCtx)
	if err != nil {
		wCancel()
		return "", fmt.Errorf("starting server: %w", err)
	}
	url := fmt.Sprintf("http://%s", addr)
	fmt.Fprintf(os.Stderr, "mermaid-preview-cli: listening on %s (%s)\n", url, label)

	// Start file watchers
	h.startFileWatchers(wCtx, cfg, srv)

	// Create webview
	w := webview.New(false)
	hideWindowOffscreen(w.Window())
	w.SetTitle("mermaid-preview-cli")
	w.SetSize(1400, 1000, webview.HintNone)

	// Bind window management functions
	_ = w.Bind("moveWindowBy", func(dx, dy float64) {
		w.Dispatch(func() {
			moveWindowBy(w.Window(), int(dx), int(dy))
		})
	})

	_ = w.Bind("showWindow", func(width, height int) {
		w.Dispatch(func() {
			showWindow(w.Window(), width, height)
		})
	})

	_ = w.Bind("resizeWindow", func(width, height int) {
		w.Dispatch(func() {
			w.SetSize(width, height, webview.HintNone)
			centerWindow(w.Window())
		})
	})

	// closeThisWindow binding — routes through host
	windowID := id
	_ = w.Bind("closeThisWindow", func() {
		w.Dispatch(func() {
			h.CloseWindow(windowID)
		})
	})

	_ = w.Bind("saveFileDialog", func(suggestedName, base64Data, extension string) bool {
		data, err := decodeBase64(base64Data)
		if err != nil {
			return false
		}
		return saveFile(w.Window(), suggestedName, data, extension)
	})

	w.Navigate(url)

	// Wire server shutdown → close this window (not terminate app)
	srv.OnShutdown = func() {
		h.primaryWV.Dispatch(func() {
			h.CloseWindow(windowID)
		})
	}

	entry := &WindowEntry{
		ID:         id,
		Filename:   cfg.Filename,
		Label:      label,
		ColorIndex: colorIndex,
		Webview:    w,
		Server:     srv,
		Cancel:     wCancel,
	}

	h.mu.Lock()
	h.windows[id] = entry
	if h.primaryWV == nil {
		h.primaryWV = w
	}
	h.mu.Unlock()

	// Apply frameless styling directly
	applyFramelessDirect(w.Window())

	return id, nil
}

// CloseWindow closes a single window and its resources.
func (h *Host) CloseWindow(id string) {
	h.mu.Lock()
	entry, ok := h.windows[id]
	if !ok {
		h.mu.Unlock()
		return
	}
	delete(h.windows, id)
	remaining := len(h.windows)
	isPrimary := entry.Webview == h.primaryWV
	h.mu.Unlock()

	entry.Cancel()
	entry.Server.Shutdown()
	closeWindow(entry.Webview.Window())

	if remaining == 0 {
		stopRunLoop()
	} else if isPrimary {
		h.mu.Lock()
		for _, e := range h.windows {
			h.primaryWV = e.Webview
			break
		}
		h.mu.Unlock()
	}
}

// closeAllWindows closes every window.
func (h *Host) closeAllWindows() {
	h.mu.Lock()
	ids := make([]string, 0, len(h.windows))
	for id := range h.windows {
		ids = append(ids, id)
	}
	h.mu.Unlock()

	for _, id := range ids {
		h.CloseWindow(id)
	}
}

// shutdownAllServers ensures all servers are fully stopped after the run loop exits.
func (h *Host) shutdownAllServers() {
	h.mu.Lock()
	entries := make([]*WindowEntry, 0, len(h.windows))
	for _, e := range h.windows {
		entries = append(entries, e)
	}
	h.mu.Unlock()

	for _, e := range entries {
		e.Cancel()
		e.Server.Shutdown()
		e.Server.Wait()
	}
}

// ActivateWindow brings a window to the front.
func (h *Host) ActivateWindow(id string) {
	h.mu.Lock()
	entry, ok := h.windows[id]
	h.mu.Unlock()
	if !ok {
		return
	}
	activateWindow(entry.Webview.Window())
}

// WindowList returns a snapshot of all window entries.
func (h *Host) WindowList() []WindowEntry {
	h.mu.Lock()
	defer h.mu.Unlock()
	list := make([]WindowEntry, 0, len(h.windows))
	for _, e := range h.windows {
		list = append(list, *e)
	}
	return list
}

// OpenFile reads a mermaid or markdown file and opens a new window.
func (h *Host) OpenFile(path string) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mermaid-preview-cli: resolve path: %v\n", err)
		return
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mermaid-preview-cli: read file: %v\n", err)
		return
	}

	content := string(data)
	isMarkdown := strings.HasSuffix(absPath, ".md") || strings.HasSuffix(absPath, ".markdown")

	if isMarkdown {
		blocks := parser.ExtractMermaidBlocks(content)
		if len(blocks) == 0 {
			fmt.Fprintf(os.Stderr, "mermaid-preview-cli: no mermaid blocks found in %s\n", filepath.Base(absPath))
			return
		}
	}

	cfg := Config{
		Theme:      "system",
		Content:    content,
		Filename:   filepath.Base(absPath),
		IsMarkdown: isMarkdown,
		WatchFiles: []string{absPath},
	}

	h.primaryWV.Dispatch(func() {
		if _, err := h.createWindow(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "mermaid-preview-cli: open window: %v\n", err)
		}
	})
}

// startFileWatchers starts watchers that can be cancelled per-window.
func (h *Host) startFileWatchers(ctx context.Context, cfg Config, srv *server.Server) {
	if cfg.NoWatch || len(cfg.WatchFiles) == 0 {
		return
	}

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

func decodeBase64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}
