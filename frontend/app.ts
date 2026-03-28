// Mark as ES module so `declare global` works correctly
export {};

import hljs from 'highlight.js/lib/core';
import javascript from 'highlight.js/lib/languages/javascript';
import typescript from 'highlight.js/lib/languages/typescript';
import xml from 'highlight.js/lib/languages/xml';
import css from 'highlight.js/lib/languages/css';
import json from 'highlight.js/lib/languages/json';
import bash from 'highlight.js/lib/languages/bash';
import go from 'highlight.js/lib/languages/go';
import markdown from 'highlight.js/lib/languages/markdown';
import yaml from 'highlight.js/lib/languages/yaml';
import sql from 'highlight.js/lib/languages/sql';
import python from 'highlight.js/lib/languages/python';
import rust from 'highlight.js/lib/languages/rust';

interface SVGGraphicsElement extends SVGElement {
    getBBox(): DOMRect;
}

declare global {
    interface Window {
        openMermaidModal: (index: number) => void;
        closeMermaidModal: () => void;
        toggleMermaidSource: () => void;
    }
}

interface FileTreeNode {
    name: string;
    href?: string;
    matchPath?: string;
    kind: 'directory' | 'file';
    navigable: boolean;
    children?: FileTreeNode[];
}

interface PageSnapshot {
    title: string;
    contentHTML: string;
    fileMetaHTML: string;
    fileTreeJSON: string;
    currentFilePathJSON: string;
}

interface RenderPageOptions {
    hash?: string;
    preserveScrollY?: number | null;
}

interface NavigationTarget {
    kind: 'page' | 'hash';
    url: URL;
}

type HistoryMode = 'push' | 'replace' | 'none';

const CONTENT_SELECTOR = '#content';
const FILE_META_SELECTOR = '#file-meta';
const FILE_TREE_DATA_ID = 'file-tree-data';
const CURRENT_FILE_PATH_DATA_ID = 'current-file-path-data';

let highlightReady = false;
let mermaidInitialized = false;
let mermaidModulePromise: Promise<typeof import('mermaid')> | null = null;
let navigationController: AbortController | null = null;
let navigationRequestId = 0;
let mermaidCounter = 0;
let setupCompleted = false;
let renderedUrl = new URL(window.location.href);

