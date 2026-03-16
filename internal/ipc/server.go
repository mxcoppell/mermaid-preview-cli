package ipc

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
)

// Handler is called when a new open request arrives from a CLI client.
type Handler func(req OpenRequest) OpenResponse

// Server listens on a Unix socket for IPC requests from CLI processes.
type Server struct {
	listener net.Listener
	handler  Handler
	done     chan struct{}
	once     sync.Once
}

// NewServer creates a new IPC server that forwards requests to handler.
func NewServer(handler Handler) (*Server, error) {
	path := SocketPath()
	os.Remove(path)

	ln, err := net.Listen("unix", path)
	if err != nil {
		return nil, fmt.Errorf("listen unix %s: %w", path, err)
	}

	return &Server{
		listener: ln,
		handler:  handler,
		done:     make(chan struct{}),
	}, nil
}

// Serve accepts connections until Close is called.
func (s *Server) Serve() {
	defer close(s.done)
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	if !scanner.Scan() {
		return
	}

	var req OpenRequest
	if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
		resp := OpenResponse{Error: "invalid request"}
		data, _ := json.Marshal(resp)
		_, _ = conn.Write(append(data, '\n'))
		return
	}

	resp := s.handler(req)
	data, _ := json.Marshal(resp)
	_, _ = conn.Write(append(data, '\n'))
}

// Close stops the server and removes the socket file.
func (s *Server) Close() {
	s.once.Do(func() {
		s.listener.Close()
		os.Remove(SocketPath())
	})
}

// Wait blocks until the server has stopped.
func (s *Server) Wait() {
	<-s.done
}
