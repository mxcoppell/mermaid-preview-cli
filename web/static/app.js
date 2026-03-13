// mermaid-preview frontend
(function() {
    'use strict';

    const config = window.__CONFIG__;
    let currentContent = null;
    let lastValidSVG = null;
    let zoom = 1;
    let panX = 0;
    let panY = 0;
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
    const errorOverlay = document.getElementById('error-overlay');
    const errorMessage = document.getElementById('error-message');
    const disconnectedBanner = document.getElementById('disconnected-banner');
    const searchBar = document.getElementById('search-bar');
    const searchInput = document.getElementById('search-input');
    const searchCount = document.getElementById('search-count');
    const zoomLevel = document.getElementById('zoom-level');
    const statusType = document.getElementById('status-type');
    const statusNodes = document.getElementById('status-nodes');
    const statusUpdated = document.getElementById('status-updated');
    const exportMenu = document.getElementById('export-menu');

    // ─── Theme ────────────────────────────────────────────────────
    const themeOrder = ['system', 'light', 'dark'];
    let currentThemeIndex = themeOrder.indexOf(
        localStorage.getItem('mermaid-preview-theme') || config.theme
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
        localStorage.setItem('mermaid-preview-theme', theme);
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
            // Clean up failed render element
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
        let nodeCount = 0;
        let diagramType = '';

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

                const svgEl = block.querySelector('svg');
                if (svgEl) {
                    nodeCount += svgEl.querySelectorAll('.node, .actor, .cluster').length;
                }

                if (i === 0) {
                    diagramType = detectDiagramType(source);
                }
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

        updateStatus(diagramType, nodeCount);
    }

    function renderCurrent() {
        if (currentContent) {
            renderContent(currentContent);
        }
    }

    function detectDiagramType(source) {
        const first = source.trim().split('\n')[0].toLowerCase();
        if (first.startsWith('graph') || first.startsWith('flowchart')) return 'Flowchart';
        if (first.startsWith('sequencediagram') || first.startsWith('sequence')) return 'Sequence';
        if (first.startsWith('classdiagram') || first.startsWith('class')) return 'Class';
        if (first.startsWith('statediagram') || first.startsWith('state')) return 'State';
        if (first.startsWith('erdiagram') || first.startsWith('er')) return 'ER';
        if (first.startsWith('gantt')) return 'Gantt';
        if (first.startsWith('pie')) return 'Pie';
        if (first.startsWith('gitgraph') || first.startsWith('git')) return 'Git Graph';
        if (first.startsWith('mindmap')) return 'Mindmap';
        if (first.startsWith('timeline')) return 'Timeline';
        return 'Diagram';
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

    // ─── Status Bar ───────────────────────────────────────────────
    function updateStatus(type, nodeCount) {
        statusType.textContent = type || '';
        statusNodes.textContent = nodeCount > 0 ? nodeCount + ' nodes' : '';
        statusUpdated.textContent = 'Updated ' + new Date().toLocaleTimeString();
    }

    // ─── Zoom/Pan ─────────────────────────────────────────────────
    function updateTransform() {
        diagramWrapper.style.transform =
            'translate(' + panX + 'px, ' + panY + 'px) scale(' + zoom + ')';
        if (zoom === 1 && panX === 0 && panY === 0) {
            zoomLevel.textContent = 'Fit';
        } else {
            zoomLevel.textContent = Math.round(zoom * 100) + '%';
        }
    }

    function zoomCentered(factor) {
        var rect = diagramContainer.getBoundingClientRect();
        var cx = rect.width / 2;
        var cy = rect.height / 2;

        var oldZoom = zoom;
        zoom = Math.min(Math.max(zoom * factor, 0.1), 10);
        var scale = zoom / oldZoom;

        panX = cx - scale * (cx - panX);
        panY = cy - scale * (cy - panY);
        updateTransform();
    }

    function zoomIn() {
        zoomCentered(1.2);
    }

    function zoomOut() {
        zoomCentered(1 / 1.2);
    }

    function resetZoom() {
        zoom = 1;
        panX = 0;
        panY = 0;
        updateTransform();
    }

    // Mouse wheel zoom (centered on cursor)
    diagramContainer.addEventListener('wheel', function(e) {
        e.preventDefault();
        const rect = diagramContainer.getBoundingClientRect();
        const mouseX = e.clientX - rect.left;
        const mouseY = e.clientY - rect.top;

        const oldZoom = zoom;
        if (e.deltaY < 0) {
            zoom = Math.min(zoom * 1.1, 10);
        } else {
            zoom = Math.max(zoom / 1.1, 0.1);
        }

        const scale = zoom / oldZoom;
        panX = mouseX - scale * (mouseX - panX);
        panY = mouseY - scale * (mouseY - panY);

        updateTransform();
    }, { passive: false });

    // Drag to pan
    diagramContainer.addEventListener('mousedown', function(e) {
        if (e.target.closest('#search-bar') || e.target.closest('button')) return;
        isDragging = true;
        dragStartX = e.clientX;
        dragStartY = e.clientY;
        dragStartPanX = panX;
        dragStartPanY = panY;
        diagramContainer.classList.add('dragging');
        e.preventDefault();
    });

    document.addEventListener('mousemove', function(e) {
        if (!isDragging) return;
        panX = dragStartPanX + (e.clientX - dragStartX);
        panY = dragStartPanY + (e.clientY - dragStartY);
        updateTransform();
    });

    document.addEventListener('mouseup', function() {
        if (!isDragging) return;
        isDragging = false;
        diagramContainer.classList.remove('dragging');
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

    function performSearch(query) {
        clearHighlights();
        searchMatches = [];
        searchIndex = -1;

        if (!query) {
            searchCount.textContent = '';
            return;
        }

        const lowerQuery = query.toLowerCase();
        const svgs = diagram.querySelectorAll('svg');

        svgs.forEach(function(svg) {
            const textEls = svg.querySelectorAll('text, tspan');
            textEls.forEach(function(textEl) {
                if (textEl.textContent.toLowerCase().includes(lowerQuery)) {
                    let group = textEl.closest('.node, .cluster, .actor, .label');
                    if (group && searchMatches.indexOf(group) === -1) {
                        searchMatches.push(group);
                    }
                }
            });
        });

        if (searchMatches.length === 0) {
            searchCount.textContent = '0 results';
            return;
        }

        searchMatches.forEach(function(el) {
            el.classList.add('search-highlight');
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
    }

    searchInput.addEventListener('input', function() {
        performSearch(searchInput.value);
    });

    searchInput.addEventListener('keydown', function(e) {
        if (e.key === 'Enter') {
            e.preventDefault();
            if (e.shiftKey) {
                prevMatch();
            } else {
                nextMatch();
            }
        }
    });

    document.getElementById('search-next').addEventListener('click', nextMatch);
    document.getElementById('search-prev').addEventListener('click', prevMatch);
    document.getElementById('search-close').addEventListener('click', closeSearch);

    // ─── Export ────────────────────────────────────────────────────
    document.getElementById('export-btn').addEventListener('click', function(e) {
        e.stopPropagation();
        exportMenu.classList.toggle('hidden');
    });

    document.addEventListener('click', function(e) {
        if (!e.target.closest('.dropdown')) {
            exportMenu.classList.add('hidden');
        }
    });

    document.getElementById('export-svg').addEventListener('click', function() {
        const svg = diagram.querySelector('svg');
        if (!svg) return;
        const serializer = new XMLSerializer();
        const svgString = serializer.serializeToString(svg);
        downloadFile(svgString, config.filename.replace(/\.[^.]+$/, '') + '.svg', 'image/svg+xml');
        exportMenu.classList.add('hidden');
    });

    document.getElementById('export-png').addEventListener('click', function() {
        const svg = diagram.querySelector('svg');
        if (!svg) return;

        const serializer = new XMLSerializer();
        const svgString = serializer.serializeToString(svg);
        const svgBlob = new Blob([svgString], { type: 'image/svg+xml;charset=utf-8' });
        const url = URL.createObjectURL(svgBlob);

        const img = new Image();
        img.onload = function() {
            const scale = window.devicePixelRatio || 1;
            const canvas = document.createElement('canvas');
            canvas.width = img.width * scale;
            canvas.height = img.height * scale;
            const ctx = canvas.getContext('2d');
            ctx.scale(scale, scale);
            ctx.drawImage(img, 0, 0);

            canvas.toBlob(function(blob) {
                const a = document.createElement('a');
                a.href = URL.createObjectURL(blob);
                a.download = config.filename.replace(/\.[^.]+$/, '') + '.png';
                a.click();
                URL.revokeObjectURL(a.href);
            });
            URL.revokeObjectURL(url);
        };
        img.src = url;
        exportMenu.classList.add('hidden');
    });

    document.getElementById('export-print').addEventListener('click', function() {
        exportMenu.classList.add('hidden');
        window.print();
    });

    function downloadFile(content, filename, mimeType) {
        const blob = new Blob([content], { type: mimeType });
        const a = document.createElement('a');
        a.href = URL.createObjectURL(blob);
        a.download = filename;
        a.click();
        URL.revokeObjectURL(a.href);
    }

    // ─── Zoom buttons ─────────────────────────────────────────────
    document.getElementById('zoom-in').addEventListener('click', zoomIn);
    document.getElementById('zoom-out').addEventListener('click', zoomOut);
    document.getElementById('zoom-reset').addEventListener('click', resetZoom);
    document.getElementById('theme-toggle').addEventListener('click', cycleTheme);

    // ─── Keyboard Shortcuts ───────────────────────────────────────
    document.addEventListener('keydown', function(e) {
        if (e.target === searchInput && e.key !== 'Escape') return;

        if ((e.metaKey || e.ctrlKey) && e.key === 'f') {
            e.preventDefault();
            openSearch();
            return;
        }

        if (e.key === 'Escape') {
            if (!searchBar.classList.contains('hidden')) {
                closeSearch();
            } else {
                fetch('/api/shutdown', { method: 'POST' }).catch(function() {});
            }
            return;
        }

        if (e.key === 't' || e.key === 'T') {
            cycleTheme();
            return;
        }

        if (e.key === '+' || e.key === '=') {
            zoomIn();
            return;
        }

        if (e.key === '-') {
            zoomOut();
            return;
        }

        if (e.key === '0') {
            resetZoom();
            return;
        }
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
            renderContent(event.data);
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
    async function init() {
        try {
            const resp = await fetch('/api/diagram');
            const content = await resp.text();
            await renderContent(content);
        } catch (err) {
            showError('Failed to load diagram: ' + err.message);
        }

        if (!config.noWatch) {
            connectWS();
        }
    }

    init();
})();
