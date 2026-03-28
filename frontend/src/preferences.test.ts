import { describe, expect, test } from 'bun:test';
import {
    DEFAULT_FONT_SIZE,
    DEFAULT_LAYOUT_WIDTH,
    MAX_FONT_SIZE,
    MIN_FONT_SIZE,
    normalizeFontSize,
    normalizeLayoutWidth,
    readStoredPreferences,
} from './preferences';

describe('frontend preferences', () => {
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
});
