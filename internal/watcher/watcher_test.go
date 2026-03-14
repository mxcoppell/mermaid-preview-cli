package watcher

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileWatcher_DetectsChanges(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.mmd")

	if err := os.WriteFile(path, []byte("graph LR; A-->B"), 0644); err != nil {
		t.Fatal(err)
	}

	w, err := NewFileWatcher(path)
	if err != nil {
		t.Fatal(err)
	}

	ctx := t.Context()

	go func() { _ = w.Start(ctx) }()

	// Wait a moment for watcher to settle
	time.Sleep(200 * time.Millisecond)

	// Modify the file
	if err := os.WriteFile(path, []byte("graph TD; X-->Y"), 0644); err != nil {
		t.Fatal(err)
	}

	select {
	case content := <-w.Content():
		if content != "graph TD; X-->Y" {
			t.Errorf("content = %q, want %q", content, "graph TD; X-->Y")
		}
	case <-time.After(2 * time.Second):
		t.Error("timeout waiting for file change")
	}
}

func TestPollWatcher_DetectsChanges(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.mmd")

	if err := os.WriteFile(path, []byte("graph LR; A-->B"), 0644); err != nil {
		t.Fatal(err)
	}

	w := NewPollWatcher(path, 100*time.Millisecond)

	ctx := t.Context()

	go func() { _ = w.Start(ctx) }()

	// Wait for initial stat
	time.Sleep(200 * time.Millisecond)

	// Modify the file
	if err := os.WriteFile(path, []byte("graph TD; X-->Y"), 0644); err != nil {
		t.Fatal(err)
	}

	select {
	case content := <-w.Content():
		if content != "graph TD; X-->Y" {
			t.Errorf("content = %q, want %q", content, "graph TD; X-->Y")
		}
	case <-time.After(2 * time.Second):
		t.Error("timeout waiting for file change")
	}
}

func TestNoopWatcher_NeverSends(t *testing.T) {
	w := NewNoopWatcher()

	ctx, cancel := context.WithCancel(t.Context())

	go func() { _ = w.Start(ctx) }()

	select {
	case <-w.Content():
		t.Error("NoopWatcher should not send content")
	case <-time.After(200 * time.Millisecond):
		// Expected — no content
	}

	cancel()

	// Channel should close after context cancel
	select {
	case _, ok := <-w.Content():
		if ok {
			t.Error("expected channel to be closed")
		}
	case <-time.After(time.Second):
		t.Error("channel not closed after cancel")
	}
}
