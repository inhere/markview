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
        if (event.data !== 'reload') {
            return;
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
