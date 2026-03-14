package server

import (
	"context"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func TestWebSocket_ConnectAndReceive(t *testing.T) {
	srv := New(Config{
		Theme:    "system",
		Content:  "graph LR; A-->B",
		Filename: "test.mmd",
	})

	ctx := t.Context()

	addr, err := srv.Start(ctx)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer srv.Shutdown()

	// Connect WebSocket
	wsURL := "ws://" + addr + "/ws"
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Allow server goroutine to register the connection
	time.Sleep(50 * time.Millisecond)

	// Update content and verify broadcast
	srv.UpdateContent("graph TD; X-->Y", nil)

	readCtx, readCancel := context.WithTimeout(ctx, 2*time.Second)
	defer readCancel()

	_, msg, err := conn.Read(readCtx)
	if err != nil {
		t.Fatalf("WebSocket read: %v", err)
	}
	if string(msg) != "graph TD; X-->Y" {
		t.Errorf("WebSocket message = %q, want %q", string(msg), "graph TD; X-->Y")
	}
}

func TestWebSocket_MultipleClients(t *testing.T) {
	srv := New(Config{
		Theme:    "system",
		Content:  "graph LR; A-->B",
		Filename: "test.mmd",
	})

	ctx := t.Context()

	addr, err := srv.Start(ctx)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer srv.Shutdown()

	wsURL := "ws://" + addr + "/ws"

	// Connect two clients
	conn1, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial 1: %v", err)
	}
	defer conn1.Close(websocket.StatusNormalClosure, "")

	conn2, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial 2: %v", err)
	}
	defer conn2.Close(websocket.StatusNormalClosure, "")

	// Allow server goroutine to register the connections
	time.Sleep(50 * time.Millisecond)

	// Broadcast should reach both
	srv.UpdateContent("updated content", nil)

	readCtx, readCancel := context.WithTimeout(ctx, 2*time.Second)
	defer readCancel()

	_, msg1, err := conn1.Read(readCtx)
	if err != nil {
		t.Fatalf("read conn1: %v", err)
	}
	if string(msg1) != "updated content" {
		t.Errorf("conn1 msg = %q", string(msg1))
	}

	_, msg2, err := conn2.Read(readCtx)
	if err != nil {
		t.Fatalf("read conn2: %v", err)
	}
	if string(msg2) != "updated content" {
		t.Errorf("conn2 msg = %q", string(msg2))
	}
}
