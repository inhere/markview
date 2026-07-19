import { renderFileTree, type FileTreeNode } from './sidebar';
import { projectURL } from '../project-url';

interface ReloadEventSource {
    onopen: null | (() => void);
    onerror: null | (() => void);
    onmessage: null | ((event: { data: string }) => void);
}

interface StatusDot {
    style: {
        backgroundColor: string;
    };
    classList: {
        add(name: string): void;
        remove(name: string): void;
    };
}

interface StatusText {
    innerText: string;
}

interface ReloadMessage {
    type: 'reload';
    files: string[];
    action?: 'create';
}

const CURRENT_FILE_PATH_DATA_ID = 'current-file-path-data';
const FILE_TREE_DATA_ID = 'file-tree-data';
const FILE_TREE_ROOT_ID = 'file-tree';

function getCurrentFilePath(): string | null {
    const script = document.getElementById(CURRENT_FILE_PATH_DATA_ID);
    if (!(script instanceof HTMLScriptElement) || !script.textContent) {
        return null;
    }
    try {
        return JSON.parse(script.textContent) as string;
    } catch {
        return null;
    }
}

function shouldRefreshCurrentPage(changedFiles: string[]): boolean {
    if (changedFiles.length === 0) {
        return true;
    }

    const currentPath = getCurrentFilePath();
    if (!currentPath) {
        return true;
    }

    const normalizedCurrentPath = currentPath.replace(/\\/g, '/');

    return changedFiles.some(file => {
        const normalizedFile = file.replace(/\\/g, '/');
        return normalizedFile === normalizedCurrentPath;
    });
}

let toastContainer: HTMLElement | null = null;
let activeToast: HTMLElement | null = null;
let toastTimeout: number | null = null;

function createToastContainer(): HTMLElement {
    if (toastContainer) {
        return toastContainer;
    }

    toastContainer = document.createElement('div');
    toastContainer.id = 'file-change-toast-container';
    toastContainer.className = 'toast-container';
    document.body.appendChild(toastContainer);
    return toastContainer;
}

function hideToast(): void {
    if (activeToast) {
        // 保存对当前 toast 的引用，避免被后续创建的 toast 覆盖
        const toastToRemove = activeToast;
        activeToast = null;

        toastToRemove.classList.remove('toast-visible');
        toastToRemove.classList.add('toast-hiding');

        setTimeout(() => {
            if (toastToRemove.parentNode) {
                toastToRemove.parentNode.removeChild(toastToRemove);
            }
        }, 300);
    }

    if (toastTimeout) {
        clearTimeout(toastTimeout);
        toastTimeout = null;
    }
}

