package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mxie/mermaid-preview/internal/browser"
	"github.com/mxie/mermaid-preview/internal/parser"
	"github.com/mxie/mermaid-preview/web"
)

// runOnce writes a self-contained HTML file for each input and opens it in
// the browser. The CLI exits immediately — no server, no live reload.
// This is ideal for agent tools that just want to display a diagram.
func runOnce(cfg Config) error {
	mermaidJS, err := web.Assets.ReadFile("static/mermaid.min.js")
	if err != nil {
		return fmt.Errorf("reading embedded mermaid.js: %w", err)
	}
	cssData, err := web.Assets.ReadFile("static/style.css")
	if err != nil {
		return fmt.Errorf("reading embedded style.css: %w", err)
	}

	for _, file := range cfg.Files {
		var content string
		if cfg.IsStdin {
			data, readErr := io.ReadAll(io.LimitReader(os.Stdin, maxStdinSize))
			if readErr != nil {
				return fmt.Errorf("reading stdin: %w", readErr)
			}
			content = string(data)
		} else {
			data, readErr := os.ReadFile(file)
			if readErr != nil {
				return fmt.Errorf("reading %s: %w", file, readErr)
			}
			content = string(data)
		}

		isMarkdown := strings.HasSuffix(file, ".md") || strings.HasSuffix(file, ".markdown")

		var sources []string
		if isMarkdown {
			sources = parser.ExtractMermaidBlocks(content)
		} else {
			sources = []string{content}
		}

		if len(sources) == 0 {
			return fmt.Errorf("no mermaid diagram blocks found in %s", file)
		}

		html := buildStandaloneHTML(filepath.Base(file), cfg.Theme, sources, string(mermaidJS), string(cssData))

		tmpFile, err := os.CreateTemp("", "mermaid-preview-*.html")
		if err != nil {
			return fmt.Errorf("creating temp file: %w", err)
		}
		if _, err := tmpFile.WriteString(html); err != nil {
			tmpFile.Close()
			return fmt.Errorf("writing temp file: %w", err)
		}
		tmpFile.Close()

		fmt.Fprintf(os.Stderr, "mermaid-preview: wrote %s\n", tmpFile.Name())

		if !cfg.NoBrowser {
			if err := browser.Open("file://" + tmpFile.Name()); err != nil {
				fmt.Fprintf(os.Stderr, "mermaid-preview: could not open browser: %v\n", err)
			}
		}
	}

	return nil
}

