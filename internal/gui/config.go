package gui

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Config holds configuration passed from CLI mode to GUI mode via a temp JSON file.
type Config struct {
	Port       int           `json:"port"`
	Theme      string        `json:"theme"`
	Content    string        `json:"content"`
	Filename   string        `json:"filename"`
	IsMarkdown bool          `json:"is_markdown"`
	WatchFiles []string      `json:"watch_files,omitempty"`
	Poll       time.Duration `json:"poll,omitempty"`
	NoWatch    bool          `json:"no_watch"`
	Verbose    bool          `json:"verbose"`
}

// WriteConfig serializes cfg to a temp JSON file and returns the path.
func WriteConfig(cfg Config) (string, error) {
	data, err := json.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("marshal config: %w", err)
	}

	f, err := os.CreateTemp("", "mermaid-preview-cli-gui-*.json")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		os.Remove(f.Name())
		return "", fmt.Errorf("write config: %w", err)
	}

	return f.Name(), nil
}

// ReadConfig reads the config from the given path and deletes the file.
func ReadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config %s: %w", path, err)
	}
	os.Remove(path)

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("unmarshal config: %w", err)
	}
	return cfg, nil
}
