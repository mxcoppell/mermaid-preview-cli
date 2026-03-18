package ipc

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"
)

// OpenRequest is sent from CLI to host to open a new window.
type OpenRequest struct {
	ConfigPath string `json:"config_path"`
}

// OpenResponse is sent from host back to CLI.
type OpenResponse struct {
	OK       bool   `json:"ok"`
	WindowID string `json:"window_id,omitempty"`
	Error    string `json:"error,omitempty"`
	Reused   bool   `json:"reused,omitempty"`
}

// SocketPath returns the Unix socket path for IPC, scoped to the current user.
func SocketPath() string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("mmdp-%d.sock", os.Getuid()))
}

// Dial connects to the host process with a timeout.
func Dial() (net.Conn, error) {
	return net.DialTimeout("unix", SocketPath(), 500*time.Millisecond)
}

// IsHostRunning checks if a host process is listening on the socket.
func IsHostRunning() bool {
	conn, err := Dial()
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// CleanStaleSocket removes the socket file if no host is listening.
func CleanStaleSocket() {
	path := SocketPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return
	}
	if !IsHostRunning() {
		os.Remove(path)
	}
}

// SendOpen sends an open request to the host and reads the response.
func SendOpen(conn net.Conn, cfgPath string) (OpenResponse, error) {
	req := OpenRequest{ConfigPath: cfgPath}
	data, err := json.Marshal(req)
	if err != nil {
		return OpenResponse{}, fmt.Errorf("marshal request: %w", err)
	}
	data = append(data, '\n')

	if err := conn.SetWriteDeadline(time.Now().Add(2 * time.Second)); err != nil {
		return OpenResponse{}, fmt.Errorf("set write deadline: %w", err)
	}
	if _, err := conn.Write(data); err != nil {
		return OpenResponse{}, fmt.Errorf("write request: %w", err)
	}

	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return OpenResponse{}, fmt.Errorf("set read deadline: %w", err)
	}
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		return OpenResponse{}, fmt.Errorf("read response: %w", err)
	}

	var resp OpenResponse
	if err := json.Unmarshal(buf[:n], &resp); err != nil {
		return OpenResponse{}, fmt.Errorf("unmarshal response: %w", err)
	}
	return resp, nil
}
