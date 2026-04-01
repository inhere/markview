import { escapeHtml } from './util';

interface SVGGraphicsElement extends SVGElement {
    getBBox(): DOMRect;
}

declare global {
    interface Window {
        openMermaidModal: (index: number) => void;
        closeMermaidModal: () => void;
    }
}

let mermaidInitialized = false;
let mermaidModulePromise: Promise<typeof import('mermaid').default> | null = null;
let mermaidCounter = 0;
let modalSetupCompleted = false;
let currentZoom = 1.0;

const minZoom = 0.3;
const maxZoom = 2.0;

export function buildMermaidContainerId(index: number) {
    return `mermaid-${index}`;
}

export function parseMermaidContainerIndex(containerId: string) {
    const numericIndex = Number(containerId.replace('mermaid-', ''));
    return Number.isFinite(numericIndex) ? numericIndex : null;
}

async function getMermaidModule() {
    if (!mermaidModulePromise) {
        mermaidModulePromise = import('mermaid').then(module => module.default);
    }

    const mermaid = await mermaidModulePromise;
    if (!mermaidInitialized) {
        mermaid.initialize({ startOnLoad: false });
        mermaidInitialized = true;
    }

    return mermaid;
}

export async function enhanceMermaidContent(contentRoot: HTMLElement) {
    const mermaidBlocks = contentRoot.querySelectorAll('pre code.language-mermaid');
    if (!mermaidBlocks.length) {
        return;
    }

    const mermaid = await getMermaidModule();

    for (const block of mermaidBlocks) {
        const pre = block.parentElement;
        const content = block.textContent;

        if (!(pre instanceof HTMLElement)) {
            continue;
        }

        const containerId = buildMermaidContainerId(mermaidCounter++);

        const container = document.createElement('div');
        container.className = 'mermaid-container';
        container.id = containerId;
        container.dataset.source = content || '';

        const actions = document.createElement('div');
        actions.className = 'mermaid-actions';

        const fullscreenButton = document.createElement('button');
        fullscreenButton.className = 'mermaid-fullscreen-btn';
        fullscreenButton.innerHTML = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M15 3h6v6M9 21H3v-6M21 3l-7 7M3 21l7-7"/></svg>';
        fullscreenButton.title = 'View Fullscreen';
        fullscreenButton.onclick = () => {
            const numericIndex = parseMermaidContainerIndex(containerId);
            if (numericIndex !== null) {
                openMermaidModal(numericIndex);
            }
        };

        const sourceButton = document.createElement('button');
        sourceButton.className = 'mermaid-source-btn';
        sourceButton.textContent = '源码';
        sourceButton.title = 'View Mermaid Source';

        const sourcePanel = document.createElement('div');
        sourcePanel.className = 'mermaid-source-inline';
        sourcePanel.innerHTML = `
            <div class="mermaid-source-inline-header">Mermaid Source</div>
            <pre><code>${escapeHtml(content || '')}</code></pre>
        `;

        sourceButton.onclick = () => {
            sourcePanel.classList.toggle('active');
        };

        const mermaidDiv = document.createElement('div');
        mermaidDiv.className = 'mermaid';
        mermaidDiv.textContent = content;

        actions.appendChild(sourceButton);
        actions.appendChild(fullscreenButton);
        container.appendChild(actions);
        container.appendChild(sourcePanel);
        container.appendChild(mermaidDiv);

        pre.replaceWith(container);
    }

    await mermaid.run();
}

export function setupMermaidModal() {
    if (modalSetupCompleted) {
        return;
    }

    window.openMermaidModal = openMermaidModal;
    window.closeMermaidModal = closeMermaidModal;

    document.addEventListener('keydown', event => {
        if (event.key === 'Escape') {
            closeMermaidModal();
        }
    });

    modalSetupCompleted = true;
}

