// web/src/link-preview.ts

import { ensureHighlightLanguages, safeHighlightElement } from './highlight';
import { enhanceMermaidContent } from './mermaid';
import { enhanceCodeBlocks } from './code-copy';
import {
    parsePageSnapshot,
    type PageSnapshot,
} from './page';
import { escapeHtml } from './util';
import {
    DEFAULT_APP_CONFIG,
    normalizePreviewExts,
} from './app-config';

async function enhancePreviewContent(contentRoot: HTMLElement) {
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
        safeHighlightElement(block);
    });

    enhanceCodeBlocks(contentRoot);
    await enhanceMermaidContent(contentRoot);
}

// 静态资源扩展名
const STATIC_RESOURCE_EXTENSIONS = [
    '.jpg', '.jpeg', '.png', '.gif', '.svg', '.webp', '.avif',
    '.mp4', '.webm', '.mp3', '.ogg', '.wav', '.pdf', '.zip', '.tar', '.gz'
];

let previewableContentExtensions = normalizePreviewExts(DEFAULT_APP_CONFIG.previewExts);

export function configureLinkPreview(options: { previewExts: unknown }): void {
    previewableContentExtensions = normalizePreviewExts(options.previewExts);
}

export function isPreviewableContentPath(pathname: string, previewExts = previewableContentExtensions): boolean {
    const lowerPath = pathname.split(/[?#]/, 1)[0].toLowerCase();
    return normalizePreviewExts(previewExts)
        // Markdown pages use the page/navigation preview path, not raw content rendering.
        .filter(ext => ext !== '.md')
        .some(ext => lowerPath.endsWith(ext));
}

export function detectPreviewFileLanguage(pathname: string): string | null {
    const lowerPath = pathname.toLowerCase();
    if (lowerPath.endsWith('.json') || lowerPath.endsWith('.jsonl')) {
        return 'json';
    }
    if (lowerPath.endsWith('.yaml') || lowerPath.endsWith('.yml')) {
        return 'yaml';
    }
    if (lowerPath.endsWith('.toml')) {
        return 'toml';
    }
    return null;
}

export function buildHighlightedFilePreview(content: string, language: string): string {
    return `<pre class="preview-file-code"><code class="language-${language}">${escapeHtml(content)}</code></pre>`;
}

function shouldShowPreviewButton(anchor: HTMLAnchorElement): boolean {
    const href = anchor.getAttribute('href');
    if (!href) return false;

    // 排除锚点链接
    if (href.startsWith('#')) return false;

    const url = new URL(anchor.href, window.location.href);
    if (url.origin !== window.location.origin) {
        return false;
    }

    const pathname = url.pathname;
    const lowerPathname = pathname.toLowerCase();

    // 排除静态资源
    for (const ext of STATIC_RESOURCE_EXTENSIONS) {
        if (lowerPathname.endsWith(ext)) return false;
    }

    // 排除 download 属性
    if (anchor.hasAttribute('download')) return false;

    // 站内路径：检查是否为 .md 或无扩展名
    const lastSegment = pathname.split('/').filter(Boolean).pop() || '';

    if (isPreviewableContentPath(pathname)) {
        return true;
    }

    if (lastSegment.includes('.')) {
        return lastSegment.toLowerCase().endsWith('.md');
    }

    return true;
}

export function setupLinkPreview(): void {
    if (window.innerWidth < 1024) {
        return; // 移动端不启用
    }

    // 监听 ESC 键关闭面板
    document.addEventListener('keydown', handleEscapeKey);

    // 增强当前内容区域的链接
    const content = document.querySelector('#content');
    if (content instanceof HTMLElement) {
        enhanceLinksInContent(content);
        console.log('Link preview: enhanced links');
    }
}

// 状态管理
let currentPreviewUrl: string | null = null;
let currentTriggerButton: HTMLElement | null = null;
let previewPanelOpen = false;

function createPreviewStateEvent(): Event {
    const CustomEventCtor = document.defaultView?.CustomEvent ?? CustomEvent;
    return new CustomEventCtor('markview:preview-state-changed', {
        detail: { open: previewPanelOpen },
    });
}

function emitPreviewStateChanged(): void {
    document.dispatchEvent(createPreviewStateEvent());
}

function handleEscapeKey(event: KeyboardEvent): void {
    if (event.key === 'Escape' && previewPanelOpen) {
        closePreviewPanel();
    }
}

export function enhanceLinksInContent(root: HTMLElement): void {
    const anchors = root.querySelectorAll('a[href]');

    for (const anchor of anchors) {
        if (!(anchor instanceof HTMLAnchorElement)) continue;
        if (!shouldShowPreviewButton(anchor)) continue;

        const existingWrapper = anchor.closest('.link-preview-wrapper');
        if (existingWrapper instanceof HTMLElement) {
            if (existingWrapper.querySelector('.link-preview-btn')) {
                continue;
            }

            existingWrapper.appendChild(createPreviewButton(anchor));
            continue;
        }

        // 为链接创建包装容器（用于定位按钮）
        const wrapper = document.createElement('span');
        wrapper.className = 'link-preview-wrapper';
        anchor.parentNode?.insertBefore(wrapper, anchor);
        wrapper.appendChild(anchor);

        // 创建预览按钮
        const btn = createPreviewButton(anchor);

        // hover 显示逻辑
        wrapper.addEventListener('mouseenter', () => {
            btn.classList.add('visible');
        });
        wrapper.addEventListener('mouseleave', () => {
            btn.classList.remove('visible');
        });

        wrapper.appendChild(btn);
    }
}

function createPreviewButton(anchor: HTMLAnchorElement): HTMLButtonElement {
    const btn = document.createElement('button');
    btn.className = 'link-preview-btn';
    btn.innerHTML = `<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2"/><line x1="12" y1="3" x2="12" y2="21"/></svg>`;
    btn.title = '分屏预览';
    btn.type = 'button';

    btn.addEventListener('click', (e) => {
        e.preventDefault();
        e.stopPropagation();
        openPreviewPanel(anchor.href, btn);
    });

    return btn;
}

export function openPreviewPanel(url: string, triggerButton?: HTMLElement | null): void {
    const panel = document.getElementById('preview-panel');
    if (!panel) return;

    // 若点击同一链接的按钮，关闭面板
    if (previewPanelOpen && currentPreviewUrl === url) {
        closePreviewPanel();
        return;
    }

    // 更新状态
    currentPreviewUrl = url;
    currentTriggerButton = triggerButton ?? null;
    previewPanelOpen = true;

    // 显示面板
    panel.style.display = 'flex';
    document.body.classList.add('preview-active');
    emitPreviewStateChanged();

    // 绑定关闭按钮
    const closeBtn = panel.querySelector('.preview-close');
    if (closeBtn) {
        closeBtn.onclick = closePreviewPanel;
    }

    // 重置面板状态
    resetPanelState();

    loadInternalContent(url);
}

function closePreviewPanel(): void {
    const panel = document.getElementById('preview-panel');
    if (!panel) return;

    // 隐藏面板
    panel.style.display = 'none';
    document.body.classList.remove('preview-active');

    currentPreviewUrl = null;
    currentTriggerButton = null;
    previewPanelOpen = false;
    emitPreviewStateChanged();

    resetPanelState();
}

function resetPanelState(): void {
    const panel = document.getElementById('preview-panel');
    if (!panel) return;

    const loading = panel.querySelector('.preview-loading');
    const body = panel.querySelector('.preview-body');
    const error = panel.querySelector('.preview-error');

    if (loading) loading.style.display = 'flex';
    if (body) body.innerHTML = '';
    if (error) error.classList.remove('visible');
}

async function loadInternalContent(url: string): Promise<void> {
    const panel = document.getElementById('preview-panel');
    if (!panel) return;

    try {
        const targetUrl = new URL(url, window.location.href);
        const isContentPreview = isPreviewableContentPath(targetUrl.pathname);
        if (!isContentPreview) {
            targetUrl.searchParams.set('q', 'main');
        }

        console.log('[link-preview] loading:', targetUrl.toString());
        const response = await fetch(targetUrl.toString(), {
            headers: { 'X-MarkView-Navigation': 'inline' },
        });

        if (!response.ok) {
            throw new Error(`Failed to fetch: ${response.status}`);
        }

        const contentType = response.headers.get('Content-Type') || '';
        let contentHTML: string;
        let docTitle = decodeURIComponent(targetUrl.pathname.split('/').filter(Boolean).pop() || url);

        if (isContentPreview) {
            const language = detectPreviewFileLanguage(targetUrl.pathname) || 'plaintext';
            contentHTML = buildHighlightedFilePreview(await response.text(), language);
        } else if (contentType.includes('application/json')) {
            const data = await response.json();
            contentHTML = data.contentHTML;
            docTitle = data.title || url;
        } else {
            const html = await response.text();
            const parser = new DOMParser();
            const doc = parser.parseFromString(html, 'text/html');
            const content = doc.querySelector('#content');

            if (!(content instanceof HTMLElement)) {
                throw new Error('Missing #content in fetched page');
            }

            contentHTML = content.innerHTML;

            // 获取第一个 h1 标题作为文档标题
            const h1 = content.querySelector('h1');
            if (h1) {
                docTitle = h1.textContent || docTitle;
            }
        }

        const titleEl = document.getElementById('preview-title');
        if (titleEl) {
            titleEl.textContent = " 📖 " + docTitle;
        }

        const bodyEl = panel.querySelector('.preview-body');
        const loadingEl = panel.querySelector('.preview-loading');

        if (bodyEl) {
            bodyEl.innerHTML = contentHTML;
            bodyEl.style.padding = '20px';
            await enhancePreviewContent(bodyEl);
        }
        if (loadingEl) loadingEl.style.display = 'none';

    } catch (error) {
        console.error('Internal content load failed:', error);
        showErrorState();
    }
}

function showErrorState(): void {
    const panel = document.getElementById('preview-panel');
    if (!panel) return;

    const loading = panel.querySelector('.preview-loading');
    const error = panel.querySelector('.preview-error');

    if (loading) loading.style.display = 'none';
    if (error) {
        error.classList.add('visible');
        setTimeout(closePreviewPanel, 3000);
    }
}
