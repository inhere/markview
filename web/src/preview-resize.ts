import {
    DEFAULT_PREVIEW_WIDTH,
    MAX_PREVIEW_WIDTH,
    MIN_PREVIEW_WIDTH,
    normalizePreviewWidth,
    persistPreviewWidth,
} from './preferences';

const PREVIEW_WIDTH_VAR = '--preview-width';
const RESIZE_HANDLE_ID = 'preview-resize-handle';
const PREVIEW_PANEL_ID = 'preview-panel';

let isResizing = false;
let startX = 0;
let startWidth = 0;

export function initPreviewResize() {
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
    startWidth = getPreviewPanelWidth();

    const handle = document.getElementById(RESIZE_HANDLE_ID);
    if (handle) {
        handle.classList.add('is-resizing');
    }
    document.body.classList.add('preview-is-resizing');

    document.body.style.cursor = 'col-resize';
    document.body.style.userSelect = 'none';

    event.preventDefault();
}

function doResize(event: MouseEvent) {
    if (!isResizing) return;

    const deltaX = startX - event.clientX;
    setPreviewWidth(startWidth + deltaX, false);
}

function endResize() {
    if (!isResizing) return;

    isResizing = false;

    const handle = document.getElementById(RESIZE_HANDLE_ID);
    if (handle) {
        handle.classList.remove('is-resizing');
    }
    document.body.classList.remove('preview-is-resizing');

    document.body.style.cursor = '';
    document.body.style.userSelect = '';

    persistPreviewWidth(getCurrentPreviewWidth());
}

function getPreviewPanelWidth(): number {
    const panel = document.getElementById(PREVIEW_PANEL_ID);
    if (panel) {
        const rect = panel.getBoundingClientRect();
        if (rect.width > 0) {
            return rect.width;
        }
    }

    return getCurrentPreviewWidth();
}

export function setPreviewWidth(width: number, persist = true) {
    const clampedWidth = Math.max(MIN_PREVIEW_WIDTH, Math.min(MAX_PREVIEW_WIDTH, Math.round(width)));
    document.documentElement.style.setProperty(PREVIEW_WIDTH_VAR, `${clampedWidth}px`);
    if (persist) {
        persistPreviewWidth(clampedWidth);
    }
}

export function getCurrentPreviewWidth(): number {
    const value = document.documentElement.style.getPropertyValue(PREVIEW_WIDTH_VAR);
    const normalized = normalizePreviewWidth(value);
    return normalized ?? DEFAULT_PREVIEW_WIDTH;
}

export function applyInitialPreviewWidth(width: number | null) {
    if (width === null) return;
    document.documentElement.style.setProperty(PREVIEW_WIDTH_VAR, `${width}px`);
}
