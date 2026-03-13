package cmd

import (
	"os"
	"testing"
	"time"
)

func TestParseFlags_SingleFile(t *testing.T) {
	cfg, err := parseFlags([]string{"diagram.mmd"}, devNull(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Files) != 1 || cfg.Files[0] != "diagram.mmd" {
		t.Errorf("Files = %v, want [diagram.mmd]", cfg.Files)
	}
	if cfg.IsStdin {
		t.Error("expected IsStdin = false")
	}
}

func TestParseFlags_MultipleFiles(t *testing.T) {
	cfg, err := parseFlags([]string{"a.mmd", "b.mmd", "c.md"}, devNull(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Files) != 3 {
		t.Fatalf("Files count = %d, want 3", len(cfg.Files))
	}
	if cfg.Files[0] != "a.mmd" || cfg.Files[1] != "b.mmd" || cfg.Files[2] != "c.md" {
		t.Errorf("Files = %v", cfg.Files)
	}
}

func TestParseFlags_Defaults(t *testing.T) {
	cfg, err := parseFlags([]string{"test.mmd"}, devNull(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != 0 {
		t.Errorf("Port = %d, want 0", cfg.Port)
	}
	if cfg.NoBrowser {
		t.Error("expected NoBrowser = false")
	}
	if cfg.Theme != "system" {
		t.Errorf("Theme = %q, want %q", cfg.Theme, "system")
	}
	if cfg.NoWatch {
		t.Error("expected NoWatch = false")
	}
}

func TestParseFlags_AllFlags(t *testing.T) {
	cfg, err := parseFlags([]string{
		"--port", "8080",
		"--no-browser",
		"--theme", "dark",
		"--no-watch",
		"--poll", "500ms",
		"test.mmd",
	}, devNull(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want 8080", cfg.Port)
	}
	if !cfg.NoBrowser {
		t.Error("expected NoBrowser = true")
	}
	if cfg.Theme != "dark" {
		t.Errorf("Theme = %q, want %q", cfg.Theme, "dark")
	}
	if !cfg.NoWatch {
		t.Error("expected NoWatch = true")
	}
	if cfg.Poll != 500*time.Millisecond {
		t.Errorf("Poll = %v, want 500ms", cfg.Poll)
	}
}

func TestParseFlags_ShortFlags(t *testing.T) {
	cfg, err := parseFlags([]string{"-p", "9090", "-b", "-t", "light", "-w", "test.mmd"}, devNull(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != 9090 {
		t.Errorf("Port = %d, want 9090", cfg.Port)
	}
	if !cfg.NoBrowser {
		t.Error("expected NoBrowser = true")
	}
	if cfg.Theme != "light" {
		t.Errorf("Theme = %q, want %q", cfg.Theme, "light")
	}
}

func TestParseFlags_InvalidTheme(t *testing.T) {
	_, err := parseFlags([]string{"--theme", "invalid", "test.mmd"}, devNull(t))
	if err == nil {
		t.Error("expected error for invalid theme")
	}
}

func TestParseFlags_NoArgs_Stdin(t *testing.T) {
	cfg, err := parseFlags([]string{}, devNull(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.IsStdin {
		t.Error("expected IsStdin = true")
	}
	if !cfg.Once {
		t.Error("expected Once = true (stdin default)")
	}
	if len(cfg.Files) != 1 || cfg.Files[0] != "<stdin>" {
		t.Errorf("Files = %v, want [<stdin>]", cfg.Files)
	}
}

func TestParseFlags_Stdin_ServeOverride(t *testing.T) {
	cfg, err := parseFlags([]string{"--serve"}, devNull(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.IsStdin {
		t.Error("expected IsStdin = true")
	}
	if cfg.Once {
		t.Error("expected Once = false when --serve is set")
	}
}

func TestParseFlags_FileWithOnce(t *testing.T) {
	cfg, err := parseFlags([]string{"--once", "test.mmd"}, devNull(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.Once {
		t.Error("expected Once = true")
	}
	if cfg.IsStdin {
		t.Error("expected IsStdin = false")
	}
}

// devNull returns a file that is not a terminal (simulates piped input).
func devNull(t *testing.T) *os.File {
	t.Helper()
	f, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.Close() })
	return f
}
