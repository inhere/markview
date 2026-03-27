// Mark as ES module so `declare global` works correctly
export {};

// Initialize Mermaid and Highlight.js dynamically
const init = async () => {
    // Dynamic imports for code splitting
    const { default: mermaid } = await import('mermaid');
    const { default: hljs } = await import('highlight.js/lib/core');

    // Import languages dynamically
    const languages = [
        import('highlight.js/lib/languages/javascript'),
        import('highlight.js/lib/languages/typescript'),
        import('highlight.js/lib/languages/xml'),
        import('highlight.js/lib/languages/css'),
        import('highlight.js/lib/languages/json'),
        import('highlight.js/lib/languages/bash'),
        import('highlight.js/lib/languages/go'),
        import('highlight.js/lib/languages/markdown'),
        import('highlight.js/lib/languages/yaml'),
        import('highlight.js/lib/languages/sql'),
        import('highlight.js/lib/languages/python'),
        import('highlight.js/lib/languages/rust')
    ];

    const loadedLanguages = await Promise.all(languages);

    // Register languages
    const langNames = ['javascript', 'typescript', 'xml', 'css', 'json', 'bash', 'go', 'markdown', 'yaml', 'sql', 'python', 'rust'];
    loadedLanguages.forEach((module, i) => {
        hljs.registerLanguage(langNames[i], module.default);
    });

    // Initialize Mermaid
    mermaid.initialize({ startOnLoad: false });

    // Initialize Highlight.js
    hljs.highlightAll();

    // Find mermaid code blocks and transform them
    const mermaidBlocks = document.querySelectorAll('pre code.language-mermaid');

    for (let i = 0; i < mermaidBlocks.length; i++) {
        const block = mermaidBlocks[i];
        const pre = block.parentElement;
        const content = block.textContent;

        if (!pre) continue;

        const container = document.createElement('div');
        container.className = 'mermaid-container';
        container.id = 'mermaid-' + i;

        // Create fullscreen button
        const btn = document.createElement('button');
        btn.className = 'mermaid-fullscreen-btn';
        btn.innerHTML = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M15 3h6v6M9 21H3v-6M21 3l-7 7M3 21l7-7"/></svg>';
        btn.title = "View Fullscreen";
        btn.onclick = () => window.openMermaidModal(i);

        // Create mermaid div
        const mermaidDiv = document.createElement('div');
        mermaidDiv.className = 'mermaid';
        mermaidDiv.textContent = content;

        container.appendChild(btn);
        container.appendChild(mermaidDiv);

        // Replace the <pre> with the new container
        pre.replaceWith(container);
    }

    // Run mermaid
    await mermaid.run();
};

// Interface for SVG elements that support getBBox
interface SVGGraphicsElement extends SVGElement {
  getBBox(): DOMRect;
}

// Global Window Interface
declare global {
    interface Window {
        openMermaidModal: (index: number) => void;
        closeMermaidModal: () => void;
    }
}

// 1. Generate TOC dynamically
function generateTOC() {
    const tocList = document.getElementById('toc-list');
    const headers = document.querySelectorAll('#content h1, #content h2, #content h3');

    if (!tocList) return;

    headers.forEach((header, index) => {
        if (!header.id) {
            const friendlyId = (header as HTMLElement).innerText.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/(^-|-$)/g, '');
            header.id = friendlyId || 'section-' + index;
        }

        let level = 'h1';
        if (header.tagName === 'H2') level = 'h2';
        if (header.tagName === 'H3') level = 'h3';

        const li = document.createElement('li');
        li.className = 'toc-item';

        const a = document.createElement('a');
        a.href = '#' + header.id;
        a.innerText = (header as HTMLElement).innerText;
        a.className = 'toc-link ' + 'toc-' + level;

        li.appendChild(a);
        tocList.appendChild(li);
    });
}

