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

// 3. Mermaid Modal Functions
window.openMermaidModal = (index: number) => {
    const container = document.getElementById('mermaid-' + index);
    if (!container) return;
    const originalSvg = container.querySelector('.mermaid svg');
    if (!originalSvg) return;

    const modal = document.getElementById('mermaid-modal');
    const modalContent = document.getElementById('mermaid-modal-content');

    if (!modal || !modalContent) return;

    // Clone the SVG for the modal
    const clonedSvg = originalSvg.cloneNode(true) as SVGElement;

    // Adjust styles for fullscreen viewing
    // Reset dimensions to allow natural scaling, but limit to viewport if smaller
    clonedSvg.removeAttribute('width');
    clonedSvg.removeAttribute('height');
    clonedSvg.style.width = 'auto';
    clonedSvg.style.height = 'auto';
    clonedSvg.style.maxWidth = 'none'; // Allow it to overflow
    clonedSvg.style.minWidth = '100%'; // Ensure it fills at least width if smaller
    
    // Check if SVG is taller than viewport
    // We need to use type assertion carefully as Element doesn't have getBBox
    const graphicsElement = originalSvg as unknown as SVGGraphicsElement;
    
    // Default to centering
    modalContent.style.alignItems = 'center';
    modalContent.style.justifyContent = 'center';
    modalContent.style.overflow = 'auto'; // Always allow scroll

    try {
        // Check if getBBox exists and is a function before calling
        if (typeof graphicsElement.getBBox === 'function') {
            const bbox = graphicsElement.getBBox();
            
            // Heuristic: If SVG is very tall, align top to allow scrolling
            // If we center a very tall SVG, the top part gets cut off and is unreachable
            if (bbox.height > window.innerHeight - 80) {
                 modalContent.style.alignItems = 'flex-start';
                 // Remove height constraint to allow full expansion
                 clonedSvg.style.height = 'auto'; 
            } else {
                 // If it fits, we can let it be max 100% height to fit screen
                 clonedSvg.style.maxHeight = '95vh';
            }

            // If SVG is very wide, align left
            if (bbox.width > window.innerWidth - 80) {
                 modalContent.style.justifyContent = 'flex-start';
            }
        }
    } catch (e) {
        console.warn('Could not get BBox', e);
    }

    modalContent.innerHTML = '';
    modalContent.appendChild(clonedSvg);
    modal.classList.add('active');
    document.body.style.overflow = 'hidden'; // Prevent background scrolling
};

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

// 4. SSE for Auto Reload
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
    generateTOC();
    window.addEventListener('scroll', highlightTOC);
    
    // Start async init
    await init();
});
