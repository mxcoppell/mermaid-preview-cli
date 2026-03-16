package ipc

import (
	"os"
	"testing"
)

func TestSocketPath(t *testing.T) {
	path := SocketPath()
	if path == "" {
		t.Error("SocketPath should not be empty")
	}
}

func TestIsHostRunning_NoHost(t *testing.T) {
	// Clean up any stale socket
	os.Remove(SocketPath())
	if IsHostRunning() {
		t.Error("IsHostRunning should return false when no host is listening")
	}
}

func TestCleanStaleSocket_NoSocket(t *testing.T) {
	os.Remove(SocketPath())
	// Should not panic
	CleanStaleSocket()
}

func TestServerRoundTrip(t *testing.T) {
	os.Remove(SocketPath())

	srv, err := NewServer(func(req OpenRequest) OpenResponse {
		return OpenResponse{OK: true, WindowID: "test-1"}
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	defer srv.Close()
	go srv.Serve()

	if !IsHostRunning() {
		t.Fatal("IsHostRunning should return true")
	}

	conn, err := Dial()
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer conn.Close()

	resp, err := SendOpen(conn, "/tmp/test-config.json")
	if err != nil {
		t.Fatalf("SendOpen: %v", err)
	}
	if !resp.OK {
		t.Errorf("expected OK=true, got %v", resp.OK)
	}
	if resp.WindowID != "test-1" {
		t.Errorf("WindowID = %q, want %q", resp.WindowID, "test-1")
	}
}
