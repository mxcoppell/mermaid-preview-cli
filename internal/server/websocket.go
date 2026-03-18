package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// WSHub manages WebSocket connections using a simple mutex + slice pattern.
type WSHub struct {
	mu      sync.Mutex
	conns   []*websocket.Conn
	server  *Server
	timer   *time.Timer
	timerMu sync.Mutex
}

func newWSHub(s *Server) *WSHub {
	return &WSHub{server: s}
}

func (h *WSHub) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // localhost only, CORS not needed
	})
	if err != nil {
		return
	}

	h.addConn(conn)

	// Keep connection alive by reading (and discarding) messages
	ctx := r.Context()
	for {
		_, _, err := conn.Read(ctx)
		if err != nil {
			break
		}
	}

	h.removeConn(conn)
}

func (h *WSHub) addConn(conn *websocket.Conn) {
	h.mu.Lock()
	h.conns = append(h.conns, conn)
	h.mu.Unlock()

	// Cancel auto-shutdown timer if any
	h.timerMu.Lock()
	if h.timer != nil {
		h.timer.Stop()
		h.timer = nil
	}
	h.timerMu.Unlock()
}

func (h *WSHub) removeConn(conn *websocket.Conn) {
	h.mu.Lock()
	for i, c := range h.conns {
		if c == conn {
			h.conns = append(h.conns[:i], h.conns[i+1:]...)
			break
		}
	}
	remaining := len(h.conns)
	h.mu.Unlock()

	conn.Close(websocket.StatusNormalClosure, "")

	// Start auto-shutdown timer if no clients remaining
	if remaining == 0 {
		h.timerMu.Lock()
		h.timer = time.AfterFunc(30*time.Second, func() {
			h.mu.Lock()
			count := len(h.conns)
			h.mu.Unlock()
			if count == 0 {
				if h.server.cfg.Verbose {
					fmt.Fprintf(os.Stderr, "mmdp: no clients connected, shutting down\n")
				}
				h.server.Shutdown()
			}
		})
		h.timerMu.Unlock()
	}
}

func (h *WSHub) broadcast(content string) {
	h.mu.Lock()
	conns := make([]*websocket.Conn, len(h.conns))
	copy(conns, h.conns)
	h.mu.Unlock()

	for _, conn := range conns {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := conn.Write(ctx, websocket.MessageText, []byte(content))
		cancel()
		if err != nil {
			// Slow client — close and let removeConn handle cleanup
			conn.Close(websocket.StatusPolicyViolation, "slow client")
		}
	}
}
