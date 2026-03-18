package ipc

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestSocketPath(t *testing.T) {
	path := SocketPath()
	if path == "" {
		t.Error("SocketPath should not be empty")
	}
}

func TestSocketPath_ContainsUID(t *testing.T) {
	path := SocketPath()
	uid := os.Getuid()
	expected := fmt.Sprintf("mmdp-%d.sock", uid)
	if !strings.Contains(path, expected) {
		t.Errorf("SocketPath %q should contain %q", path, expected)
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

func TestServerRoundTrip_Reused(t *testing.T) {
	os.Remove(SocketPath())

	srv, err := NewServer(func(req OpenRequest) OpenResponse {
		return OpenResponse{OK: true, WindowID: "test-1", Reused: true}
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	defer srv.Close()
	go srv.Serve()

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
		t.Errorf("expected OK=true")
	}
	if !resp.Reused {
		t.Errorf("expected Reused=true")
	}
	if resp.WindowID != "test-1" {
		t.Errorf("WindowID = %q, want %q", resp.WindowID, "test-1")
	}
}

func TestOpenResponse_JSON_Reused(t *testing.T) {
	resp := OpenResponse{OK: true, WindowID: "w-1", Reused: true}
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var got OpenResponse
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !got.Reused {
		t.Error("expected Reused=true after round-trip")
	}
}

func TestOpenResponse_JSON_ReusedOmitted(t *testing.T) {
	resp := OpenResponse{OK: true, WindowID: "w-1"}
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	// Reused should be omitted from JSON when false
	if strings.Contains(string(data), "reused") {
		t.Errorf("expected reused to be omitted, got %s", data)
	}
}

func TestNewServer_ErrHostAlreadyRunning(t *testing.T) {
	os.Remove(SocketPath())

	srv1, err := NewServer(func(req OpenRequest) OpenResponse {
		return OpenResponse{OK: true}
	})
	if err != nil {
		t.Fatalf("NewServer (first): %v", err)
	}
	defer srv1.Close()
	go srv1.Serve()

	// Second server should get ErrHostAlreadyRunning.
	_, err = NewServer(func(req OpenRequest) OpenResponse {
		return OpenResponse{OK: true}
	})
	if err != ErrHostAlreadyRunning {
		t.Errorf("expected ErrHostAlreadyRunning, got %v", err)
	}
}
