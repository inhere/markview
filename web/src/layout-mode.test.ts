import { describe, expect, test } from 'bun:test';
import { JSDOM } from 'jsdom';
import { LAYOUT_MODE_STORAGE_KEY } from './preferences';
import {
    applyLayoutMode,
    setupLayoutControls,
    syncLayoutControls,
} from './layout-mode';

function createLayoutDom() {
    return new JSDOM(`<!doctype html>
        <html>
            <body>
                <button data-layout-mode="compact"></button>
                <button data-layout-mode="toc-middle"></button>
                <button data-layout-mode="toc-right"></button>
                <button data-layout-mode="wide"></button>
                <button id="layout-default"></button>
            </body>
        </html>`);
}

function createStorage(initial: Record<string, string> = {}) {
    const entries = new Map<string, string>(Object.entries(initial));

    return {
        getItem(key: string) {
            return entries.get(key) ?? null;
        },
        setItem(key: string, value: string) {
            entries.set(key, value);
        },
        removeItem(key: string) {
            entries.delete(key);
        },
    } as Storage;
}

describe('layout mode controls', () => {
    test('applyLayoutMode sets the document layout dataset', () => {
        const dom = createLayoutDom();

        applyLayoutMode('toc-middle', dom.window.document);

        expect(dom.window.document.documentElement.dataset.layout).toBe('toc-middle');
    });

    test('syncLayoutControls marks selected layout button and enables default when local override exists', () => {
        const dom = createLayoutDom();

        syncLayoutControls(dom.window.document, 'toc-right', true);

        const selected = dom.window.document.querySelector('[data-layout-mode="toc-right"]');
        const compact = dom.window.document.querySelector('[data-layout-mode="compact"]');
        const defaultButton = dom.window.document.getElementById('layout-default') as HTMLButtonElement;

        expect(selected?.classList.contains('active')).toBe(true);
        expect(selected?.getAttribute('aria-pressed')).toBe('true');
        expect(compact?.classList.contains('active')).toBe(false);
        expect(compact?.getAttribute('aria-pressed')).toBe('false');
        expect(defaultButton.disabled).toBe(false);
        expect(defaultButton.classList.contains('active')).toBe(false);
    });

    test('syncLayoutControls disables default and marks it active when using project default', () => {
        const dom = createLayoutDom();

        syncLayoutControls(dom.window.document, 'toc-right', false);

        const defaultButton = dom.window.document.getElementById('layout-default') as HTMLButtonElement;

        expect(defaultButton.disabled).toBe(true);
        expect(defaultButton.classList.contains('active')).toBe(true);
    });

    test('setupLayoutControls persists selected layout and clears override on default', () => {
        const dom = createLayoutDom();
        const storage = createStorage();

        setupLayoutControls({
            documentRef: dom.window.document,
            storage,
            configuredLayout: 'toc-right',
            initialLayout: 'compact',
        });

        const middleButton = dom.window.document.querySelector('[data-layout-mode="toc-middle"]') as HTMLButtonElement;
        const defaultButton = dom.window.document.getElementById('layout-default') as HTMLButtonElement;

        middleButton.click();

        expect(dom.window.document.documentElement.dataset.layout).toBe('toc-middle');
        expect(storage.getItem(LAYOUT_MODE_STORAGE_KEY)).toBe('toc-middle');
        expect(middleButton.classList.contains('active')).toBe(true);
        expect(defaultButton.disabled).toBe(false);

        defaultButton.click();

        expect(dom.window.document.documentElement.dataset.layout).toBe('toc-right');
        expect(storage.getItem(LAYOUT_MODE_STORAGE_KEY)).toBeNull();
        expect(defaultButton.disabled).toBe(true);
        expect(defaultButton.classList.contains('active')).toBe(true);
    });

    test('setupLayoutControls ignores invalid layout data attributes', () => {
        const dom = createLayoutDom();
        const storage = createStorage();

        setupLayoutControls({
            documentRef: dom.window.document,
            storage,
            configuredLayout: 'toc-right',
            initialLayout: 'compact',
        });

        const invalidButton = dom.window.document.querySelector('[data-layout-mode="wide"]') as HTMLButtonElement;
        invalidButton.click();

        expect(dom.window.document.documentElement.dataset.layout).toBeUndefined();
        expect(storage.getItem(LAYOUT_MODE_STORAGE_KEY)).toBeNull();
    });
});
