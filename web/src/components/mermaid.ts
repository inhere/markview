import { escapeHtml } from '../util';

interface SVGGraphicsElement extends SVGElement {
    getBBox(): DOMRect;
}

interface D3EventListener {
    type: string;
    listener: EventListener;
    options?: AddEventListenerOptions | boolean;
}

interface D3ListenerElement extends SVGElement {
    __on?: D3EventListener[];
}

declare global {
    interface Window {
        openMermaidModal: (index: number) => void;
        closeMermaidModal: () => void;
    }
}

let mermaidInitialized = false;
let mermaidModulePromise: Promise<typeof import('mermaid').default> | null = null;
let d3TransitionPromise: Promise<void> | null = null;
let mermaidCounter = 0;
let modalSetupCompleted = false;
let currentZoom = 1.0;

const minZoom = 0.3;
const maxZoom = 2.0;
const mermaidTooltipEventTypes = new Set(['mouseover', 'mouseout']);

export function buildMermaidContainerId(index: number) {
    return `mermaid-${index}`;
}

export function parseMermaidContainerIndex(containerId: string) {
    const numericIndex = Number(containerId.replace('mermaid-', ''));
    return Number.isFinite(numericIndex) ? numericIndex : null;
}

export function createMermaidCopyButton(source: string, navigatorRef: Navigator = navigator): HTMLButtonElement {
    const button = document.createElement('button');
    button.className = 'mermaid-copy-btn';
    button.type = 'button';
    button.title = '复制 Mermaid 源码';
    button.setAttribute('aria-label', '复制 Mermaid 源码');
    button.innerHTML = '<svg class="copy-icon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg><svg class="check-icon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="display: none;"><polyline points="20 6 9 17 4 12"></polyline></svg>';

    button.addEventListener('click', async event => {
        event.preventDefault();
        event.stopPropagation();

        try {
            await copyText(source, navigatorRef);
            showMermaidCopySuccess(button);
        } catch (error) {
            console.error('复制 Mermaid 源码失败:', error);
        }
    });

    return button;
}

async function copyText(text: string, navigatorRef: Navigator) {
    if (navigatorRef.clipboard?.writeText) {
        await navigatorRef.clipboard.writeText(text);
        return;
    }

    const textarea = document.createElement('textarea');
    textarea.value = text;
    textarea.style.position = 'fixed';
    textarea.style.opacity = '0';
    textarea.style.left = '-9999px';
    document.body.appendChild(textarea);
    textarea.select();
    document.execCommand('copy');
    document.body.removeChild(textarea);
}

function showMermaidCopySuccess(button: HTMLElement) {
    const copyIcon = button.querySelector('.copy-icon') as SVGElement | null;
    const checkIcon = button.querySelector('.check-icon') as SVGElement | null;

    if (copyIcon && checkIcon) {
        copyIcon.style.display = 'none';
        checkIcon.style.display = 'block';
    }

    button.classList.add('copied');
    button.title = '已复制!';

    setTimeout(() => {
        if (copyIcon && checkIcon) {
            copyIcon.style.display = 'block';
            checkIcon.style.display = 'none';
        }
        button.classList.remove('copied');
        button.title = '复制 Mermaid 源码';
    }, 2000);
}

export function ensureD3TransitionSupport() {
    if (!d3TransitionPromise) {
        // Mermaid 的部分图形会在 hover 时调用 d3 selection.transition()，
        // 这里显式加载副作用模块，确保打包后 selection 原型已完成注入。
        d3TransitionPromise = import('d3-transition').then(() => undefined);
    }

    return d3TransitionPromise;
}

export function setupMermaidTooltipGuard(container: HTMLElement) {
    if (container.dataset.tooltipGuard === 'true') {
        return;
    }

    container.addEventListener('mouseover', event => {
        const node = findMermaidNode(event.target);
        if (!node || isMovingWithinNode(node, event.relatedTarget)) {
            return;
        }

        const title = node.getAttribute('title');
        if (title === null) {
            return;
        }

        // Flowchart 内部 tooltip 依赖 d3 transition；这里接管 tooltip，
        // 并阻止 Mermaid 自带 handler 继续调用缺失的 transition()。
        event.stopPropagation();
        if (title) {
            showMermaidTooltip(node, title);
        } else {
            hideMermaidTooltip(node);
        }
    }, true);

    container.addEventListener('mouseout', event => {
        const node = findMermaidNode(event.target);
        if (!node || isMovingWithinNode(node, event.relatedTarget)) {
            return;
        }

        event.stopPropagation();
        hideMermaidTooltip(node);
    }, true);

    container.dataset.tooltipGuard = 'true';
}

