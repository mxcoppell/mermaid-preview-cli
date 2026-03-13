package browser

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// chromeCandidates returns Chromium-based browser paths to try, in priority order.
func chromeCandidates() []string {
	switch runtime.GOOS {
	case "darwin":
		return []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Google Chrome Canary.app/Contents/MacOS/Google Chrome Canary",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
			"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
			"/Applications/Brave Browser.app/Contents/MacOS/Brave Browser",
		}
	case "windows":
		localApp := os.Getenv("LOCALAPPDATA")
		progFiles := os.Getenv("ProgramFiles")
		progFiles86 := os.Getenv("ProgramFiles(x86)")
		return []string{
			filepath.Join(localApp, `Google\Chrome\Application\chrome.exe`),
			filepath.Join(progFiles, `Google\Chrome\Application\chrome.exe`),
			filepath.Join(progFiles86, `Google\Chrome\Application\chrome.exe`),
			filepath.Join(progFiles, `Microsoft\Edge\Application\msedge.exe`),
			filepath.Join(progFiles86, `Microsoft\Edge\Application\msedge.exe`),
		}
	default: // linux
		return []string{
			"google-chrome",
			"google-chrome-stable",
			"chromium-browser",
			"chromium",
			"/opt/google/chrome/google-chrome",
			"/snap/bin/chromium",
			"microsoft-edge",
		}
	}
}

// findChrome returns the path to the first available Chromium-based browser.
func findChrome() string {
	for _, candidate := range chromeCandidates() {
		if runtime.GOOS == "linux" {
			// On Linux, candidates may be bare names — check PATH
			if p, err := exec.LookPath(candidate); err == nil {
				return p
			}
		} else {
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
		}
	}
	return ""
}

// Open opens the specified URL in a standalone app-mode window (no address bar,
// no tabs) using a Chromium-based browser. Falls back to the system default
// browser if no Chromium browser is found.
func Open(url string) error {
	if chrome := findChrome(); chrome != "" {
		return openAppWindow(chrome, url)
	}
	return openDefault(url)
}

// openAppWindow launches Chrome/Chromium in --app mode with a temporary profile
// so the flags are guaranteed to take effect.
func openAppWindow(chromePath, url string) error {
	profileDir, err := os.MkdirTemp("", "mermaid-preview-chrome-*")
	if err != nil {
		return openDefault(url)
	}

	args := []string{
		"--app=" + url,
		"--window-size=1200,800",
		"--user-data-dir=" + profileDir,
		"--no-first-run",
		"--no-default-browser-check",
	}

	cmd := exec.Command(chromePath, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Start()
}

// openDefault falls back to the system default browser.
func openDefault(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}
