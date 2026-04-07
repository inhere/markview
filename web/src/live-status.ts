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
}

// Toast notification container and state
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
        activeToast.classList.remove('toast-visible');
        activeToast.classList.add('toast-hiding');
        
        // Remove after animation completes
        setTimeout(() => {
            if (activeToast && activeToast.parentNode) {
                activeToast.parentNode.removeChild(activeToast);
            }
            activeToast = null;
        }, 300);
    }
    
    if (toastTimeout) {
        clearTimeout(toastTimeout);
        toastTimeout = null;
    }
}

function showFileChangeToast(files: string[]): void {
    // Hide any existing toast first
    hideToast();
    
    const container = createToastContainer();
    
    // Create toast element
    const toast = document.createElement('div');
    toast.className = 'file-change-toast';
    
    // Create content
    const content = document.createElement('div');
    content.className = 'toast-content';
    
    // Icon
    const icon = document.createElement('span');
    icon.className = 'toast-icon';
    icon.innerHTML = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path><polyline points="14 2 14 8 20 8"></polyline></svg>`;
    
    // Message
    const message = document.createElement('div');
    message.className = 'toast-message';
    
    if (files.length === 1) {
        const fileName = files[0].split('/').pop() || files[0];
        message.innerHTML = `<span class="toast-label">文件变动</span><span class="toast-file">${fileName}</span>`;
    } else {
        message.innerHTML = `<span class="toast-label">文件变动</span><span class="toast-count">${files.length} 个文件</span>`;
    }
    
    // Close button
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
    
    // Add to container
    container.appendChild(toast);
    activeToast = toast;
    
    // Trigger animation
    requestAnimationFrame(() => {
        toast.classList.add('toast-visible');
    });
    
    // Auto hide after 4 seconds
    toastTimeout = window.setTimeout(hideToast, 4000);
}

function parseReloadMessage(data: string): ReloadMessage | null {
    // Handle legacy format: plain "reload" string
    if (data === 'reload') {
        return { type: 'reload', files: [] };
    }
    
    // Try to parse JSON format
    try {
        const msg = JSON.parse(data) as ReloadMessage;
        if (msg.type === 'reload' && Array.isArray(msg.files)) {
            return msg;
        }
    } catch {
        // Not valid JSON, ignore
        return null;
    }
    
    return null;
}

export function setupLiveReloadStatus(
    evtSource: ReloadEventSource,
    liveDot: StatusDot | null,
    statusText: StatusText | null,
    refreshCurrentPage: () => Promise<void>,
) {
    evtSource.onopen = () => {
        setLiveState(liveDot, statusText);
    };

    evtSource.onmessage = event => {
        const msg = parseReloadMessage(event.data);
        if (!msg) {
            return;
        }

        // Show toast notification if files are provided
        if (msg.files.length > 0) {
            showFileChangeToast(msg.files);
        }

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
    };

    evtSource.onerror = () => {
        if (liveDot) {
            liveDot.style.backgroundColor = 'var(--status-warn)';
        }
        if (statusText) {
            statusText.innerText = 'Offline';
        }
        
        // Hide any active toast on connection error
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