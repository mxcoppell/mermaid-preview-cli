// mmdp frontend
(function() {
    'use strict';

    const config = window.__CONFIG__;
    let currentContent = null;
    let lastValidSVG = null;
    let zoom = 1;
    let panX = 0;
    let panY = 0;
    let isFitted = true;
    let isDragging = false;
    let dragStartX = 0;
    let dragStartY = 0;
    let dragStartPanX = 0;
    let dragStartPanY = 0;
    let searchMatches = [];
    let searchIndex = -1;
    let ws = null;
    let reconnectDelay = 500;
    const MAX_RECONNECT_DELAY = 5000;

    // DOM elements
    const diagram = document.getElementById('diagram');
    const diagramContainer = document.getElementById('diagram-container');
    const diagramWrapper = document.getElementById('diagram-wrapper');
    const toolbar = document.getElementById('toolbar');
    const errorOverlay = document.getElementById('error-overlay');
    const errorMessage = document.getElementById('error-message');
    const disconnectedBanner = document.getElementById('disconnected-banner');
    const searchBar = document.getElementById('search-bar');
    const searchInput = document.getElementById('search-input');
    const searchCount = document.getElementById('search-count');
    const zoomLevel = document.getElementById('zoom-level');
    const exportMenu = document.getElementById('export-menu');

    // ─── Theme ────────────────────────────────────────────────────
    const themeOrder = ['system', 'light', 'dark'];
    let currentThemeIndex = themeOrder.indexOf(
        localStorage.getItem('mmdp-theme') || config.theme
    );
    if (currentThemeIndex === -1) currentThemeIndex = 0;

    function getEffectiveTheme() {
        const theme = themeOrder[currentThemeIndex];
        if (theme === 'system') {
            return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
        }
        return theme;
    }

    function applyTheme() {
        const theme = themeOrder[currentThemeIndex];
        document.documentElement.setAttribute('data-theme', theme);
        localStorage.setItem('mmdp-theme', theme);
    }

    function cycleTheme() {
        currentThemeIndex = (currentThemeIndex + 1) % themeOrder.length;
        applyTheme();
        renderCurrent();
    }

    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', function() {
        if (themeOrder[currentThemeIndex] === 'system') {
            renderCurrent();
        }
    });

    applyTheme();

    // ─── Mermaid Init ─────────────────────────────────────────────
    function initMermaid() {
        const effective = getEffectiveTheme();
        mermaid.initialize({
            startOnLoad: false,
            theme: effective === 'dark' ? 'dark' : 'default',
            securityLevel: 'strict',
            logLevel: 'error',
            suppressErrorRendering: true
        });
    }

    // ─── Rendering ────────────────────────────────────────────────
    let renderCounter = 0;

    async function renderDiagram(source) {
        initMermaid();
        const id = 'mermaid-' + (renderCounter++);
        try {
            const { svg } = await mermaid.render(id, source);
            return { svg: svg, error: null };
        } catch (err) {
            const el = document.getElementById('d' + id);
            if (el) el.remove();
            return { svg: null, error: err.message || String(err) };
        }
    }

    async function renderContent(content) {
        currentContent = content;
        diagram.innerHTML = '';

        let sources;
        if (config.isMarkdown) {
            sources = extractMermaidBlocks(content);
        } else {
            sources = [content];
        }

        if (!sources || sources.length === 0) {
            showError('No mermaid diagram blocks found');
            return;
        }

        let hasError = false;
        let firstError = null;

        for (let i = 0; i < sources.length; i++) {
            const source = sources[i].trim();
            if (!source) continue;

            if (sources.length > 1) {
                if (i > 0) {
                    const divider = document.createElement('div');
                    divider.className = 'diagram-block-divider';
                    diagram.appendChild(divider);
                }
                const label = document.createElement('div');
                label.className = 'diagram-block-label';
                label.textContent = 'Block ' + (i + 1);
                diagram.appendChild(label);
            }

            const block = document.createElement('div');
            block.className = 'diagram-block';

            const result = await renderDiagram(source);
            if (result.svg) {
                block.innerHTML = result.svg;
                lastValidSVG = diagram.innerHTML;
            } else {
                hasError = true;
                if (!firstError) firstError = result.error;
            }

            diagram.appendChild(block);
        }

        if (hasError) {
            showError(firstError);
        } else {
            hideError();
        }
    }

    async function renderCurrent() {
        if (currentContent) {
            await renderContent(currentContent);
            if (isFitted) {
                requestAnimationFrame(fitToViewport);
            }
        }
    }

    function extractMermaidBlocks(content) {
        const re = /^```mermaid\s*\n([\s\S]*?)^```/gm;
        const blocks = [];
        let match;
        while ((match = re.exec(content)) !== null) {
            blocks.push(match[1]);
        }
        return blocks;
    }

    // ─── Error Display ────────────────────────────────────────────
    function showError(msg) {
        errorMessage.textContent = msg;
        errorOverlay.classList.remove('hidden');
        if (lastValidSVG) {
            diagramContainer.classList.add('diagram-dimmed');
        }
    }

    function hideError() {
        errorOverlay.classList.add('hidden');
        diagramContainer.classList.remove('diagram-dimmed');
    }

    // ─── Zoom/Pan ─────────────────────────────────────────────────
    function updateTransform() {
        diagramWrapper.style.transform =
            'translate(' + panX + 'px, ' + panY + 'px) scale(' + zoom + ')';
        if (isFitted) {
            zoomLevel.textContent = 'Fit';
        } else {
            zoomLevel.textContent = Math.round(zoom * 100) + '%';
        }
    }

    function fitToViewport() {
        var prevTransform = diagramWrapper.style.transform;
        diagramWrapper.style.transform = 'none';

        var containerRect = diagramContainer.getBoundingClientRect();
        var contentRect = diagram.getBoundingClientRect();
        if (contentRect.width <= 0 || contentRect.height <= 0) {
            diagramWrapper.style.transform = prevTransform;
            return;
        }

        var PADDING = 48;
        var scaleX = (containerRect.width - PADDING) / contentRect.width;
        var scaleY = (containerRect.height - PADDING) / contentRect.height;
        var fitZoom = Math.min(scaleX, scaleY);

        var naturalCX = (contentRect.left - containerRect.left) + contentRect.width / 2;
        var naturalCY = (contentRect.top - containerRect.top) + contentRect.height / 2;

        zoom = fitZoom;
        panX = containerRect.width / 2 - naturalCX * fitZoom;
        panY = containerRect.height / 2 - naturalCY * fitZoom;
        isFitted = true;
        updateTransform();
    }

    function zoomCentered(factor) {
        var rect = diagramContainer.getBoundingClientRect();
        var cx = rect.width / 2;
        var cy = rect.height / 2;
        var oldZoom = zoom;
        zoom = Math.min(Math.max(zoom * factor, 0.1), 50);
        var scale = zoom / oldZoom;
        panX = cx - scale * (cx - panX);
        panY = cy - scale * (cy - panY);
        isFitted = false;
        updateTransform();
    }

    // Mouse wheel zoom (centered on cursor)
    diagramContainer.addEventListener('wheel', function(e) {
        e.preventDefault();
        const rect = diagramContainer.getBoundingClientRect();
        const mouseX = e.clientX - rect.left;
        const mouseY = e.clientY - rect.top;
        const oldZoom = zoom;
        zoom = e.deltaY < 0 ? Math.min(zoom * 1.1, 50) : Math.max(zoom / 1.1, 0.1);
        const scale = zoom / oldZoom;
        panX = mouseX - scale * (mouseX - panX);
        panY = mouseY - scale * (mouseY - panY);
        isFitted = false;
        updateTransform();
    }, { passive: false });

    // ─── Diagram Pan & Window Drag ────────────────────────────────
    // Click on diagram SVG: pan the diagram
    // Click on empty background: move the window (native webview only)
    let winDragging = false;
    let winStartScreenX = 0;
    let winStartScreenY = 0;

    diagramContainer.addEventListener('mousedown', function(e) {
        if (e.target.closest('.floating-toolbar, .search-float, .tag-badge, button, .dropdown-menu')) return;

        // If clicking on the diagram content, pan
        if (e.target.closest('#diagram')) {
            isDragging = true;
            dragStartX = e.clientX;
            dragStartY = e.clientY;
            dragStartPanX = panX;
            dragStartPanY = panY;
            diagramContainer.classList.add('dragging');
            e.preventDefault();
            return;
        }

        // Otherwise, move the window (native webview only)
        if (typeof window.moveWindowBy === 'function') {
            winDragging = true;
            winStartScreenX = e.screenX;
            winStartScreenY = e.screenY;
            diagramContainer.classList.add('dragging');
            e.preventDefault();
        }
    });

    document.addEventListener('mousemove', function(e) {
        if (isDragging) {
            panX = dragStartPanX + (e.clientX - dragStartX);
            panY = dragStartPanY + (e.clientY - dragStartY);
            isFitted = false;
            updateTransform();
            return;
        }
        if (winDragging) {
            var dx = e.screenX - winStartScreenX;
            var dy = e.screenY - winStartScreenY;
            winStartScreenX = e.screenX;
            winStartScreenY = e.screenY;
            window.moveWindowBy(dx, dy);
        }
    });

    document.addEventListener('mouseup', function() {
        if (isDragging) {
            isDragging = false;
            diagramContainer.classList.remove('dragging');
        }
        if (winDragging) {
            winDragging = false;
            diagramContainer.classList.remove('dragging');
        }
    });

    // ─── Toolbar Drag (reposition within window) ──────────────────
    let tbDragging = false;
    let tbOffsetX = 0;
    let tbOffsetY = 0;

    toolbar.addEventListener('mousedown', function(e) {
        if (e.target.closest('button, .dropdown-menu, input, .zoom-label')) return;
        tbDragging = true;
        const rect = toolbar.getBoundingClientRect();
        tbOffsetX = e.clientX - rect.left;
        tbOffsetY = e.clientY - rect.top;
        e.preventDefault();
    });

    document.addEventListener('mousemove', function(e) {
        if (!tbDragging) return;
        // Switch to left-based positioning once dragging starts
        toolbar.style.right = 'auto';
        toolbar.style.left = (e.clientX - tbOffsetX) + 'px';
        toolbar.style.top = (e.clientY - tbOffsetY) + 'px';
    });

    document.addEventListener('mouseup', function() {
        tbDragging = false;
    });

    // ─── Search ───────────────────────────────────────────────────
    function openSearch() {
        searchBar.classList.remove('hidden');
        searchInput.focus();
        searchInput.select();
    }

    function closeSearch() {
        searchBar.classList.add('hidden');
        searchInput.value = '';
        clearHighlights();
        searchMatches = [];
        searchIndex = -1;
        searchCount.textContent = '';
    }

    // Wrap matching substrings in <mark> elements using safe DOM manipulation.
    function markTextNodes(el, query) {
        var walker = document.createTreeWalker(el, NodeFilter.SHOW_TEXT, null, false);
        var textNodes = [];
        var node;
        while ((node = walker.nextNode())) textNodes.push(node);
        var lowerQuery = query.toLowerCase();
        textNodes.forEach(function(tn) {
            var text = tn.nodeValue;
            var idx = text.toLowerCase().indexOf(lowerQuery);
            if (idx === -1) return;
            var before = text.substring(0, idx);
            var match = text.substring(idx, idx + query.length);
            var after = text.substring(idx + query.length);
            var parent = tn.parentNode;
            var mark = document.createElement('mark');
            mark.className = 'search-mark';
            mark.textContent = match;
            if (before) parent.insertBefore(document.createTextNode(before), tn);
            parent.insertBefore(mark, tn);
            if (after) parent.insertBefore(document.createTextNode(after), tn);
            parent.removeChild(tn);
        });
    }

    function performSearch(query) {
        clearHighlights();
        searchMatches = [];
        searchIndex = -1;

        if (!query) {
            searchCount.textContent = '';
            return;
        }

        var lowerQuery = query.toLowerCase();
        var svgs = diagram.querySelectorAll('svg');

        // Collect leaf text elements (skip parents whose child also matches)
        var matchedTextEls = [];
        svgs.forEach(function(svg) {
            var textEls = svg.querySelectorAll('span, tspan, text');
            textEls.forEach(function(textEl) {
                if (!textEl.textContent) return;
                if (!textEl.textContent.toLowerCase().includes(lowerQuery)) return;
                // Skip SVG <text> if a child <tspan> also matches (avoid double-counting)
                var childAlsoMatches = false;
                for (var i = 0; i < textEl.children.length; i++) {
                    var childTag = textEl.children[i].tagName;
                    if ((childTag === 'tspan' || childTag === 'SPAN' || childTag === 'text') &&
                        textEl.children[i].textContent &&
                        textEl.children[i].textContent.toLowerCase().includes(lowerQuery)) {
                        childAlsoMatches = true;
                        break;
                    }
                }
                if (childAlsoMatches) return;
                matchedTextEls.push(textEl);
            });
        });

        if (matchedTextEls.length === 0) {
            searchCount.textContent = '0 results';
            return;
        }

        matchedTextEls.forEach(function(textEl) {
            // Inline text highlight
            if (textEl.tagName === 'SPAN' || textEl.tagName === 'DIV') {
                markTextNodes(textEl, query);
            } else {
                // SVG text/tspan: insert a highlight rect behind the text
                try {
                    var bbox = textEl.getBBox();
                    var pad = 3;
                    var ns = 'http://www.w3.org/2000/svg';
                    var rect = document.createElementNS(ns, 'rect');
                    rect.setAttribute('x', bbox.x - pad);
                    rect.setAttribute('y', bbox.y - pad);
                    rect.setAttribute('width', bbox.width + pad * 2);
                    rect.setAttribute('height', bbox.height + pad * 2);
                    rect.setAttribute('rx', '3');
                    rect.setAttribute('class', 'search-bg-rect');
                    var textRoot = textEl.closest('text') || textEl;
                    // Copy the text's transform so the rect aligns
                    var tf = textRoot.getAttribute('transform');
                    if (tf) rect.setAttribute('transform', tf);
                    textRoot.parentNode.insertBefore(rect, textRoot);
                } catch (e) { /* getBBox may fail if hidden */ }
            }

            // Navigation group: outer node container, or text element as fallback
            var group = textEl.closest('.node')
                     || textEl.closest('.cluster')
                     || textEl.closest('.actor')
                     || textEl.closest('.label')
                     || textEl;
            if (searchMatches.indexOf(group) === -1) {
                searchMatches.push(group);
                group.classList.add('search-highlight');
            }
        });

        searchIndex = 0;
        highlightActive();
        updateSearchCount();
    }

    function highlightActive() {
        searchMatches.forEach(function(el) {
            el.classList.remove('search-active');
        });
        if (searchIndex >= 0 && searchIndex < searchMatches.length) {
            const active = searchMatches[searchIndex];
            active.classList.add('search-active');
            scrollToElement(active);
        }
    }

    function scrollToElement(el) {
        const rect = el.getBoundingClientRect();
        const containerRect = diagramContainer.getBoundingClientRect();
        const centerX = containerRect.left + containerRect.width / 2;
        const centerY = containerRect.top + containerRect.height / 2;
        const elCenterX = rect.left + rect.width / 2;
        const elCenterY = rect.top + rect.height / 2;
        panX += (centerX - elCenterX);
        panY += (centerY - elCenterY);
        updateTransform();
    }

    function nextMatch() {
        if (searchMatches.length === 0) return;
        searchIndex = (searchIndex + 1) % searchMatches.length;
        highlightActive();
        updateSearchCount();
    }

    function prevMatch() {
        if (searchMatches.length === 0) return;
        searchIndex = (searchIndex - 1 + searchMatches.length) % searchMatches.length;
        highlightActive();
        updateSearchCount();
    }

    function updateSearchCount() {
        if (searchMatches.length === 0) {
            searchCount.textContent = '0 results';
        } else {
            searchCount.textContent = (searchIndex + 1) + ' of ' + searchMatches.length;
        }
    }

    function clearHighlights() {
        document.querySelectorAll('.search-highlight, .search-active').forEach(function(el) {
            el.classList.remove('search-highlight', 'search-active');
        });
        // Remove injected SVG highlight rects
        document.querySelectorAll('.search-bg-rect').forEach(function(el) {
            el.remove();
        });
        // Unwrap <mark> elements back to plain text
        document.querySelectorAll('mark.search-mark').forEach(function(mark) {
            var parent = mark.parentNode;
            parent.replaceChild(document.createTextNode(mark.textContent), mark);
            parent.normalize();
        });
    }

    searchInput.addEventListener('input', function() {
        performSearch(searchInput.value);
    });

    searchInput.addEventListener('keydown', function(e) {
        if (e.key === 'Enter') {
            e.preventDefault();
            if (e.shiftKey) { prevMatch(); } else { nextMatch(); }
        }
    });

    document.getElementById('search-next').addEventListener('click', nextMatch);
    document.getElementById('search-prev').addEventListener('click', prevMatch);
    document.getElementById('search-close').addEventListener('click', closeSearch);

    // ─── Close button ─────────────────────────────────────────────
    document.getElementById('close-btn').addEventListener('click', function() {
        fetch('/api/shutdown', { method: 'POST' }).catch(function() {});
    });

    // ─── Auto-shape window (live reload) ────────────────────────────
    // Used on WebSocket updates — window is already visible, so just reshape.
    function autoShapeWindow() {
        // If the user has manually zoomed/panned, preserve their view.
        if (!isFitted) return;
        fitToViewport();
        if (typeof window.resizeWindow !== 'function') return;
        var dims = computeShapeDimensions();
        if (dims) {
            window.resizeWindow(dims.w, dims.h);
            requestAnimationFrame(fitToViewport);
        }
    }

    // ─── Toolbar Buttons ──────────────────────────────────────────
    document.getElementById('zoom-in').addEventListener('click', function() { zoomCentered(1.2); });
    document.getElementById('zoom-out').addEventListener('click', function() { zoomCentered(1/1.2); });
    document.getElementById('zoom-reset').addEventListener('click', fitToViewport);
    document.getElementById('theme-toggle').addEventListener('click', cycleTheme);
    document.getElementById('search-btn').addEventListener('click', openSearch);

    // ─── Export ───────────────────────────────────────────────────
    document.getElementById('export-btn').addEventListener('click', function(e) {
        e.stopPropagation();
        exportMenu.classList.toggle('hidden');
    });

    document.addEventListener('click', function(e) {
        if (!e.target.closest('.dropdown')) {
            exportMenu.classList.add('hidden');
        }
    });

    function baseName() {
        return (config.label || config.filename || 'diagram').replace(/\.[^.]+$/, '');
    }

    // Encode string as base64 (UTF-8 safe)
    function toBase64(str) {
        return btoa(unescape(encodeURIComponent(str)));
    }

    // Encode ArrayBuffer as base64
    function arrayBufferToBase64(buffer) {
        var bytes = new Uint8Array(buffer);
        var binary = '';
        for (var i = 0; i < bytes.length; i++) {
            binary += String.fromCharCode(bytes[i]);
        }
        return btoa(binary);
    }

    document.getElementById('export-svg').addEventListener('click', function() {
        var svg = diagram.querySelector('svg');
        if (!svg) return;
        exportMenu.classList.add('hidden');
        var s = new XMLSerializer().serializeToString(svg);

        if (typeof window.saveFileDialog === 'function') {
            window.saveFileDialog(baseName() + '.svg', toBase64(s), 'svg');
        } else {
            // Fallback for browser/E2E context
            var blob = new Blob([s], { type: 'image/svg+xml' });
            var a = document.createElement('a');
            a.href = URL.createObjectURL(blob);
            a.download = baseName() + '.svg';
            a.click();
            URL.revokeObjectURL(a.href);
        }
    });

    document.getElementById('export-png').addEventListener('click', function() {
        var svg = diagram.querySelector('svg');
        if (!svg) return;
        exportMenu.classList.add('hidden');
        var s = new XMLSerializer().serializeToString(svg);
        var url = URL.createObjectURL(new Blob([s], { type: 'image/svg+xml;charset=utf-8' }));
        var img = new Image();
        img.onload = function() {
            var sc = window.devicePixelRatio || 1;
            var c = document.createElement('canvas');
            c.width = img.width * sc;
            c.height = img.height * sc;
            var ctx = c.getContext('2d');
            ctx.scale(sc, sc);
            ctx.drawImage(img, 0, 0);
            c.toBlob(function(blob) {
                if (!blob) return;
                var reader = new FileReader();
                reader.onload = function() {
                    if (typeof window.saveFileDialog === 'function') {
                        window.saveFileDialog(baseName() + '.png', arrayBufferToBase64(reader.result), 'png');
                    } else {
                        var a = document.createElement('a');
                        a.href = URL.createObjectURL(blob);
                        a.download = baseName() + '.png';
                        a.click();
                        URL.revokeObjectURL(a.href);
                    }
                };
                reader.readAsArrayBuffer(blob);
            });
            URL.revokeObjectURL(url);
        };
        img.src = url;
    });

    // ─── Keyboard Shortcuts ───────────────────────────────────────
    document.addEventListener('keydown', function(e) {
        if (e.target === searchInput && e.key !== 'Escape') return;

        if ((e.metaKey || e.ctrlKey) && e.key === 'f') {
            e.preventDefault();
            openSearch();
            return;
        }

        if (e.key === 'Escape') {
            e.preventDefault();
            if (!searchBar.classList.contains('hidden')) {
                closeSearch();
            } else {
                fetch('/api/shutdown', { method: 'POST' }).catch(function() {});
            }
            return;
        }

        // Spacebar: lightbox-style dismiss (not when focused on text input)
        if (e.key === ' ' && !e.target.matches('input, textarea, [contenteditable]')) {
            e.preventDefault();
            fetch('/api/shutdown', { method: 'POST' }).catch(function() {});
            return;
        }

        if (e.key === 't' || e.key === 'T') { cycleTheme(); return; }
        if (e.key === '+' || e.key === '=') { zoomCentered(1.2); return; }
        if (e.key === '-') { zoomCentered(1/1.2); return; }
        if (e.key === '0') { fitToViewport(); return; }
    });

    // ─── WebSocket ────────────────────────────────────────────────
    function connectWS() {
        const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
        ws = new WebSocket(proto + '//' + location.host + '/ws');

        ws.onopen = function() {
            disconnectedBanner.classList.add('hidden');
            reconnectDelay = 500;
        };

        ws.onmessage = function(event) {
            renderContent(event.data).then(function() {
                requestAnimationFrame(autoShapeWindow);
            });
        };

        ws.onclose = function() {
            disconnectedBanner.classList.remove('hidden');
            ws = null;
            setTimeout(function() {
                reconnectDelay = Math.min(reconnectDelay * 2, MAX_RECONNECT_DELAY);
                connectWS();
            }, reconnectDelay);
        };

        ws.onerror = function() {
            if (ws) ws.close();
        };
    }

    // ─── Init ─────────────────────────────────────────────────────
    // Compute the window shape dimensions without applying them.
    function computeShapeDimensions() {
        var svgs = diagram.querySelectorAll('svg');
        if (!svgs.length) return null;

        var contentW = 0, contentH = 0;
        svgs.forEach(function(svg) {
            var rect = svg.getBoundingClientRect();
            contentW = Math.max(contentW, rect.width);
            contentH += rect.height;
        });
        var blocks = diagram.querySelectorAll('.diagram-block');
        if (blocks.length > 1) contentH += (blocks.length - 1) * 60;
        if (contentW <= 0 || contentH <= 0) return null;

        var aspect = contentW / contentH;
        var MAX_ASPECT = 2.5;
        aspect = Math.min(Math.max(aspect, 1 / MAX_ASPECT), MAX_ASPECT);

        var AREA = 1200000;
        var w = Math.sqrt(AREA * aspect);
        var h = AREA / w;

        var MIN_W = 700, MIN_H = 500;
        var maxW = Math.min(1500, screen.availWidth * 0.75);
        var maxH = Math.min(1100, screen.availHeight * 0.75);
        w = Math.min(Math.max(w, MIN_W), maxW);
        h = Math.min(Math.max(h, MIN_H), maxH);

        return { w: Math.round(w), h: Math.round(h) };
    }

    async function init() {
        try {
            const resp = await fetch('/api/diagram');
            const content = await resp.text();
            await renderContent(content);
            requestAnimationFrame(function() {
                fitToViewport();
                // Compute shape and pass to showWindow so resize + reveal
                // happen atomically — no flash.
                if (typeof window.showWindow === 'function') {
                    var dims = computeShapeDimensions();
                    window.showWindow(dims ? dims.w : 0, dims ? dims.h : 0);
                    // Re-fit after window reshape — the viewport changed.
                    requestAnimationFrame(fitToViewport);
                }
            });
        } catch (err) {
            showError('Failed to load diagram: ' + err.message);
            if (typeof window.showWindow === 'function') {
                window.showWindow(0, 0);
            }
        }

        if (!config.noWatch) {
            connectWS();
        }
    }

    init();
})();
