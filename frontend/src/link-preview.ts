// frontend/src/link-preview.ts

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
    // TODO: 实现
}

function openPreviewPanel(url: string, triggerButton: HTMLElement): void {
    // TODO: 实现
}

function closePreviewPanel(): void {
    // TODO: 实现
}