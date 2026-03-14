package gui

import (
	"encoding/base64"
	"sync"

	webview "github.com/webview/webview_go"
)

// createWindow creates a webview window pointed at the given URL.
// The window starts offscreen and hidden — JS calls showWindow() after
// rendering to reveal it fully formed (no flash).
func createWindow(url string) webview.WebView {
	w := webview.New(false)
	// Move offscreen IMMEDIATELY — before SetTitle/SetSize/Navigate
	// can trigger any visible window appearance.
	hideWindowOffscreen(w.Window())
	w.SetTitle("mermaid-preview-cli")
	w.SetSize(1400, 1000, webview.HintNone)

	// Auto-size binding with dampening
	var (
		prevW, prevH int
		mu           sync.Mutex
	)

	_ = w.Bind("resizeWindow", func(width, height int) {
		mu.Lock()
		defer mu.Unlock()

		if prevW > 0 && prevH > 0 {
			dw := intAbs(width-prevW) * 100 / prevW
			dh := intAbs(height-prevH) * 100 / prevH
			if dw < 15 && dh < 15 {
				return
			}
		}

		prevW = width
		prevH = height
		w.Dispatch(func() {
			w.SetSize(width, height, webview.HintNone)
			centerWindow(w.Window())
		})
	})

	// Window move binding (for dragging borderless window from JS)
	_ = w.Bind("moveWindowBy", func(dx, dy float64) {
		w.Dispatch(func() {
			moveWindowBy(w.Window(), int(dx), int(dy))
		})
	})

	// Reveal the window — called by JS after initial render + auto-shape.
	// Accepts width/height so resize + frameless + center + reveal happen
	// in a single atomic Dispatch — no flash.
	_ = w.Bind("showWindow", func(width, height int) {
		w.Dispatch(func() {
			showWindow(w.Window(), width, height)
		})
	})

	// Save file with native NSSavePanel dialog.
	// JS passes base64-encoded data to avoid UTF-8 issues with binary content.
	// The CGO function handles main thread dispatch internally via dispatch_sync.
	w.Bind("saveFileDialog", func(suggestedName, base64Data, extension string) bool {
		data, err := base64.StdEncoding.DecodeString(base64Data)
		if err != nil {
			return false
		}
		return saveFile(w.Window(), suggestedName, data, extension)
	})

	w.Navigate(url)
	return w
}

func intAbs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
