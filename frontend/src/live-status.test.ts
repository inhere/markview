import { describe, expect, test } from 'bun:test';
import { setupLiveReloadStatus } from './live-status';

describe('live reload status', () => {
    test('restores live status after reconnecting from offline', () => {
        const source: {
            onopen: null | (() => void);
            onerror: null | (() => void);
            onmessage: null | ((event: { data: string }) => void);
        } = {
            onopen: null,
            onerror: null,
            onmessage: null,
        };

        const liveDot = createFakeDot();
        const statusText = { innerText: 'Live' };

        setupLiveReloadStatus(source, liveDot, statusText, async () => {});

        source.onerror?.();
        expect(statusText.innerText).toBe('Offline');
        expect(liveDot.style.backgroundColor).toBe('var(--status-warn)');

        source.onopen?.();
        expect(statusText.innerText).toBe('Live');
        expect(liveDot.style.backgroundColor).toBe('');
    });
});

function createFakeDot() {
    const classNames = new Set<string>();

    return {
        style: {
            backgroundColor: '',
        },
        classList: {
            add(name: string) {
                classNames.add(name);
            },
            remove(name: string) {
                classNames.delete(name);
            },
            contains(name: string) {
                return classNames.has(name);
            },
        },
    };
}