function showFileChangeToast(files: string[]): void {
    hideToast();

    const container = createToastContainer();

    const toast = document.createElement('div');
    toast.className = 'file-change-toast';

    const content = document.createElement('div');
    content.className = 'toast-content';

    const icon = document.createElement('span');
    icon.className = 'toast-icon';
    icon.innerHTML = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path><polyline points="14 2 14 8 20 8"></polyline></svg>`;

    const message = document.createElement('div');
    message.className = 'toast-message';

    const label = document.createElement('span');
    label.className = 'toast-label';
    label.textContent = '文件变动';
    message.appendChild(label);

    files.slice(0, 3).forEach(file => {
        const normalizedPath = normalizeFilePath(file);
        const link = document.createElement('a');
        link.className = 'toast-file';
        link.href = fileChangeURL(normalizedPath);
        link.textContent = normalizedPath;
        message.appendChild(link);
    });

    if (files.length > 3) {
        const count = document.createElement('span');
        count.className = 'toast-count';
        count.textContent = `还有 ${files.length - 3} 个文件`;
        message.appendChild(count);
    }

    const closeBtn = document.createElement('button');
    closeBtn.className = 'toast-close';
    closeBtn.innerHTML = '×';
    closeBtn.onclick = (e) => {
        e.stopPropagation();
        hideToast();
    };

    content.appendChild(icon);
    content.appendChild(message);
    toast.appendChild(content);
    toast.appendChild(closeBtn);

    container.appendChild(toast);
    activeToast = toast;

    requestAnimationFrame(() => {
        toast.classList.add('toast-visible');
    });

    toastTimeout = window.setTimeout(hideToast, 4000);
}

function parseReloadMessage(data: string): ReloadMessage | null {
    if (data === 'reload') {
        return { type: 'reload', files: [] };
    }

    try {
        const msg = JSON.parse(data) as ReloadMessage;
        if (msg.type === 'reload' && Array.isArray(msg.files)) {
            return msg;
        }
    } catch {
        return null;
    }

    return null;
}

async function refreshFileTree(): Promise<void> {
    try {
        const response = await fetch(projectURL('/api/file-tree'));
        if (!response.ok) {
            return;
        }

        const fileTree = await response.json();

        const scriptEl = document.getElementById(FILE_TREE_DATA_ID);
        if (scriptEl instanceof HTMLScriptElement) {
            scriptEl.textContent = JSON.stringify(fileTree);

            renderFileTree({
                treeRootId: FILE_TREE_ROOT_ID,
                treeDataId: FILE_TREE_DATA_ID,
                currentFilePathDataId: CURRENT_FILE_PATH_DATA_ID,
            });
        }
    } catch (e) {
        console.error('Failed to refresh file tree:', e);
    }
}

function fileTreeContainsPath(nodes: FileTreeNode[], targetPath: string): boolean {
    const normalizedTargetPath = normalizeFilePath(targetPath);
    return nodes.some(node => {
        if (node.matchPath && normalizeFilePath(node.matchPath) === normalizedTargetPath) {
            return true;
        }
        return node.children ? fileTreeContainsPath(node.children, normalizedTargetPath) : false;
    });
}

function hasFilesMissingFromLocalTree(files: string[]): boolean {
    if (files.length === 0) {
        return false;
    }

    const scriptEl = document.getElementById(FILE_TREE_DATA_ID);
    if (!(scriptEl instanceof HTMLScriptElement) || !scriptEl.textContent) {
        return true;
    }

    try {
        const fileTree = JSON.parse(scriptEl.textContent) as FileTreeNode[];
        if (!Array.isArray(fileTree)) {
            return true;
        }
        // 新建文件在本地树里通常还不存在，需要刷新树来补齐导航节点。
        return files.some(file => !fileTreeContainsPath(fileTree, file));
    } catch {
        return true;
    }
}

function normalizeFilePath(path: string): string {
    return path.replace(/\\/g, '/').replace(/^\/+/, '');
}

export function fileChangeURL(path: string): string {
    return projectURL('/' + normalizeFilePath(path).split('/').map(encodeURIComponent).join('/'));
}

export function setupLiveReloadStatus(
    evtSource: ReloadEventSource,
    liveDot: StatusDot | null,
    statusText: StatusText | null,
    refreshCurrentPage: () => Promise<void>,
) {
    let hasOpened = false;

    evtSource.onopen = () => {
        if (hasOpened) {
            void refreshFileTree();
        }
        hasOpened = true;
        setLiveState(liveDot, statusText);
    };

    evtSource.onmessage = event => {
        console.log('Received message:', event.data);
        const msg = parseReloadMessage(event.data);
        if (!msg) {
            return;
        }

        if (msg.files.length > 0) {
            showFileChangeToast(msg.files);
        }

        const needsPageRefresh = shouldRefreshCurrentPage(msg.files);
        const needsFileTreeRefresh = hasFilesMissingFromLocalTree(msg.files);

        if (needsPageRefresh) {
            if (liveDot) {
                liveDot.classList.add('reloading');
            }
            if (statusText) {
                statusText.innerText = 'Syncing...';
            }

            void refreshCurrentPage().finally(() => {
                if (liveDot) {
                    liveDot.classList.remove('reloading');
                }
                setLiveState(liveDot, statusText);
            });
        }

        if (needsFileTreeRefresh || (!needsPageRefresh && msg.action === 'create')) {
            void refreshFileTree();
        }
    };

    evtSource.onerror = () => {
        if (liveDot) {
            liveDot.style.backgroundColor = 'var(--status-warn)';
        }
        if (statusText) {
            statusText.innerText = 'Offline';
        }

        hideToast();
    };
}

function setLiveState(liveDot: StatusDot | null, statusText: StatusText | null) {
    if (liveDot) {
        liveDot.style.backgroundColor = '';
        liveDot.classList.remove('reloading');
    }
    if (statusText) {
        statusText.innerText = 'Live';
    }
}
