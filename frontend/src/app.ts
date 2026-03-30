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
import powershell from 'highlight.js/lib/languages/powershell';
import { setupLiveReloadStatus } from './live-status';
import { enhanceMermaidContent, setupMermaidModal } from './mermaid';
import {
    applyPageSnapshot,
    parsePageSnapshot,
    type PageSnapshot,
} from './page';
import {
    DEFAULT_FONT_SIZE,
    MAX_FONT_SIZE,
    MIN_FONT_SIZE,
    persistFontSize,
    persistLayoutWidth,
    readStoredPreferences,
    readSidebarPreferences,
    type LayoutWidth,
} from './preferences';
import {
    generateTOC,
    highlightTOC,
    renderFileTree,
    setupSidebarCollapse,
} from './sidebar';
import {
    applyInitialSidebarWidth,
    initSidebarResize,
} from './sidebar-resize';
import {
    buildContentBaseURL,
    isInlineNavigablePath,
    readJSONScript,
    rewriteAttributeURLs,
    scrollToHash,
} from './util';

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
const PAGE_MOUNT_SELECTORS = {
    contentSelector: CONTENT_SELECTOR,
    fileMetaSelector: FILE_META_SELECTOR,
    fileTreeDataId: FILE_TREE_DATA_ID,
    currentFilePathDataId: CURRENT_FILE_PATH_DATA_ID,
} as const;

let highlightReady = false;
let navigationController: AbortController | null = null;
let navigationRequestId = 0;
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
    hljs.registerLanguage('powershell', powershell);

    highlightReady = true;
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
    await enhanceMermaidContent(contentRoot);
}

function setupToolbar() {
    const toolbar = document.getElementById('toolbar');
    if (!toolbar) {
        return;
    }

    const widthButtons = toolbar.querySelectorAll('[data-width]');
    const fontReset = document.getElementById('font-reset');
    widthButtons.forEach(button => {
        button.addEventListener('click', () => {
            const width = (button as HTMLElement).dataset.width as LayoutWidth | undefined;
            if (!width) {
                return;
            }

            applyLayoutWidth(widthButtons, width);
        });
    });

    const fontIncrease = document.getElementById('font-inc');
    const fontDecrease = document.getElementById('font-dec');
    const storedPreferences = readStoredPreferences();
    let currentFontSize = storedPreferences.fontSize;

    applyLayoutWidth(widthButtons, storedPreferences.layoutWidth);
    applyFontSize(currentFontSize);

    fontIncrease?.addEventListener('click', () => {
        if (currentFontSize < MAX_FONT_SIZE) {
            currentFontSize++;
            applyFontSize(currentFontSize);
        }
    });

    fontDecrease?.addEventListener('click', () => {
        if (currentFontSize > MIN_FONT_SIZE) {
            currentFontSize--;
            applyFontSize(currentFontSize);
        }
    });

    fontReset?.addEventListener('click', () => {
        currentFontSize = DEFAULT_FONT_SIZE;
        applyFontSize(currentFontSize);
    });
}

async function renderCurrentPage(options: RenderPageOptions = {}) {
    rewriteContentRelativeURLs();
    renderFileTree({
        treeRootId: 'file-tree',
        treeDataId: FILE_TREE_DATA_ID,
        currentFilePathDataId: CURRENT_FILE_PATH_DATA_ID,
    });
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

    return parsePageSnapshot(html, PAGE_MOUNT_SELECTORS);
}

async function navigateTo(url: URL, historyMode: HistoryMode, options: { preserveScroll?: boolean } = {}) {
    const preserveScrollY = options.preserveScroll ? window.scrollY : null;

    try {
        const snapshot = await fetchPageSnapshot(url);
        applyPageSnapshot(snapshot, PAGE_MOUNT_SELECTORS);

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

function setupOnce() {
    if (setupCompleted) {
        return;
    }

    setupToolbar();
    setupInlineNavigation();
    setupMermaidModal();
    // Sidebar collapse and resize
    const sidebarPrefs = readSidebarPreferences();
    applyInitialSidebarWidth(sidebarPrefs.sidebarWidth);
    initSidebarResize();
    setupSidebarCollapse();
    window.addEventListener('scroll', () => {
        highlightTOC();
    }, { passive: true });

    const evtSource = new EventSource('/sse');
    const liveDot = document.getElementById('live-dot');
    const statusText = document.getElementById('status-text');
    setupLiveReloadStatus(evtSource, liveDot, statusText, refreshCurrentPage);

    setupCompleted = true;
}

function applyLayoutWidth(widthButtons: NodeListOf<Element>, width: LayoutWidth) {
    document.documentElement.style.setProperty('--layout-max-width', width);
    persistLayoutWidth(width);
    widthButtons.forEach(node => {
        node.classList.toggle('active', (node as HTMLElement).dataset.width === width);
    });
}

function applyFontSize(fontSize: number) {
    document.documentElement.style.fontSize = `${fontSize}px`;
    persistFontSize(fontSize);
}

document.addEventListener('DOMContentLoaded', async () => {
    setupOnce();
    renderedUrl = new URL(window.location.href);
    await renderCurrentPage({ hash: window.location.hash });
});