export function removeMermaidTooltipListeners(container: HTMLElement) {
    container.querySelectorAll('g.node').forEach(node => {
        if (!(node instanceof SVGElement)) {
            return;
        }

        const d3Listeners = readD3Listeners(node);
        if (!d3Listeners.length) {
            return;
        }

        // Mermaid Flowchart 的 tooltip handler 直接调用 selection.transition()。
        // 清掉 mouseover/mouseout 后由 setupMermaidTooltipGuard 接管 tooltip。
        for (const listener of d3Listeners) {
            if (mermaidTooltipEventTypes.has(listener.type)) {
                node.removeEventListener(listener.type, listener.listener, listener.options);
            }
        }

        node.__on = d3Listeners.filter(listener => !mermaidTooltipEventTypes.has(listener.type));
    });
}

async function getMermaidModule() {
    await ensureD3TransitionSupport();

    if (!mermaidModulePromise) {
        mermaidModulePromise = import('mermaid').then(module => module.default);
    }

    const mermaid = await mermaidModulePromise;
    if (!mermaidInitialized) {
        // 禁用交互功能以避免 d3 transition 错误
        // 当鼠标滑过图表时，mermaid 内部的 d3 会尝试调用 transition 方法
        // 但由于打包问题，transition 可能未正确注入到 selection.prototype
        mermaid.initialize({
            startOnLoad: false,
            flowchart: {
                useMaxWidth: true,
                htmlLabels: true,
                curve: 'basis',
            },
            sequence: {
                useMaxWidth: true,
                wrap: true,
            },
            gantt: {
                useMaxWidth: true,
            },
            // 在这里关闭动画
            themeVariables: {
                'transitionDuration': '0'
            }
        });
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

        const copyButton = createMermaidCopyButton(content || '');

        const mermaidDiv = document.createElement('div');
        mermaidDiv.className = 'mermaid';
        mermaidDiv.textContent = content;

        actions.appendChild(copyButton);
        actions.appendChild(sourceButton);
        actions.appendChild(fullscreenButton);
        container.appendChild(actions);
        container.appendChild(sourcePanel);
        container.appendChild(mermaidDiv);

        pre.replaceWith(container);
    }

    await mermaid.run();

    contentRoot.querySelectorAll('.mermaid-container').forEach(container => {
        if (container instanceof HTMLElement) {
            removeMermaidTooltipListeners(container);
            setupMermaidTooltipGuard(container);
        }
    });
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

function findMermaidNode(target: EventTarget | null) {
    if (!(target instanceof Element)) {
        return null;
    }

    const node = target.closest('g.node');
    return node instanceof SVGElement ? node : null;
}

function isMovingWithinNode(node: SVGElement, relatedTarget: EventTarget | null) {
    return relatedTarget instanceof Node && node.contains(relatedTarget);
}

function getMermaidTooltipElement() {
    let tooltip = document.querySelector('.mermaidTooltip');
    if (!(tooltip instanceof HTMLElement)) {
        tooltip = document.createElement('div');
        tooltip.className = 'mermaidTooltip';
        tooltip.style.opacity = '0';
        document.body.appendChild(tooltip);
    }

    return tooltip;
}

function showMermaidTooltip(node: SVGElement, title: string) {
    const tooltip = getMermaidTooltipElement();
    const rect = node.getBoundingClientRect();

    tooltip.textContent = title;
    tooltip.style.left = `${window.scrollX + rect.left + (rect.right - rect.left) / 2}px`;
    tooltip.style.top = `${window.scrollY + rect.bottom}px`;
    tooltip.style.opacity = '.9';
    tooltip.innerHTML = tooltip.innerHTML.replace(/&lt;br\/&gt;/g, '<br/>');
    node.classList.add('hover');
}

function hideMermaidTooltip(node: SVGElement) {
    const tooltip = getMermaidTooltipElement();

    tooltip.style.opacity = '0';
    node.classList.remove('hover');
}

function readD3Listeners(node: SVGElement) {
    const d3Node = node as D3ListenerElement;
    return Array.isArray(d3Node.__on) ? d3Node.__on : [];
}
