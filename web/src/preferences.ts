export const LAYOUT_WIDTH_STORAGE_KEY = 'markview:layout-width';
export const FONT_SIZE_STORAGE_KEY = 'markview:font-size';
export const COLOR_SCHEME_STORAGE_KEY = 'markview:color-scheme';
export const THEME_STORAGE_KEY = 'markview:theme';
export const TOOLBAR_COLLAPSED_STORAGE_KEY = 'markview:toolbar-collapsed';

export const SIDEBAR_COLLAPSED_STORAGE_KEY = 'markview:sidebar-collapsed';
export const SIDEBAR_WIDTH_STORAGE_KEY = 'markview:sidebar-width';
export const FILES_COLLAPSED_STORAGE_KEY = 'markview:files-collapsed';

export const DEFAULT_SIDEBAR_WIDTH = 280;
export const MIN_SIDEBAR_WIDTH = 200;
export const MAX_SIDEBAR_WIDTH = 800;

export const LAYOUT_WIDTH_OPTIONS = ['768px', '960px', '1200px', '100%'] as const;
export type LayoutWidth = (typeof LAYOUT_WIDTH_OPTIONS)[number];

export const DEFAULT_LAYOUT_WIDTH: LayoutWidth = '960px';
export const DEFAULT_FONT_SIZE = 16;
export const MIN_FONT_SIZE = 12;
export const MAX_FONT_SIZE = 24;

export const COLOR_SCHEME_OPTIONS = ['system', 'light', 'dark'] as const;
export type ColorScheme = (typeof COLOR_SCHEME_OPTIONS)[number];
export const DEFAULT_COLOR_SCHEME: ColorScheme = 'system';

export const THEME_OPTIONS = [
    'default',
    'github',
    'one-dark',
    'dracula',
    'nord',
] as const;
export type ThemePreset = (typeof THEME_OPTIONS)[number];
export const DEFAULT_THEME: ThemePreset = 'default';

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

export function normalizeColorScheme(value: string | null | undefined): ColorScheme {
    if (value && COLOR_SCHEME_OPTIONS.includes(value as ColorScheme)) {
        return value as ColorScheme;
    }
    return DEFAULT_COLOR_SCHEME;
}

export function normalizeTheme(value: string | null | undefined): ThemePreset {
    if (value && THEME_OPTIONS.includes(value as ThemePreset)) {
        return value as ThemePreset;
    }
    return DEFAULT_THEME;
}

export function normalizeToolbarCollapsed(value: string | null | undefined): boolean {
    return value !== 'false';
}

export function readStoredPreferences(storage: StorageReader = window.localStorage) {
    try {
        return {
            layoutWidth: normalizeLayoutWidth(storage.getItem(LAYOUT_WIDTH_STORAGE_KEY)),
            fontSize: normalizeFontSize(storage.getItem(FONT_SIZE_STORAGE_KEY)),
            colorScheme: normalizeColorScheme(storage.getItem(COLOR_SCHEME_STORAGE_KEY)),
            theme: normalizeTheme(storage.getItem(THEME_STORAGE_KEY)),
            toolbarCollapsed: normalizeToolbarCollapsed(storage.getItem(TOOLBAR_COLLAPSED_STORAGE_KEY)),
        };
    } catch {
        return {
            layoutWidth: DEFAULT_LAYOUT_WIDTH,
            fontSize: DEFAULT_FONT_SIZE,
            colorScheme: DEFAULT_COLOR_SCHEME,
            theme: DEFAULT_THEME,
            toolbarCollapsed: true,
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

export function persistColorScheme(value: ColorScheme, storage: StorageWriter = window.localStorage) {
    try {
        storage.setItem(COLOR_SCHEME_STORAGE_KEY, value);
    } catch {}
}

export function persistTheme(value: ThemePreset, storage: StorageWriter = window.localStorage) {
    try {
        storage.setItem(THEME_STORAGE_KEY, value);
    } catch {}
}

export function persistToolbarCollapsed(value: boolean, storage: StorageWriter = window.localStorage) {
    try {
        storage.setItem(TOOLBAR_COLLAPSED_STORAGE_KEY, String(value));
    } catch {}
}

export function normalizeSidebarWidth(value: string | null | undefined): number {
    if (!value) return DEFAULT_SIDEBAR_WIDTH;
    const parsed = Number.parseInt(value, 10);
    if (Number.isNaN(parsed)) return DEFAULT_SIDEBAR_WIDTH;
    return Math.min(MAX_SIDEBAR_WIDTH, Math.max(MIN_SIDEBAR_WIDTH, parsed));
}

export function normalizeSidebarCollapsed(value: string | null | undefined): boolean {
    return value === 'true';
}

export function readSidebarPreferences(storage: StorageReader = window.localStorage) {
    try {
        return {
            sidebarWidth: normalizeSidebarWidth(storage.getItem(SIDEBAR_WIDTH_STORAGE_KEY)),
            sidebarCollapsed: normalizeSidebarCollapsed(storage.getItem(SIDEBAR_COLLAPSED_STORAGE_KEY)),
            filesCollapsed: normalizeSidebarCollapsed(storage.getItem(FILES_COLLAPSED_STORAGE_KEY)),
        };
    } catch {
        return {
            sidebarWidth: DEFAULT_SIDEBAR_WIDTH,
            sidebarCollapsed: false,
            filesCollapsed: false,
        };
    }
}

export function persistSidebarWidth(value: number, storage: StorageWriter = window.localStorage) {
    try {
        storage.setItem(SIDEBAR_WIDTH_STORAGE_KEY, String(value));
    } catch {}
}

export function persistSidebarCollapsed(value: boolean, storage: StorageWriter = window.localStorage) {
    try {
        storage.setItem(SIDEBAR_COLLAPSED_STORAGE_KEY, String(value));
    } catch {}
}

export function persistFilesCollapsed(value: boolean, storage: StorageWriter = window.localStorage) {
    try {
        storage.setItem(FILES_COLLAPSED_STORAGE_KEY, String(value));
    } catch {}
}