function openMermaidModal(index: number) {
    const container = document.getElementById(buildMermaidContainerId(index));
    if (!container) return;
    const originalSvg = container.querySelector('.mermaid svg');
    if (!originalSvg) return;

    const modal = document.getElementById('mermaid-modal');
    const modalContent = document.getElementById('mermaid-modal-content');
    if (!modal || !modalContent) return;

    currentZoom = 1.0;

    const clonedSvg = originalSvg.cloneNode(true) as SVGElement;
    clonedSvg.removeAttribute('width');
    clonedSvg.removeAttribute('height');
    clonedSvg.style.width = 'auto';
    clonedSvg.style.height = 'auto';
    clonedSvg.style.maxWidth = 'none';
    clonedSvg.style.minWidth = '0';

    const graphicsElement = originalSvg as unknown as SVGGraphicsElement;

    modalContent.style.alignItems = 'flex-start';
    modalContent.style.justifyContent = 'center';
    modalContent.scrollTop = 0;
    modalContent.scrollLeft = 0;

    try {
        if (typeof graphicsElement.getBBox === 'function') {
            const bbox = graphicsElement.getBBox();

            if (bbox.height > window.innerHeight - 80) {
                modalContent.style.alignItems = 'flex-start';
                clonedSvg.style.height = 'auto';
            } else {
                clonedSvg.style.maxHeight = '90vh';
            }

            if (bbox.width > window.innerWidth - 80) {
                modalContent.style.justifyContent = 'flex-start';
            }
        }
    } catch (error) {
        console.warn('Could not get BBox', error);
    }

    modalContent.innerHTML = '';
    modalContent.appendChild(clonedSvg);

    let controls = document.getElementById('mermaid-modal-controls');
    if (controls) controls.remove();

    controls = document.createElement('div');
    controls.id = 'mermaid-modal-controls';
    controls.className = 'mermaid-modal-controls';
    controls.innerHTML = `
        <button class="mermaid-control-btn mermaid-control-step" id="zoom-out" title="Zoom Out">−</button>
        <span class="mermaid-zoom-level" id="mermaid-zoom-level">100%</span>
        <button class="mermaid-control-btn mermaid-control-step" id="zoom-in" title="Zoom In">+</button>
        <span class="mermaid-ctrl-divider"></span>
        <button class="mermaid-control-btn" data-zoom="0.3" title="30%">30%</button>
        <button class="mermaid-control-btn" data-zoom="0.5" title="50%">50%</button>
        <button class="mermaid-control-btn" data-zoom="0.75" title="75%">75%</button>
        <button class="mermaid-control-btn active" data-zoom="1" title="100%">100%</button>
    `;
    modal.appendChild(controls);

    controls.querySelectorAll('[data-zoom]').forEach(button => {
        button.addEventListener('click', event => {
            event.stopPropagation();
            const zoom = parseFloat((button as HTMLElement).dataset.zoom || '1');
            currentZoom = zoom;
            updateZoomLevel(clonedSvg, currentZoom);
            controls!.querySelectorAll('[data-zoom]').forEach(node => node.classList.remove('active'));
            button.classList.add('active');
        });
    });

    const zoomInButton = document.getElementById('zoom-in');
    const zoomOutButton = document.getElementById('zoom-out');

    let isDragging = false;
    let dragStartX = 0;
    let dragStartY = 0;
    let startScrollLeft = 0;
    let startScrollTop = 0;

    modalContent.onmousedown = (event: MouseEvent) => {
        if (event.button !== 0) return;
        isDragging = true;
        dragStartX = event.clientX;
        dragStartY = event.clientY;
        startScrollLeft = modalContent.scrollLeft;
        startScrollTop = modalContent.scrollTop;
        modalContent.classList.add('is-dragging');
        event.preventDefault();
    };

    modalContent.onmousemove = (event: MouseEvent) => {
        if (!isDragging) return;
        modalContent.scrollLeft = startScrollLeft - (event.clientX - dragStartX);
        modalContent.scrollTop = startScrollTop - (event.clientY - dragStartY);
    };

    modalContent.onmouseup = () => {
        isDragging = false;
        modalContent.classList.remove('is-dragging');
    };

    modalContent.onmouseleave = () => {
        isDragging = false;
        modalContent.classList.remove('is-dragging');
    };

    zoomInButton?.addEventListener('click', event => {
        event.stopPropagation();
        if (currentZoom < maxZoom) {
            currentZoom = Math.min(maxZoom, Math.round((currentZoom + 0.05) * 100) / 100);
            updateZoomLevel(clonedSvg, currentZoom);
            syncPresetButtons(controls!);
        }
    });

    zoomOutButton?.addEventListener('click', event => {
        event.stopPropagation();
        if (currentZoom > minZoom) {
            currentZoom = Math.max(minZoom, Math.round((currentZoom - 0.05) * 100) / 100);
            updateZoomLevel(clonedSvg, currentZoom);
            syncPresetButtons(controls!);
        }
    });

    modal.classList.add('active');
    document.body.style.overflow = 'hidden';
}

function syncPresetButtons(controls: HTMLElement) {
    controls.querySelectorAll('[data-zoom]').forEach(button => {
        const zoom = parseFloat((button as HTMLElement).dataset.zoom || '1');
        button.classList.toggle('active', Math.abs(zoom - currentZoom) < 0.001);
    });
}

function closeMermaidModal() {
    const modal = document.getElementById('mermaid-modal');
    if (modal) {
        modal.classList.remove('active');
        document.body.style.overflow = '';
    }
}

function updateZoomLevel(svg: SVGElement, zoom: number) {
    const label = document.getElementById('mermaid-zoom-level');
    if (label) label.textContent = `${Math.round(zoom * 100)}%`;

    if (zoom === 1.0) {
        svg.style.width = 'auto';
        svg.style.height = 'auto';
        svg.style.minWidth = '0';
        svg.style.maxWidth = 'none';

        const graphicsElement = svg as unknown as SVGGraphicsElement;
        try {
            if (typeof graphicsElement.getBBox === 'function') {
                const bbox = graphicsElement.getBBox();
                svg.style.maxHeight = bbox.height < window.innerHeight - 80 ? '90vh' : 'none';
            }
        } catch {
            svg.style.maxHeight = 'none';
        }
    } else {
        svg.style.width = `${Math.round(zoom * 100)}%`;
        svg.style.maxHeight = 'none';
        svg.style.height = 'auto';
    }
}
