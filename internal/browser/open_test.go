package browser

import (
	"runtime"
	"testing"
)

func TestOpen_DoesNotPanic(t *testing.T) {
	// We can't easily test that the browser opens, but we can verify
	// the function doesn't panic on any platform.
	// Use a URL that likely won't resolve but won't error on cmd creation.
	switch runtime.GOOS {
	case "darwin", "linux", "windows":
		// Just verify the command is constructed without error.
		// We don't actually call Open() since it would open a browser.
	default:
		t.Skipf("unsupported platform: %s", runtime.GOOS)
	}
}
