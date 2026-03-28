export const LAYOUT_WIDTH_STORAGE_KEY = 'markview:layout-width';
export const FONT_SIZE_STORAGE_KEY = 'markview:font-size';

export const LAYOUT_WIDTH_OPTIONS = ['768px', '960px', '1200px', '100%'] as const;
export type LayoutWidth = (typeof LAYOUT_WIDTH_OPTIONS)[number];

export const DEFAULT_LAYOUT_WIDTH: LayoutWidth = '960px';
export const DEFAULT_FONT_SIZE = 16;
export const MIN_FONT_SIZE = 12;
export const MAX_FONT_SIZE = 24;

interface StorageReader {
    getItem(key: string): string | null;
}

interface StorageWriter {
    setItem(key: string, value: string): void;
}

export function normalizeLayoutWidth(value: string | null | undefined): LayoutWidth {
    if (value && LAYOUT_WIDTH_OPTIONS.includes(value as LayoutWidth)) {
        return value as LayoutWidth;
    }

    return DEFAULT_LAYOUT_WIDTH;
}

export function normalizeFontSize(value: string | null | undefined) {
    if (!value) {
        return DEFAULT_FONT_SIZE;
    }

    const parsed = Number.parseInt(value, 10);
    if (Number.isNaN(parsed)) {
        return DEFAULT_FONT_SIZE;
    }

    return Math.min(MAX_FONT_SIZE, Math.max(MIN_FONT_SIZE, parsed));
}

export function readStoredPreferences(storage: StorageReader = window.localStorage) {
    try {
        return {
            layoutWidth: normalizeLayoutWidth(storage.getItem(LAYOUT_WIDTH_STORAGE_KEY)),
            fontSize: normalizeFontSize(storage.getItem(FONT_SIZE_STORAGE_KEY)),
        };
    } catch {
        return {
            layoutWidth: DEFAULT_LAYOUT_WIDTH,
            fontSize: DEFAULT_FONT_SIZE,
        };
    }
}

export function persistLayoutWidth(value: LayoutWidth, storage: StorageWriter = window.localStorage) {
    try {
        storage.setItem(LAYOUT_WIDTH_STORAGE_KEY, value);
    } catch {
        // Ignore storage failures so reading continues to work in restrictive contexts.
    }
}

export function persistFontSize(value: number, storage: StorageWriter = window.localStorage) {
    try {
        storage.setItem(FONT_SIZE_STORAGE_KEY, String(normalizeFontSize(String(value))));
    } catch {
        // Ignore storage failures so reading continues to work in restrictive contexts.
    }
}
