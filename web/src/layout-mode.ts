import type { AppLayout } from './app-config';
import {
    clearStoredLayoutMode,
    LAYOUT_MODE_STORAGE_KEY,
    persistLayoutMode,
} from './preferences';

interface StorageReaderWriter {
    getItem(key: string): string | null;
    setItem(key: string, value: string): void;
    removeItem(key: string): void;
}

interface LayoutControlOptions {
    documentRef?: Document;
    storage?: StorageReaderWriter;
    configuredLayout: AppLayout;
    initialLayout: AppLayout;
}

function readLayoutMode(value: string | undefined): AppLayout | null {
    if (value === 'compact' || value === 'toc-middle' || value === 'toc-right') {
        return value;
    }
    return null;
}

function createDocumentEvent(documentRef: Document, type: string, detail: Record<string, unknown>) {
    const CustomEventCtor = documentRef.defaultView?.CustomEvent ?? CustomEvent;
    return new CustomEventCtor(type, { detail });
}

export function applyLayoutMode(mode: AppLayout, documentRef: Document = document) {
    documentRef.documentElement.dataset.layout = mode;
    documentRef.dispatchEvent(createDocumentEvent(documentRef, 'markview:layout-mode-changed', { mode }));
}

export function hasStoredLayoutMode(storage: Pick<Storage, 'getItem'> = window.localStorage): boolean {
    try {
        return storage.getItem(LAYOUT_MODE_STORAGE_KEY) !== null;
    } catch {
        return false;
    }
}

export function syncLayoutControls(documentRef: Document, mode: AppLayout, hasOverride: boolean) {
    documentRef.querySelectorAll('[data-layout-mode]').forEach(button => {
        const selected = readLayoutMode((button as HTMLElement).dataset.layoutMode) === mode;
        button.classList.toggle('active', selected);
        button.setAttribute('aria-pressed', String(selected));
    });

    const defaultButton = documentRef.getElementById('layout-default') as HTMLButtonElement | null;
    if (!defaultButton) {
        return;
    }

    defaultButton.disabled = !hasOverride;
    defaultButton.classList.toggle('active', !hasOverride);
}

export function setupLayoutControls({
    documentRef = document,
    storage = window.localStorage,
    configuredLayout,
    initialLayout,
}: LayoutControlOptions) {
    syncLayoutControls(documentRef, initialLayout, hasStoredLayoutMode(storage));

    documentRef.querySelectorAll('[data-layout-mode]').forEach(button => {
        button.addEventListener('click', () => {
            const mode = readLayoutMode((button as HTMLElement).dataset.layoutMode);
            if (!mode) {
                return;
            }

            applyLayoutMode(mode, documentRef);
            persistLayoutMode(mode, storage);
            syncLayoutControls(documentRef, mode, true);
        });
    });

    documentRef.getElementById('layout-default')?.addEventListener('click', () => {
        clearStoredLayoutMode(storage);
        applyLayoutMode(configuredLayout, documentRef);
        syncLayoutControls(documentRef, configuredLayout, false);
    });
}