function ensureHighlightLanguages() {
    if (highlightReady) {
        return;
    }

    hljs.registerLanguage('javascript', javascript);
    hljs.registerLanguage('typescript', typescript);
    hljs.registerLanguage('xml', xml);
    hljs.registerLanguage('css', css);
    hljs.registerLanguage('json', json);
    hljs.registerLanguage('bash', bash);
    hljs.registerLanguage('go', go);
    hljs.registerLanguage('markdown', markdown);
    hljs.registerLanguage('yaml', yaml);
    hljs.registerLanguage('sql', sql);
    hljs.registerLanguage('python', python);
    hljs.registerLanguage('rust', rust);

    highlightReady = true;
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

async function enhancePageContent() {
    const contentRoot = document.querySelector(CONTENT_SELECTOR);
    if (!(contentRoot instanceof HTMLElement)) {
        return;
    }

    ensureHighlightLanguages();

    contentRoot.querySelectorAll('pre code').forEach(block => {
        if (!(block instanceof HTMLElement)) {
            return;
        }
        if (block.classList.contains('language-mermaid')) {
            return;
        }
        if (block.dataset.highlighted === 'yes') {
            return;
        }
        hljs.highlightElement(block);
    });

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

        const container = document.createElement('div');
        container.className = 'mermaid-container';
        container.id = `mermaid-${mermaidCounter++}`;
        container.dataset.source = content || '';

        const actions = document.createElement('div');
        actions.className = 'mermaid-actions';

        const fullscreenButton = document.createElement('button');
        fullscreenButton.className = 'mermaid-fullscreen-btn';
        fullscreenButton.innerHTML = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M15 3h6v6M9 21H3v-6M21 3l-7 7M3 21l7-7"/></svg>';
        fullscreenButton.title = 'View Fullscreen';

        const containerId = container.id;
        fullscreenButton.onclick = () => {
            const numericIndex = Number(containerId.replace('mermaid-', ''));
            if (Number.isFinite(numericIndex)) {
                window.openMermaidModal(numericIndex);
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

function generateTOC() {
    const tocList = document.getElementById('toc-list');
    const headers = document.querySelectorAll('#content h1, #content h2, #content h3');

    if (!tocList) {
        return;
    }

    tocList.innerHTML = '';

    if (!headers.length) {
        const empty = document.createElement('li');
        empty.className = 'sidebar-empty';
        empty.textContent = 'No headings';
        tocList.appendChild(empty);
        return;
    }

    headers.forEach((header, index) => {
        if (!header.id) {
            const friendlyId = (header as HTMLElement).innerText
                .toLowerCase()
                .replace(/[^a-z0-9]+/g, '-')
                .replace(/(^-|-$)/g, '');
            header.id = friendlyId || `section-${index}`;
        }

        let level = 'h1';
        if (header.tagName === 'H2') level = 'h2';
        if (header.tagName === 'H3') level = 'h3';

        const li = document.createElement('li');
        li.className = 'toc-item';

        const anchor = document.createElement('a');
        anchor.href = `#${header.id}`;
        anchor.innerText = (header as HTMLElement).innerText;
        anchor.className = `toc-link toc-${level}`;

        li.appendChild(anchor);
        tocList.appendChild(li);
    });
}

function renderFileTree() {
    const treeRoot = document.getElementById('file-tree');
    const treeData = readJSONScript<FileTreeNode[]>(FILE_TREE_DATA_ID);
    const currentFilePath = readJSONScript<string>(CURRENT_FILE_PATH_DATA_ID);

    if (!treeRoot) {
        return;
    }

    treeRoot.innerHTML = '';

    if (!treeData?.length) {
        const empty = document.createElement('div');
        empty.className = 'sidebar-empty';
        empty.textContent = 'No markdown files';
        treeRoot.appendChild(empty);
        return;
    }

    const list = document.createElement('ul');
    list.className = 'file-tree-list';

    treeData.forEach(node => {
        list.appendChild(createTreeNode(node, currentFilePath || ''));
    });

    treeRoot.appendChild(list);

    const activeNode = treeRoot.querySelector('.tree-link.active, .tree-label.active');
    if (activeNode instanceof HTMLElement) {
        activeNode.scrollIntoView({ block: 'nearest' });
    }
}

function createTreeNode(node: FileTreeNode, currentFilePath: string): HTMLLIElement {
    const item = document.createElement('li');
    item.className = 'file-tree-node';

    const row = document.createElement('div');
    row.className = 'file-tree-row';

    const hasChildren = Boolean(node.children?.length);
    const isInCurrentBranch = nodeContainsPath(node, currentFilePath);
    const shouldExpand = hasChildren && isInCurrentBranch;
    if (isInCurrentBranch && hasChildren) {
        item.classList.add('branch-active');
    }

    const toggle = document.createElement('button');
    toggle.type = 'button';
    toggle.className = 'tree-toggle';

    if (hasChildren) {
        toggle.innerHTML = chevronIcon();
        toggle.setAttribute('aria-label', shouldExpand ? 'Collapse folder' : 'Expand folder');
        toggle.setAttribute('aria-expanded', shouldExpand ? 'true' : 'false');
        if (shouldExpand) {
            toggle.classList.add('expanded');
        }
    } else {
        toggle.classList.add('placeholder');
        toggle.setAttribute('aria-hidden', 'true');
    }

    const isActive = Boolean(node.matchPath && node.matchPath === currentFilePath);
    const content = node.navigable && node.href ? document.createElement('a') : document.createElement('span');
    content.className = node.navigable && node.href ? 'tree-link' : 'tree-label';
    if (isActive) {
        content.classList.add('active');
    } else if (isInCurrentBranch && hasChildren) {
        content.classList.add('ancestor');
    }

    if (content instanceof HTMLAnchorElement && node.href) {
        content.href = node.href;
    }

    const icon = document.createElement('span');
    icon.className = 'tree-icon';
    icon.innerHTML = node.kind === 'directory' ? folderIcon() : fileIcon();

    const text = document.createElement('span');
    text.className = 'tree-text';
    text.textContent = node.name;

    content.appendChild(icon);
    content.appendChild(text);

    row.appendChild(toggle);
    row.appendChild(content);
    item.appendChild(row);

    if (hasChildren && node.children) {
        const childList = document.createElement('ul');
        childList.className = 'file-tree-children';
        childList.hidden = !shouldExpand;

        node.children.forEach(child => {
            childList.appendChild(createTreeNode(child, currentFilePath));
        });

        toggle.addEventListener('click', event => {
            event.preventDefault();
            const expanded = toggle.getAttribute('aria-expanded') === 'true';
            const nextExpanded = !expanded;
            toggle.setAttribute('aria-expanded', nextExpanded ? 'true' : 'false');
            toggle.setAttribute('aria-label', nextExpanded ? 'Collapse folder' : 'Expand folder');
            toggle.classList.toggle('expanded', nextExpanded);
            childList.hidden = !nextExpanded;
        });

        item.appendChild(childList);
    }

    return item;
}

function nodeContainsPath(node: FileTreeNode, currentFilePath: string): boolean {
    if (!currentFilePath) {
        return false;
    }
    if (node.matchPath === currentFilePath) {
        return true;
    }
    return node.children?.some(child => nodeContainsPath(child, currentFilePath)) ?? false;
}

function readJSONScript<T>(id: string): T | null {
    const element = document.getElementById(id);
    if (!element?.textContent) {
        return null;
    }

    try {
        return JSON.parse(element.textContent) as T;
    } catch (error) {
        console.warn(`Failed to parse ${id}`, error);
        return null;
    }
}

function highlightTOC() {
    const scrollPos = window.scrollY + 100;
    const headers = document.querySelectorAll('#content h1, #content h2, #content h3');
    let currentId = '';

    headers.forEach(header => {
        if ((header as HTMLElement).offsetTop <= scrollPos) {
            currentId = header.id;
        }
    });

    if (currentId) {
        document.querySelectorAll('.toc-link').forEach(link => {
            link.classList.remove('active');
            if (link.getAttribute('href') === `#${currentId}`) {
                link.classList.add('active');
                const sidebar = document.querySelector('.toc-container');
                if (sidebar) {
                    const linkRect = link.getBoundingClientRect();
                    const sidebarRect = sidebar.getBoundingClientRect();
                    if (linkRect.top < sidebarRect.top || linkRect.bottom > sidebarRect.bottom) {
                        link.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
                    }
                }
            }
        });
    }
}

function setupToolbar() {
    const toolbar = document.getElementById('toolbar');
    if (!toolbar) {
        return;
    }

    const widthButtons = toolbar.querySelectorAll('[data-width]');
    widthButtons.forEach(button => {
        button.addEventListener('click', () => {
            const width = (button as HTMLElement).dataset.width;
            if (!width) {
                return;
            }

            document.documentElement.style.setProperty('--layout-max-width', width);
            widthButtons.forEach(node => node.classList.remove('active'));
            button.classList.add('active');
        });
    });

    const fontIncrease = document.getElementById('font-inc');
    const fontDecrease = document.getElementById('font-dec');
    let currentFontSize = 16;

    const updateFont = () => {
        document.documentElement.style.fontSize = `${currentFontSize}px`;
    };

    fontIncrease?.addEventListener('click', () => {
        if (currentFontSize < 24) {
            currentFontSize++;
            updateFont();
        }
    });

    fontDecrease?.addEventListener('click', () => {
        if (currentFontSize > 12) {
            currentFontSize--;
            updateFont();
        }
    });
}

let currentZoom = 1.0;
const minZoom = 0.3;
const maxZoom = 2.0;

window.openMermaidModal = (index: number) => {
    const container = document.getElementById(`mermaid-${index}`);
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
};

function syncPresetButtons(controls: HTMLElement) {
    controls.querySelectorAll('[data-zoom]').forEach(button => {
        const zoom = parseFloat((button as HTMLElement).dataset.zoom || '1');
        button.classList.toggle('active', Math.abs(zoom - currentZoom) < 0.001);
    });
}

window.closeMermaidModal = () => {
    const modal = document.getElementById('mermaid-modal');
    if (modal) {
        modal.classList.remove('active');
        document.body.style.overflow = '';
    }
};

document.addEventListener('keydown', event => {
    if (event.key === 'Escape') window.closeMermaidModal();
});

function escapeHtml(value: string) {
    return value
        .replaceAll('&', '&amp;')
        .replaceAll('<', '&lt;')
        .replaceAll('>', '&gt;')
        .replaceAll('"', '&quot;')
        .replaceAll("'", '&#39;');
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

async function renderCurrentPage(options: RenderPageOptions = {}) {
    rewriteContentRelativeURLs();
    renderFileTree();
    generateTOC();
    await enhancePageContent();
    highlightTOC();

    const applyScroll = () => {
        if (options.hash) {
            scrollToHash(options.hash);
            return;
        }

        if (typeof options.preserveScrollY === 'number') {
            window.scrollTo({ top: options.preserveScrollY, left: 0, behavior: 'auto' });
            return;
        }

        window.scrollTo({ top: 0, left: 0, behavior: 'auto' });
    };

    requestAnimationFrame(() => {
        applyScroll();
        highlightTOC();
    });
}

function parsePageSnapshot(html: string): PageSnapshot {
    const parser = new DOMParser();
    const nextDocument = parser.parseFromString(html, 'text/html');

    const content = nextDocument.querySelector(CONTENT_SELECTOR);
    const fileMeta = nextDocument.querySelector(FILE_META_SELECTOR);
    const fileTreeScript = nextDocument.getElementById(FILE_TREE_DATA_ID);
    const currentFilePathScript = nextDocument.getElementById(CURRENT_FILE_PATH_DATA_ID);

    if (!(content instanceof HTMLElement)) {
        throw new Error('Missing content node in fetched page');
    }
    if (!(fileMeta instanceof HTMLElement)) {
        throw new Error('Missing file meta node in fetched page');
    }
    if (!fileTreeScript?.textContent) {
        throw new Error('Missing file tree data in fetched page');
    }
    if (!currentFilePathScript?.textContent) {
        throw new Error('Missing current file path data in fetched page');
    }

    return {
        title: nextDocument.title,
        contentHTML: content.innerHTML,
        fileMetaHTML: fileMeta.innerHTML,
        fileTreeJSON: fileTreeScript.textContent,
        currentFilePathJSON: currentFilePathScript.textContent,
    };
}

function applyPageSnapshot(snapshot: PageSnapshot) {
    const content = document.querySelector(CONTENT_SELECTOR);
    const fileMeta = document.querySelector(FILE_META_SELECTOR);
    const fileTreeScript = document.getElementById(FILE_TREE_DATA_ID);
    const currentFilePathScript = document.getElementById(CURRENT_FILE_PATH_DATA_ID);

    if (!(content instanceof HTMLElement) || !(fileMeta instanceof HTMLElement) || !fileTreeScript || !currentFilePathScript) {
        throw new Error('Missing current page mount points');
    }

    document.title = snapshot.title;
    content.innerHTML = snapshot.contentHTML;
    fileMeta.innerHTML = snapshot.fileMetaHTML;
    fileTreeScript.textContent = snapshot.fileTreeJSON;
    currentFilePathScript.textContent = snapshot.currentFilePathJSON;
}

async function fetchPageSnapshot(url: URL): Promise<PageSnapshot> {
    navigationController?.abort();

    const requestId = ++navigationRequestId;
    const controller = new AbortController();
    navigationController = controller;

    const response = await fetch(url.toString(), {
        cache: 'no-store',
        headers: {
            'X-MarkView-Navigation': 'inline',
        },
        signal: controller.signal,
    });

    if (!response.ok) {
        throw new Error(`Navigation request failed with ${response.status}`);
    }

    const contentType = response.headers.get('Content-Type') || '';
    if (!contentType.includes('text/html')) {
        throw new Error(`Unexpected content-type: ${contentType}`);
    }

    const html = await response.text();
    if (requestId !== navigationRequestId) {
        throw new DOMException('Stale navigation response', 'AbortError');
    }

    return parsePageSnapshot(html);
}

async function navigateTo(url: URL, historyMode: HistoryMode, options: { preserveScroll?: boolean } = {}) {
    const preserveScrollY = options.preserveScroll ? window.scrollY : null;

    try {
        const snapshot = await fetchPageSnapshot(url);
        applyPageSnapshot(snapshot);

        if (historyMode === 'push') {
            history.pushState({ markview: true }, '', url.toString());
        } else if (historyMode === 'replace') {
            history.replaceState({ markview: true }, '', url.toString());
        }

        renderedUrl = new URL(url.toString());
        await renderCurrentPage({
            hash: url.hash,
            preserveScrollY,
        });
    } catch (error) {
        if (error instanceof DOMException && error.name === 'AbortError') {
            return;
        }

        console.error('Inline navigation failed, falling back to full page load', error);
        window.location.assign(url.toString());
    }
}

async function refreshCurrentPage() {
    await navigateTo(new URL(window.location.href), 'replace', { preserveScroll: true });
}

function setupInlineNavigation() {
    document.addEventListener('click', event => {
        const target = event.target instanceof Element ? event.target : event.target instanceof Node ? event.target.parentElement : null;
        const anchor = target?.closest('a[href]');
        if (!(anchor instanceof HTMLAnchorElement)) {
            return;
        }

        const navigation = resolveNavigationTarget(anchor, event);
        if (!navigation) {
            return;
        }

        event.preventDefault();

        if (navigation.kind === 'hash') {
            history.pushState({ markview: true }, '', navigation.url.toString());
            renderedUrl = new URL(navigation.url.toString());
            scrollToHash(navigation.url.hash);
            highlightTOC();
            return;
        }

        void navigateTo(navigation.url, 'push');
    });

    window.addEventListener('popstate', () => {
        const url = new URL(window.location.href);
        if (url.pathname === renderedUrl.pathname && url.search === renderedUrl.search && url.hash) {
            renderedUrl = new URL(url.toString());
            scrollToHash(url.hash);
            return;
        }

        void navigateTo(url, 'none');
    });

    history.replaceState({ markview: true }, '', window.location.href);
}

function resolveNavigationTarget(anchor: HTMLAnchorElement, event: MouseEvent): NavigationTarget | null {
    if (event.defaultPrevented || event.button !== 0) {
        return null;
    }

    if (event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) {
        return null;
    }

    if (anchor.target && anchor.target !== '_self') {
        return null;
    }

    if (anchor.hasAttribute('download')) {
        return null;
    }

    const url = new URL(anchor.href, window.location.href);
    const currentUrl = new URL(window.location.href);

    if (url.origin !== currentUrl.origin) {
        return null;
    }

    if (url.pathname === currentUrl.pathname && url.search === currentUrl.search && url.hash) {
        return { kind: 'hash', url };
    }

    if (!isInlineNavigablePath(url.pathname)) {
        return null;
    }

    return { kind: 'page', url };
}

function isInlineNavigablePath(pathname: string): boolean {
    if (pathname === '/') {
        return true;
    }

    const lastSegment = pathname.split('/').filter(Boolean).pop() || '';
    if (!lastSegment) {
        return true;
    }

    return !lastSegment.includes('.') || lastSegment.toLowerCase().endsWith('.md');
}

function scrollToHash(hash: string) {
    if (!hash) {
        return;
    }

    const rawId = decodeURIComponent(hash.replace(/^#/, ''));
    if (!rawId) {
        return;
    }

    const target = document.getElementById(rawId)
        || document.querySelector(`[name="${CSS.escape(rawId)}"]`);

    if (target instanceof HTMLElement) {
        target.scrollIntoView({ behavior: 'auto', block: 'start' });
    }
}

function rewriteContentRelativeURLs() {
    const content = document.querySelector(CONTENT_SELECTOR);
    const currentFilePath = readJSONScript<string>(CURRENT_FILE_PATH_DATA_ID);

    if (!(content instanceof HTMLElement) || !currentFilePath) {
        return;
    }

    const baseURL = buildContentBaseURL(currentFilePath);
    rewriteAttributeURLs(content, 'a[href]', 'href', baseURL);
    rewriteAttributeURLs(content, 'img[src]', 'src', baseURL);
}

function rewriteAttributeURLs(root: HTMLElement, selector: string, attribute: 'href' | 'src', baseURL: URL) {
    root.querySelectorAll(selector).forEach(node => {
        if (!(node instanceof HTMLElement)) {
            return;
        }

        const rawValue = node.getAttribute(attribute);
        if (!rawValue || isAlreadyAbsoluteURL(rawValue)) {
            return;
        }

        try {
            const resolved = new URL(rawValue, baseURL);
            const nextValue = resolved.origin === window.location.origin
                ? `${resolved.pathname}${resolved.search}${resolved.hash}`
                : resolved.toString();
            node.setAttribute(attribute, nextValue);
        } catch (error) {
            console.warn(`Failed to rewrite ${attribute} for`, rawValue, error);
        }
    });
}

function buildContentBaseURL(currentFilePath: string) {
    const normalizedPath = currentFilePath.replace(/\\/g, '/');
    const lastSlashIndex = normalizedPath.lastIndexOf('/');
    const directory = lastSlashIndex >= 0 ? normalizedPath.slice(0, lastSlashIndex + 1) : '';
    return new URL(`/${directory}`, window.location.origin);
}

function isAlreadyAbsoluteURL(value: string) {
    const trimmed = value.trim();
    return trimmed === ''
        || trimmed.startsWith('#')
        || trimmed.startsWith('/')
        || trimmed.startsWith('//')
        || /^[a-zA-Z][a-zA-Z\d+\-.]*:/.test(trimmed);
}

function chevronIcon() {
    return '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><polyline points="9 18 15 12 9 6"></polyline></svg>';
}

function folderIcon() {
    return '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V7z"></path></svg>';
}

function fileIcon() {
    return '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path><polyline points="14 2 14 8 20 8"></polyline></svg>';
}

function setupOnce() {
    if (setupCompleted) {
        return;
    }

    setupToolbar();
    setupInlineNavigation();
    window.addEventListener('scroll', highlightTOC, { passive: true });

    const evtSource = new EventSource('/sse');
    const liveDot = document.getElementById('live-dot');
    const statusText = document.getElementById('status-text');

    evtSource.onmessage = event => {
        if (event.data === 'reload') {
            if (liveDot) liveDot.classList.add('reloading');
            if (statusText) statusText.innerText = 'Syncing...';
            void refreshCurrentPage().finally(() => {
                if (liveDot) liveDot.classList.remove('reloading');
                if (statusText) statusText.innerText = 'Connected';
            });
        }
    };

    evtSource.onerror = () => {
        if (liveDot) liveDot.style.backgroundColor = 'var(--status-warn)';
        if (statusText) statusText.innerText = 'Disconnected';
    };

    setupCompleted = true;
}

document.addEventListener('DOMContentLoaded', async () => {
    setupOnce();
    renderedUrl = new URL(window.location.href);
    await renderCurrentPage({ hash: window.location.hash });
});