// 2. Active TOC Highlighting (Scroll Spy)
const highlightTOC = () => {
    const scrollPos = window.scrollY + 100;
    const headers = document.querySelectorAll('h1, h2, h3');
    let currentId = '';

    headers.forEach(header => {
        if ((header as HTMLElement).offsetTop <= scrollPos) {
            currentId = header.id;
        }
    });

    if (currentId) {
        document.querySelectorAll('.toc-link').forEach(l => {
            l.classList.remove('active');
            if (l.getAttribute('href') === '#' + currentId) {
                l.classList.add('active');
                const sidebar = document.querySelector('.toc-container');
                if (sidebar) {
                    const linkRect = l.getBoundingClientRect();
                    const sidebarRect = sidebar.getBoundingClientRect();
                    if (linkRect.top < sidebarRect.top || linkRect.bottom > sidebarRect.bottom) {
                        l.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
                    }
                }
            }
        });
    }
};

// 3. Toolbar and Styling
function setupToolbar() {
    const toolbar = document.getElementById('toolbar');
    if (!toolbar) return;

    // Width Controls
    const widthBtns = toolbar.querySelectorAll('[data-width]');
    widthBtns.forEach(btn => {
        btn.addEventListener('click', () => {
            const width = (btn as HTMLElement).dataset.width;
            if (width) {
                document.documentElement.style.setProperty('--layout-max-width', width);
                // Update active state
                widthBtns.forEach(b => b.classList.remove('active'));
                btn.classList.add('active');
            }
        });
    });

    // Font Controls
    const fontInc = document.getElementById('font-inc');
    const fontDec = document.getElementById('font-dec');
    let currentFontSize = 16;

    const updateFont = () => {
        document.documentElement.style.fontSize = `${currentFontSize}px`;
    };

    fontInc?.addEventListener('click', () => {
        if (currentFontSize < 24) {
            currentFontSize++;
            updateFont();
        }
    });

    fontDec?.addEventListener('click', () => {
        if (currentFontSize > 12) {
            currentFontSize--;
            updateFont();
        }
    });
}

// 4. Mermaid Modal Functions
let currentZoom = 1.0;

window.openMermaidModal = (index: number) => {
    const container = document.getElementById('mermaid-' + index);
    if (!container) return;
    const originalSvg = container.querySelector('.mermaid svg');
    if (!originalSvg) return;

    const modal = document.getElementById('mermaid-modal');
    const modalContent = document.getElementById('mermaid-modal-content');

    if (!modal || !modalContent) return;

    // Reset Zoom
    currentZoom = 1.0;

    // Clone the SVG for the modal
    const clonedSvg = originalSvg.cloneNode(true) as SVGElement;

    // Adjust styles for fullscreen viewing
    clonedSvg.removeAttribute('width');
    clonedSvg.removeAttribute('height');
    // Start with auto/natural size
    clonedSvg.style.width = 'auto';
    clonedSvg.style.height = 'auto';
    clonedSvg.style.maxWidth = 'none';
    clonedSvg.style.minWidth = '0'; // Allow shrinking

    // Check dimensions
    const graphicsElement = originalSvg as unknown as SVGGraphicsElement;

    // Default alignment
    modalContent.style.alignItems = 'center';
    modalContent.style.justifyContent = 'center';

    try {
        if (typeof graphicsElement.getBBox === 'function') {
            const bbox = graphicsElement.getBBox();

            // If tall, align top
            if (bbox.height > window.innerHeight - 80) {
                 modalContent.style.alignItems = 'flex-start';
                 clonedSvg.style.height = 'auto';
            } else {
                 // Fit screen height initially if it's not too tall
                 clonedSvg.style.maxHeight = '90vh';
            }

            // If wide, align left
            if (bbox.width > window.innerWidth - 80) {
                 modalContent.style.justifyContent = 'flex-start';
            }
        }
    } catch (e) {
        console.warn('Could not get BBox', e);
    }

    modalContent.innerHTML = '';
    modalContent.appendChild(clonedSvg);

    // Create/Update Controls
    let controls = document.getElementById('mermaid-modal-controls');

    // Always remove and recreate to bind new SVG listeners
    if (controls) controls.remove();

    controls = document.createElement('div');
    controls.id = 'mermaid-modal-controls';
    controls.className = 'mermaid-modal-controls';
    controls.innerHTML = `
        <button class="mermaid-control-btn mermaid-control-step" id="zoom-out" title="Zoom Out">−</button>
        <span class="mermaid-zoom-level" id="mermaid-zoom-level">100%</span>
        <button class="mermaid-control-btn mermaid-control-step" id="zoom-in" title="Zoom In">+</button>
        <span class="mermaid-ctrl-divider"></span>
        <button class="mermaid-control-btn" data-zoom="0.5" title="50%">50%</button>
        <button class="mermaid-control-btn" data-zoom="0.75" title="75%">75%</button>
        <button class="mermaid-control-btn active" data-zoom="1" title="100%">100%</button>
    `;
    modal.appendChild(controls);

    // Preset zoom buttons
    controls.querySelectorAll('[data-zoom]').forEach(btn => {
        btn.addEventListener('click', (e) => {
            e.stopPropagation();
            const z = parseFloat((btn as HTMLElement).dataset.zoom || '1');
            currentZoom = z;
            updateZoomLevel(clonedSvg, currentZoom);
            // Update active state
            controls!.querySelectorAll('[data-zoom]').forEach(b => b.classList.remove('active'));
            btn.classList.add('active');
        });
    });

    // Add listeners
    const zoomInBtn = document.getElementById('zoom-in');
    const zoomOutBtn = document.getElementById('zoom-out');

    if (zoomInBtn) {
        zoomInBtn.addEventListener('click', (e) => {
            e.stopPropagation();
            if (currentZoom < 2.0) {
                currentZoom = Math.round((currentZoom + 0.05) * 100) / 100;
                updateZoomLevel(clonedSvg, currentZoom);
                syncPresetBtns(controls!);
            }
        });
    }

    if (zoomOutBtn) {
        zoomOutBtn.addEventListener('click', (e) => {
            e.stopPropagation();
            if (currentZoom > 0.5) {
                currentZoom = Math.round((currentZoom - 0.05) * 100) / 100;
                updateZoomLevel(clonedSvg, currentZoom);
                syncPresetBtns(controls!);
            }
        });
    }

    modal.classList.add('active');
    document.body.style.overflow = 'hidden';
};