func buildStandaloneHTML(filename, theme string, sources []string, mermaidJS, css string) string {
	var b strings.Builder

	b.WriteString(`<!DOCTYPE html>
<html lang="en" data-theme="` + theme + `">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>` + filename + ` &#8212; mermaid-preview</title>
<style>` + css + `</style>
<script>` + mermaidJS + `</script>
</head>
<body>
<header id="toolbar">
  <div class="toolbar-left"><span class="filename">` + filename + `</span></div>
  <div class="toolbar-center"></div>
  <div class="toolbar-right">
    <span id="zoom-level" class="zoom-level">Fit</span>
    <button id="zoom-in" class="btn-icon" title="Zoom in (+)">+</button>
    <button id="zoom-out" class="btn-icon" title="Zoom out (-)">&#8722;</button>
    <button id="zoom-reset" class="btn-icon" title="Reset zoom (0)">&#8857;</button>
    <button id="theme-toggle" class="btn-icon" title="Toggle theme (T)">&#9680;</button>
    <div class="dropdown">
      <button id="export-btn" class="btn-icon" title="Export">&#10515;</button>
      <div id="export-menu" class="dropdown-menu hidden">
        <button id="export-svg">Export SVG</button>
        <button id="export-png">Export PNG</button>
        <button id="export-print">Print</button>
      </div>
    </div>
  </div>
</header>
<div id="error-overlay" class="error-overlay hidden">
  <div class="error-content"><strong>Parse Error</strong><pre id="error-message"></pre></div>
</div>
<main id="diagram-container" class="diagram-container">
  <div id="diagram-wrapper" class="diagram-wrapper">
    <div id="diagram"></div>
  </div>
</main>
<footer class="status-bar"><span id="status-type"></span><span id="status-nodes"></span></footer>
<script>
`)

	// Emit diagram sources as a JS array
	b.WriteString("var sources = [\n")
	for i, src := range sources {
		if i > 0 {
			b.WriteString(",\n")
		}
		b.WriteString(jsStringLiteral(src))
	}
	b.WriteString("\n];\n")

	// Self-contained rendering + UI logic (no server dependencies)
	// Note: block.innerHTML receives mermaid.render() SVG output only,
	// with securityLevel:'strict' preventing any script execution in diagrams.
	b.WriteString(`
(function() {
var diagram = document.getElementById('diagram');
var diagramContainer = document.getElementById('diagram-container');
var diagramWrapper = document.getElementById('diagram-wrapper');
var zoomLevel = document.getElementById('zoom-level');
var zoom = 1, panX = 0, panY = 0;
var isDragging = false, dragStartX, dragStartY, dragStartPanX, dragStartPanY;
var themeOrder = ['system','light','dark'];
var themeIdx = themeOrder.indexOf(localStorage.getItem('mermaid-preview-theme') || '` + theme + `');
if (themeIdx === -1) themeIdx = 0;

function effectiveTheme() {
  var t = themeOrder[themeIdx];
  return t === 'system' ? (window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light') : t;
}
function applyTheme() {
  document.documentElement.setAttribute('data-theme', themeOrder[themeIdx]);
  localStorage.setItem('mermaid-preview-theme', themeOrder[themeIdx]);
}
function cycleTheme() { themeIdx = (themeIdx + 1) % 3; applyTheme(); renderAll(); }
applyTheme();

var renderCounter = 0;
async function renderAll() {
  mermaid.initialize({
    startOnLoad: false,
    theme: effectiveTheme() === 'dark' ? 'dark' : 'default',
    securityLevel: 'strict', logLevel: 'error', suppressErrorRendering: true
  });
  diagram.textContent = '';
  var nodeCount = 0, dtype = '';
  for (var i = 0; i < sources.length; i++) {
    var src = sources[i].trim();
    if (!src) continue;
    if (sources.length > 1) {
      if (i > 0) { var d = document.createElement('div'); d.className = 'diagram-block-divider'; diagram.appendChild(d); }
      var lbl = document.createElement('div'); lbl.className = 'diagram-block-label'; lbl.textContent = 'Block ' + (i+1); diagram.appendChild(lbl);
    }
    var block = document.createElement('div'); block.className = 'diagram-block';
    try {
      var id = 'mermaid-' + (renderCounter++);
      var result = await mermaid.render(id, src);
      block.innerHTML = result.svg;
      var svgEl = block.querySelector('svg');
      if (svgEl) nodeCount += svgEl.querySelectorAll('.node,.actor,.cluster').length;
      if (i === 0) {
        var fl = src.split('\n')[0].toLowerCase();
        dtype = fl.startsWith('graph') || fl.startsWith('flowchart') ? 'Flowchart' :
          fl.startsWith('sequence') ? 'Sequence' : fl.startsWith('class') ? 'Class' :
          fl.startsWith('state') ? 'State' : fl.startsWith('er') ? 'ER' :
          fl.startsWith('gantt') ? 'Gantt' : fl.startsWith('pie') ? 'Pie' :
          fl.startsWith('git') ? 'Git Graph' : fl.startsWith('mindmap') ? 'Mindmap' :
          fl.startsWith('journey') ? 'Journey' : 'Diagram';
      }
    } catch(e) {
      document.getElementById('error-message').textContent = e.message || String(e);
      document.getElementById('error-overlay').classList.remove('hidden');
    }
    diagram.appendChild(block);
  }
  document.getElementById('status-type').textContent = dtype;
  document.getElementById('status-nodes').textContent = nodeCount > 0 ? nodeCount + ' nodes' : '';
}
renderAll();

function updateTransform() {
  diagramWrapper.style.transform = 'translate(' + panX + 'px,' + panY + 'px) scale(' + zoom + ')';
  zoomLevel.textContent = (zoom === 1 && panX === 0 && panY === 0) ? 'Fit' : Math.round(zoom * 100) + '%';
}
function zoomCentered(f) {
  var r = diagramContainer.getBoundingClientRect(), cx = r.width/2, cy = r.height/2, old = zoom;
  zoom = Math.min(Math.max(zoom * f, 0.1), 10); var s = zoom / old;
  panX = cx - s * (cx - panX); panY = cy - s * (cy - panY); updateTransform();
}

document.getElementById('zoom-in').addEventListener('click', function() { zoomCentered(1.2); });
document.getElementById('zoom-out').addEventListener('click', function() { zoomCentered(1/1.2); });
document.getElementById('zoom-reset').addEventListener('click', function() { zoom=1; panX=0; panY=0; updateTransform(); });
document.getElementById('theme-toggle').addEventListener('click', cycleTheme);

diagramContainer.addEventListener('wheel', function(e) {
  e.preventDefault();
  var r = diagramContainer.getBoundingClientRect(), mx = e.clientX - r.left, my = e.clientY - r.top, old = zoom;
  zoom = e.deltaY < 0 ? Math.min(zoom * 1.1, 10) : Math.max(zoom / 1.1, 0.1);
  var s = zoom / old; panX = mx - s * (mx - panX); panY = my - s * (my - panY); updateTransform();
}, {passive: false});

diagramContainer.addEventListener('mousedown', function(e) {
  if (e.target.closest('button')) return;
  isDragging = true; dragStartX = e.clientX; dragStartY = e.clientY;
  dragStartPanX = panX; dragStartPanY = panY;
  diagramContainer.classList.add('dragging'); e.preventDefault();
});
document.addEventListener('mousemove', function(e) {
  if (!isDragging) return;
  panX = dragStartPanX + (e.clientX - dragStartX);
  panY = dragStartPanY + (e.clientY - dragStartY);
  updateTransform();
});
document.addEventListener('mouseup', function() {
  if (!isDragging) return; isDragging = false;
  diagramContainer.classList.remove('dragging');
});

var exportMenu = document.getElementById('export-menu');
document.getElementById('export-btn').addEventListener('click', function(e) { e.stopPropagation(); exportMenu.classList.toggle('hidden'); });
document.addEventListener('click', function(e) { if (!e.target.closest('.dropdown')) exportMenu.classList.add('hidden'); });
document.getElementById('export-svg').addEventListener('click', function() {
  var svg = diagram.querySelector('svg'); if (!svg) return;
  var s = new XMLSerializer().serializeToString(svg);
  var a = document.createElement('a'); a.href = URL.createObjectURL(new Blob([s], {type: 'image/svg+xml'}));
  a.download = 'diagram.svg'; a.click(); exportMenu.classList.add('hidden');
});
document.getElementById('export-png').addEventListener('click', function() {
  var svg = diagram.querySelector('svg'); if (!svg) return;
  var s = new XMLSerializer().serializeToString(svg);
  var url = URL.createObjectURL(new Blob([s], {type: 'image/svg+xml;charset=utf-8'}));
  var img = new Image();
  img.onload = function() {
    var sc = window.devicePixelRatio || 1, c = document.createElement('canvas');
    c.width = img.width * sc; c.height = img.height * sc;
    var ctx = c.getContext('2d'); ctx.scale(sc, sc); ctx.drawImage(img, 0, 0);
    c.toBlob(function(b) { var a = document.createElement('a'); a.href = URL.createObjectURL(b); a.download = 'diagram.png'; a.click(); });
    URL.revokeObjectURL(url);
  };
  img.src = url; exportMenu.classList.add('hidden');
});
document.getElementById('export-print').addEventListener('click', function() { exportMenu.classList.add('hidden'); window.print(); });

document.addEventListener('keydown', function(e) {
  if (e.key === 't' || e.key === 'T') { cycleTheme(); return; }
  if (e.key === '+' || e.key === '=') { zoomCentered(1.2); return; }
  if (e.key === '-') { zoomCentered(1/1.2); return; }
  if (e.key === '0') { zoom = 1; panX = 0; panY = 0; updateTransform(); return; }
});
})();
</script></body></html>`)

	return b.String()
}

// jsStringLiteral wraps s in JS template literal backticks, escaping special chars.
func jsStringLiteral(s string) string {
	var b strings.Builder
	b.WriteByte('`')
	for _, r := range s {
		switch r {
		case '`':
			b.WriteString("\\`")
		case '\\':
			b.WriteString("\\\\")
		case '$':
			b.WriteString("\\$")
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('`')
	return b.String()
}
