// frontend/src/link-preview.ts

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
    
    // 排除 target="_blank" (外站链接已有此属性)
    if (anchor.target === '_blank') {
        // 外站链接仍可预览（iframe方式）
        return true;
    }
    
    // 站内链接需要是 Markdown 文件
    const url = new URL(anchor.href, window.location.href);
    if (url.origin !== window.location.origin) {
        return true; // 站外链接，iframe 预览
    }
    
    // 站内路径：检查是否为 .md 或无扩展名
    const pathname = url.pathname;
    const lastSegment = pathname.split('/').filter(Boolean).pop() || '';
    
    if (lastSegment.includes('.')) {
        return lastSegment.toLowerCase().endsWith('.md');
    }
    
    return true; // 无扩展名的路径视为可预览
}

function isInternalLink(anchor: HTMLAnchorElement): boolean {
    const url = new URL(anchor.href, window.location.href);
    return url.origin === window.location.origin;
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

function enhanceLinksInContent(root: HTMLElement): void {
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
    // TODO: 实现
}

function closePreviewPanel(): void {
    // TODO: 实现
}