import { describe, expect, test } from 'bun:test';
import {
    clearStoredLayoutMode,
    DEFAULT_LAYOUT_MODE,
    DEFAULT_FONT_SIZE,
    DEFAULT_LAYOUT_WIDTH,
    DEFAULT_PREVIEW_WIDTH,
    LAYOUT_MODE_STORAGE_KEY,
    MAX_FONT_SIZE,
    MAX_PREVIEW_WIDTH,
    MIN_FONT_SIZE,
    MIN_PREVIEW_WIDTH,
    normalizePreviewWidth,
    normalizeFontSize,
    normalizeLayoutMode,
    normalizeLayoutWidth,
    persistPreviewWidth,
    persistLayoutMode,
    PREVIEW_WIDTH_STORAGE_KEY,
    readStoredLayoutMode,
    readPreviewPreferences,
    readStoredPreferences,
    resolveLayoutMode,
} from './preferences';

describe('web preferences', () => {
    test('normalizeLayoutWidth accepts supported widths and falls back otherwise', () => {
        expect(normalizeLayoutWidth('1200px')).toBe('1200px');
        expect(normalizeLayoutWidth('bad-width')).toBe(DEFAULT_LAYOUT_WIDTH);
    });

    test('normalizeFontSize clamps invalid values back to defaults or bounds', () => {
        expect(normalizeFontSize('20')).toBe(20);
        expect(normalizeFontSize('999')).toBe(MAX_FONT_SIZE);
        expect(normalizeFontSize('1')).toBe(MIN_FONT_SIZE);
        expect(normalizeFontSize('oops')).toBe(DEFAULT_FONT_SIZE);
    });

    test('normalizePreviewWidth accepts stored pixels and clamps out of range values', () => {
        expect(normalizePreviewWidth('720')).toBe(720);
        expect(normalizePreviewWidth('99999')).toBe(MAX_PREVIEW_WIDTH);
        expect(normalizePreviewWidth('1')).toBe(MIN_PREVIEW_WIDTH);
        expect(normalizePreviewWidth('oops')).toBeNull();
        expect(DEFAULT_PREVIEW_WIDTH).toBe(560);
    });

    test('readStoredPreferences restores stored values', () => {
        const storage = new Map<string, string>([
            ['markview:layout-width', '100%'],
            ['markview:font-size', '18'],
        ]);

        const preferences = readStoredPreferences({
            getItem(key: string) {
                return storage.get(key) ?? null;
            },
        });

        expect(preferences.layoutWidth).toBe('100%');
        expect(preferences.fontSize).toBe(18);
    });

    test('normalizeLayoutMode accepts supported modes and falls back otherwise', () => {
        expect(normalizeLayoutMode('compact')).toBe('compact');
        expect(normalizeLayoutMode('toc-middle')).toBe('toc-middle');
        expect(normalizeLayoutMode('toc-right')).toBe('toc-right');
        expect(normalizeLayoutMode('wide')).toBe(DEFAULT_LAYOUT_MODE);
    });

    test('readStoredPreferences includes layout mode', () => {
        const storage = new Map<string, string>([
            [LAYOUT_MODE_STORAGE_KEY, 'toc-right'],
        ]);

        const preferences = readStoredPreferences({
            getItem(key: string) {
                return storage.get(key) ?? null;
            },
        });

        expect(preferences.layoutMode).toBe('toc-right');
    });

    test('readStoredLayoutMode distinguishes missing preference from compact', () => {
        expect(readStoredLayoutMode({
            getItem() {
                return null;
            },
        })).toBeNull();

        expect(readStoredLayoutMode({
            getItem() {
                return 'compact';
            },
        })).toBe('compact');

        expect(readStoredLayoutMode({
            getItem() {
                return 'wide';
            },
        })).toBeNull();
    });

    test('resolveLayoutMode prefers stored mode before configured layout', () => {
        expect(resolveLayoutMode('toc-middle', 'toc-right')).toBe('toc-middle');
        expect(resolveLayoutMode(null, 'toc-right')).toBe('toc-right');
    });

    test('clearStoredLayoutMode removes local layout override', () => {
        const storage = new Map<string, string>([
            [LAYOUT_MODE_STORAGE_KEY, 'toc-right'],
        ]);

        clearStoredLayoutMode({
            removeItem(key: string) {
                storage.delete(key);
            },
        });

        expect(storage.has(LAYOUT_MODE_STORAGE_KEY)).toBe(false);
    });

    test('invalid stored layout does not override configured layout', () => {
        const stored = readStoredLayoutMode({
            getItem() {
                return 'wide';
            },
        });

        expect(resolveLayoutMode(stored, 'toc-right')).toBe('toc-right');
    });

    test('persistLayoutMode stores normalized supported value', () => {
        const storage = new Map<string, string>();

        persistLayoutMode('toc-middle', {
            setItem(key: string, value: string) {
                storage.set(key, value);
            },
        });

        expect(storage.get(LAYOUT_MODE_STORAGE_KEY)).toBe('toc-middle');
    });

    test('readPreviewPreferences and persistPreviewWidth use preview width storage', () => {
        const storage = new Map<string, string>([
            [PREVIEW_WIDTH_STORAGE_KEY, '700'],
        ]);

        expect(readPreviewPreferences({
            getItem(key: string) {
                return storage.get(key) ?? null;
            },
        }).previewWidth).toBe(700);

        persistPreviewWidth(900, {
            setItem(key: string, value: string) {
                storage.set(key, value);
            },
        });

        expect(storage.get(PREVIEW_WIDTH_STORAGE_KEY)).toBe('900');
    });
});