// Helper: sync preset btn active state after step zoom
function syncPresetBtns(controls: HTMLElement) {
    controls.querySelectorAll('[data-zoom]').forEach(btn => {
        const z = parseFloat((btn as HTMLElement).dataset.zoom || '1');
        btn.classList.toggle('active', Math.abs(z - currentZoom) < 0.001);
    });
}


window.closeMermaidModal = () => {
    const modal = document.getElementById('mermaid-modal');
    if (modal) {
        modal.classList.remove('active');
        document.body.style.overflow = '';
    }
};

// Close on Escape key
document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') window.closeMermaidModal();
});

// 5. SSE for Auto Reload
const evtSource = new EventSource("/sse");
const liveDot = document.getElementById('live-dot');
const statusText = document.getElementById('status-text');

evtSource.onmessage = (event) => {
    if (event.data === 'reload') {
        if (liveDot) liveDot.classList.add('reloading');
        if (statusText) statusText.innerText = 'Syncing...';
        setTimeout(() => window.location.reload(), 500);
    }
};

evtSource.onerror = () => {
    if (liveDot) liveDot.style.backgroundColor = 'var(--status-warn)';
    if (statusText) statusText.innerText = 'Disconnected';
};

// Initialization
document.addEventListener('DOMContentLoaded', async () => {
    setupToolbar();
    generateTOC();
    window.addEventListener('scroll', highlightTOC);

    // Start async init
    await init();
});

// Helper for zoom updates
function updateZoomLevel(svg: SVGElement, zoom: number) {
    const label = document.getElementById('mermaid-zoom-level');
    if (label) label.textContent = Math.round(zoom * 100) + '%';

    if (zoom === 1.0) {
        // Reset to initial state logic
        svg.style.width = 'auto';
        svg.style.height = 'auto';
        svg.style.minWidth = '0';
        svg.style.maxWidth = 'none';

        // Re-apply height constraint if it fits screen
        const graphicsElement = svg as unknown as SVGGraphicsElement;
         try {
            if (typeof graphicsElement.getBBox === 'function') {
                const bbox = graphicsElement.getBBox();
                if (bbox.height < window.innerHeight - 80) {
                     svg.style.maxHeight = '90vh';
                } else {
                     svg.style.maxHeight = 'none';
                }
            }
        } catch (e) {}
    } else {
        // Apply percentage width relative to viewport
        svg.style.width = Math.round(zoom * 100) + '%';
        svg.style.maxHeight = 'none';
        svg.style.height = 'auto';
    }
}
