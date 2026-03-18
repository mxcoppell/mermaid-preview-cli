package server

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/mxcoppell/mmdp/internal/parser"
	"github.com/mxcoppell/mmdp/web"
)

// Config holds server configuration.
type Config struct {
	Port       int
	Theme      string
	Content    string
	Filename   string
	IsMarkdown bool
	Verbose    bool
	Label      string // display name for window badge
	ColorHex   string // hex color for window badge (e.g. "#FF453A")
}

// Server is the HTTP server for mmdp.
type Server struct {
	cfg        Config
	mu         sync.RWMutex
	content    string
	blocks     []string // parsed mermaid blocks (for markdown files)
	srv        *http.Server
	ws         *WSHub
	cancel     context.CancelFunc
	done       chan struct{}
	listener   net.Listener
	OnShutdown func() // called before server shutdown (e.g. to terminate webview)
}

// New creates a new Server.
func New(cfg Config) *Server {
	s := &Server{
		cfg:  cfg,
		done: make(chan struct{}),
	}
	s.content = cfg.Content
	if cfg.IsMarkdown {
		s.blocks = parser.ExtractMermaidBlocks(cfg.Content)
	}
	s.ws = newWSHub(s)
	return s
}

// Start starts the HTTP server and returns the listening address.
func (s *Server) Start(ctx context.Context) (string, error) {
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	addr := "127.0.0.1:0"
	if s.cfg.Port > 0 {
		addr = fmt.Sprintf("127.0.0.1:%d", s.cfg.Port)
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		cancel()
		return "", fmt.Errorf("listen: %w", err)
	}
	s.listener = ln

	mux := http.NewServeMux()
	s.registerRoutes(mux)

	s.srv = &http.Server{Handler: mux}

	go func() {
		defer close(s.done)
		if err := s.srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			fmt.Printf("mmdp: server error: %v\n", err)
		}
	}()

	// Graceful shutdown on context cancellation
	go func() {
		<-ctx.Done()
		shutCtx, shutCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer shutCancel()
		_ = s.srv.Shutdown(shutCtx)
	}()

	return ln.Addr().String(), nil
}

// Wait blocks until the server has shut down.
func (s *Server) Wait() {
	<-s.done
}

// Shutdown initiates graceful shutdown.
func (s *Server) Shutdown() {
	if s.cancel != nil {
		s.cancel()
	}
}

// UpdateContent updates the diagram content and broadcasts to WebSocket clients.
func (s *Server) UpdateContent(content string, blocks []string) {
	s.mu.Lock()
	s.content = content
	if blocks != nil {
		s.blocks = blocks
	} else if s.cfg.IsMarkdown {
		s.blocks = parser.ExtractMermaidBlocks(content)
	}
	s.mu.Unlock()

	s.ws.broadcast(content)
}

// Addr returns the listener address, or empty string if not started.
func (s *Server) Addr() string {
	if s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

func (s *Server) registerRoutes(mux *http.ServeMux) {
	// Serve embedded static files
	staticFS, _ := fs.Sub(web.Assets, "static")
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// HTML template
	mux.HandleFunc("/", s.handleIndex)

	// API endpoints
	mux.HandleFunc("/api/diagram", s.handleDiagram)
	mux.HandleFunc("/api/shutdown", s.handleShutdown)

	// WebSocket
	mux.HandleFunc("/ws", s.ws.handleWS)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	tmplData, err := web.Assets.ReadFile("templates/index.html")
	if err != nil {
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}

	tmpl, err := template.New("index").Parse(string(tmplData))
	if err != nil {
		http.Error(w, "template parse error", http.StatusInternalServerError)
		return
	}

	s.mu.RLock()
	content := s.content
	blocks := s.blocks
	s.mu.RUnlock()

	// For markdown files, pass the parsed blocks as JSON
	// Use template.JS to avoid HTML-escaping the JSON
	var blocksJSON template.JS = "null"
	if s.cfg.IsMarkdown && len(blocks) > 0 {
		b, _ := json.Marshal(blocks)
		blocksJSON = template.JS(b)
	}

	data := map[string]any{
		"Theme":      s.cfg.Theme,
		"Content":    content,
		"Filename":   s.cfg.Filename,
		"IsMarkdown": s.cfg.IsMarkdown,
		"Blocks":     blocksJSON,
		"NoWatch":    false,
		"Label":      s.cfg.Label,
		"ColorHex":   s.cfg.ColorHex,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tmpl.Execute(w, data)
}

func (s *Server) handleDiagram(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	content := s.content
	s.mu.RUnlock()

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(content))
}

func (s *Server) handleShutdown(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("shutting down"))

	// Shutdown asynchronously so the response is sent
	go func() {
		time.Sleep(100 * time.Millisecond)
		if s.OnShutdown != nil {
			s.OnShutdown()
		}
		s.Shutdown()
	}()
}
