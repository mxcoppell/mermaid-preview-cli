package server

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestServer_StartAndServe(t *testing.T) {
	srv := New(Config{
		Theme:    "system",
		Content:  "graph LR; A-->B",
		Filename: "test.mmd",
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	addr, err := srv.Start(ctx)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	base := "http://" + addr

	// Test index page
	resp, err := http.Get(base + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("GET / status = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "mermaid.min.js") {
		t.Error("GET / missing mermaid.min.js reference")
	}
	if !strings.Contains(string(body), "test.mmd") {
		t.Error("GET / missing filename")
	}

	// Test diagram API
	resp2, err := http.Get(base + "/api/diagram")
	if err != nil {
		t.Fatalf("GET /api/diagram: %v", err)
	}
	defer resp2.Body.Close()
	body2, _ := io.ReadAll(resp2.Body)
	if string(body2) != "graph LR; A-->B" {
		t.Errorf("GET /api/diagram = %q, want %q", string(body2), "graph LR; A-->B")
	}

	// Test static files
	resp3, err := http.Get(base + "/static/style.css")
	if err != nil {
		t.Fatalf("GET /static/style.css: %v", err)
	}
	defer resp3.Body.Close()
	if resp3.StatusCode != 200 {
		t.Errorf("GET /static/style.css status = %d, want 200", resp3.StatusCode)
	}

	// Test 404
	resp4, err := http.Get(base + "/nonexistent")
	if err != nil {
		t.Fatalf("GET /nonexistent: %v", err)
	}
	defer resp4.Body.Close()
	if resp4.StatusCode != 404 {
		t.Errorf("GET /nonexistent status = %d, want 404", resp4.StatusCode)
	}

	// Test shutdown
	resp5, err := http.Post(base+"/api/shutdown", "", nil)
	if err != nil {
		t.Fatalf("POST /api/shutdown: %v", err)
	}
	defer resp5.Body.Close()
	if resp5.StatusCode != 200 {
		t.Errorf("POST /api/shutdown status = %d, want 200", resp5.StatusCode)
	}

	// Wait for shutdown
	done := make(chan struct{})
	go func() {
		srv.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Error("server did not shut down within 5s")
	}
}

func TestServer_ShutdownMethodNotAllowed(t *testing.T) {
	srv := New(Config{
		Theme:    "system",
		Content:  "graph LR; A-->B",
		Filename: "test.mmd",
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	addr, err := srv.Start(ctx)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer srv.Shutdown()

	resp, err := http.Get("http://" + addr + "/api/shutdown")
	if err != nil {
		t.Fatalf("GET /api/shutdown: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("GET /api/shutdown status = %d, want %d", resp.StatusCode, http.StatusMethodNotAllowed)
	}
}

func TestServer_UpdateContent(t *testing.T) {
	srv := New(Config{
		Theme:    "system",
		Content:  "graph LR; A-->B",
		Filename: "test.mmd",
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	addr, err := srv.Start(ctx)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer srv.Shutdown()

	srv.UpdateContent("graph TD; X-->Y", nil)

	resp, err := http.Get("http://" + addr + "/api/diagram")
	if err != nil {
		t.Fatalf("GET /api/diagram: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "graph TD; X-->Y" {
		t.Errorf("content = %q, want %q", string(body), "graph TD; X-->Y")
	}
}

func TestServer_MarkdownContent(t *testing.T) {
	mdContent := "# Test\n\n```mermaid\ngraph LR; A-->B\n```\n"
	srv := New(Config{
		Theme:      "system",
		Content:    mdContent,
		Filename:   "test.md",
		IsMarkdown: true,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	addr, err := srv.Start(ctx)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer srv.Shutdown()

	resp, err := http.Get("http://" + addr + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "isMarkdown") {
		t.Error("GET / missing isMarkdown config")
	}
}
