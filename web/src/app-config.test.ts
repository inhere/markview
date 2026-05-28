import { describe, expect, test } from 'bun:test';
import { JSDOM } from 'jsdom';
import {
    DEFAULT_APP_CONFIG,
    normalizeAppConfig,
    readAppConfig,
} from './app-config';

function documentWithConfig(configJSON?: string): Document {
    const script = configJSON === undefined
        ? ''
        : `<script id="app-config-data" type="application/json">${configJSON}</script>`;
    return new JSDOM(`<!DOCTYPE html><body>${script}</body>`).window.document;
}

describe('app config', () => {
    test('returns defaults when config script is missing', () => {
        expect(readAppConfig(documentWithConfig())).toEqual(DEFAULT_APP_CONFIG);
    });

    test('normalizes injected preview extensions and layout', () => {
        const config = readAppConfig(documentWithConfig(JSON.stringify({
            previewExts: [' ini ', ' .JSON ', ' json ', '', 123, '.Env'],
            layout: 'toc-right',
        })));

        expect(config).toEqual({
            previewExts: ['.ini', '.json', '.env'],
            layout: 'toc-right',
        });
    });

    test('falls back to defaults when config JSON is invalid', () => {
        expect(readAppConfig(documentWithConfig('{bad json'))).toEqual(DEFAULT_APP_CONFIG);
    });

    test('falls back to compact for invalid layout', () => {
        const config = readAppConfig(documentWithConfig(JSON.stringify({
            previewExts: ['.ini'],
            layout: 'wide',
        })));

        expect(config.layout).toBe('compact');
    });

    test('falls back to default preview extensions when none are valid', () => {
        const config = readAppConfig(documentWithConfig(JSON.stringify({
            previewExts: ['', 42, null],
            layout: 'toc-middle',
        })));

        expect(config.previewExts).toEqual(DEFAULT_APP_CONFIG.previewExts);
        expect(config.layout).toBe('toc-middle');
    });

    test('uses compact layout when injected layout is missing', () => {
        expect(normalizeAppConfig({ previewExts: ['json'] })).toEqual({
            previewExts: ['.json'],
            layout: 'compact',
        });
    });
});
