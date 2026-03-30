import {
    MIN_SIDEBAR_WIDTH,
    MAX_SIDEBAR_WIDTH,
    DEFAULT_SIDEBAR_WIDTH,
    persistSidebarWidth,
} from './preferences';

const SIDEBAR_WIDTH_VAR = '--sidebar-width';
const RESIZE_HANDLE_ID = 'sidebar-resize-handle';
const SIDEBAR_SELECTOR = '.sidebar';

let isResizing = false;
let startX = 0;
let startWidth = 0;

export function initSidebarResize() {
    const handle = document.getElementById(RESIZE_HANDLE_ID);
    if (!handle) return;

    handle.addEventListener('mousedown', startResize);
    document.addEventListener('mousemove', doResize);
    document.addEventListener('mouseup', endResize);
}

function startResize(event: MouseEvent) {
    if (event.button !== 0) return;
    
    isResizing = true;
    startX = event.clientX;
    
    const sidebar = document.querySelector(SIDEBAR_SELECTOR);
    if (sidebar) {
        const rect = sidebar.getBoundingClientRect();
        startWidth = rect.width;
    }
    
    const handle = document.getElementById(RESIZE_HANDLE_ID);
    if (handle) {
        handle.classList.add('is-resizing');
    }
    
    document.body.style.cursor = 'col-resize';
    document.body.style.userSelect = 'none';
    
    event.preventDefault();
}

function doResize(event: MouseEvent) {
    if (!isResizing) return;
    
    const deltaX = startX - event.clientX;
    const newWidth = Math.max(MIN_SIDEBAR_WIDTH, Math.min(MAX_SIDEBAR_WIDTH, startWidth - deltaX));
    
    document.documentElement.style.setProperty(SIDEBAR_WIDTH_VAR, `${newWidth}px`);
}

function endResize() {
    if (!isResizing) return;
    
    isResizing = false;
    
    const handle = document.getElementById(RESIZE_HANDLE_ID);
    if (handle) {
        handle.classList.remove('is-resizing');
    }
    
    document.body.style.cursor = '';
    document.body.style.userSelect = '';
    
    const currentWidth = getCurrentSidebarWidth();
    persistSidebarWidth(currentWidth);
}

export function setSidebarWidth(width: number) {
    const clampedWidth = Math.max(MIN_SIDEBAR_WIDTH, Math.min(MAX_SIDEBAR_WIDTH, width));
    document.documentElement.style.setProperty(SIDEBAR_WIDTH_VAR, `${clampedWidth}px`);
    persistSidebarWidth(clampedWidth);
}

export function getCurrentSidebarWidth(): number {
    const value = document.documentElement.style.getPropertyValue(SIDEBAR_WIDTH_VAR);
    if (!value) return DEFAULT_SIDEBAR_WIDTH;
    
    const parsed = Number.parseInt(value.replace('px', ''), 10);
    if (Number.isNaN(parsed)) return DEFAULT_SIDEBAR_WIDTH;
    
    return parsed;
}

export function applyInitialSidebarWidth(width: number) {
    document.documentElement.style.setProperty(SIDEBAR_WIDTH_VAR, `${width}px`);
}