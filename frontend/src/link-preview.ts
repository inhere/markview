// frontend/src/link-preview.ts

import {
    parsePageSnapshot,
    type PageSnapshot,
} from './page';

// 静态资源扩展名
const STATIC_RESOURCE_EXTENSIONS = [
    '.jpg', '.jpeg', '.png', '.gif', '.svg', '.webp', '.avif',
    '.mp4', '.webm', '.mp3', '.ogg', '.wav', '.pdf', '.zip', '.tar', '.gz'
];

function shouldShowPreviewButton(anchor: HTMLAnchorElement): boolean {
    const href = anchor.getAttribute('href');
    if (!href) return false;
    
    // 排除锚点链接
    if (href.startsWith('#')) return false;
    
    // 排除静态资源
    const lowerHref = href.toLowerCase();
    for (const ext of STATIC_RESOURCE_EXTENSIONS) {
        if (lowerHref.endsWith(ext)) return false;
    }
    
    // 排除 download 属性
    if (anchor.hasAttribute('download')) return false;
    
    const url = new URL(anchor.href, window.location.href);
    if (url.origin !== window.location.origin) {
        return false;
    }
    
    // 站内路径：检查是否为 .md 或无扩展名
    const pathname = url.pathname;
    const lastSegment = pathname.split('/').filter(Boolean).pop() || '';
    
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
        
        // 为链接创建包装容器（用于定位按钮）
        const wrapper = document.createElement('span');
        wrapper.className = 'link-preview-wrapper';
        anchor.parentNode?.insertBefore(wrapper, anchor);
        wrapper.appendChild(anchor);
        
        // 创建预览按钮
        const btn = document.createElement('button');
        btn.className = 'link-preview-btn';
        btn.innerHTML = `<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2"/><line x1="12" y1="3" x2="12" y2="21"/></svg>`;
        btn.title = '分屏预览';
        btn.type = 'button';
        
        // hover 显示逻辑
        wrapper.addEventListener('mouseenter', () => {
            btn.classList.add('visible');
        });
        wrapper.addEventListener('mouseleave', () => {
            btn.classList.remove('visible');
        });
        
        // 点击处理
        btn.addEventListener('click', (e) => {
            e.preventDefault();
            e.stopPropagation();
            openPreviewPanel(anchor.href, btn);
        });
        
        wrapper.appendChild(btn);
    }
}

function openPreviewPanel(url: string, triggerButton: HTMLElement): void {
    const panel = document.getElementById('preview-panel');
    if (!panel) return;
    
    // 若点击同一链接的按钮，关闭面板
    if (previewPanelOpen && currentPreviewUrl === url) {
        closePreviewPanel();
        return;
    }
    
    // 更新状态
    currentPreviewUrl = url;
    currentTriggerButton = triggerButton;
    previewPanelOpen = true;
    
    // 显示面板
    panel.style.display = 'flex';
    document.body.classList.add('preview-active');
    
    // 绑定关闭按钮
    const closeBtn = panel.querySelector('.preview-close');
    if (closeBtn) {
        closeBtn.onclick = closePreviewPanel;
    }
    
    // 重置面板状态
    resetPanelState();
    
    console.log('[link-preview] loading:', url);
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
        // 构造 URL（添加 inline navigation header）
        const targetUrl = new URL(url, window.location.href);
        
        // fetch 页面
        const response = await fetch(targetUrl.toString(), {
            headers: { 'X-MarkView-Navigation': 'inline' },
        });
        
        if (!response.ok) {
            throw new Error(`Failed to fetch: ${response.status}`);
        }
        
        const html = await response.text();
        
        // 解析页面，只提取 #content
        const parser = new DOMParser();
        const doc = parser.parseFromString(html, 'text/html');
        const content = doc.querySelector('#content');
        
        if (!(content instanceof HTMLElement)) {
            throw new Error('Missing #content in fetched page');
        }
        
        // 渲染到 preview-body
        const bodyEl = panel.querySelector('.preview-body');
        const loadingEl = panel.querySelector('.preview-loading');
        
        if (bodyEl) {
            bodyEl.innerHTML = content.innerHTML;
            // 添加 paper 样式给预览内容
            bodyEl.style.padding = '20px';
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