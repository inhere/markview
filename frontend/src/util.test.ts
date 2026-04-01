import { describe, expect, test } from 'bun:test';
import {
    buildContentBaseURL,
    buildHeadingAnchorId,
    escapeHtml,
    isAlreadyAbsoluteURL,
    isInlineNavigablePath,
    toDisplaySlug,
} from './util';

describe('web util', () => {
    test('escapeHtml escapes reserved characters', () => {
        expect(escapeHtml(`<tag attr="x">&'`)).toBe('&lt;tag attr=&quot;x&quot;&gt;&amp;&#39;');
    });

    test('buildContentBaseURL uses markdown directory as base', () => {
        const base = buildContentBaseURL('example/basics.md', 'http://127.0.0.1:6125');
        expect(base.toString()).toBe('http://127.0.0.1:6125/example/');
    });

    test('buildContentBaseURL falls back to root for top-level markdown', () => {
        const base = buildContentBaseURL('README.md', 'http://127.0.0.1:6125');
        expect(base.toString()).toBe('http://127.0.0.1:6125/');
    });

    test('isAlreadyAbsoluteURL recognizes absolute and root-based paths', () => {
        expect(isAlreadyAbsoluteURL('https://example.com')).toBe(true);
        expect(isAlreadyAbsoluteURL('/example/basics.md')).toBe(true);
        expect(isAlreadyAbsoluteURL('#section')).toBe(true);
        expect(isAlreadyAbsoluteURL('./basics.md')).toBe(false);
    });

    test('isInlineNavigablePath accepts markdown pages and directory entries', () => {
        expect(isInlineNavigablePath('/')).toBe(true);
        expect(isInlineNavigablePath('/example')).toBe(true);
        expect(isInlineNavigablePath('/example/basics.md')).toBe(true);
        expect(isInlineNavigablePath('/assets/logo.png')).toBe(false);
    });

    test('buildHeadingAnchorId keeps readable slug for ascii headings', () => {
        expect(buildHeadingAnchorId('1.2 Product Positioning')).toBe('1-2-product-positioning');
        expect(toDisplaySlug('API Design')).toBe('api-design');
    });

    test('buildHeadingAnchorId preserves numeric separators for mixed chinese headings', () => {
        const anchorId = buildHeadingAnchorId('1.2 产品定位');
        expect(anchorId).toMatch(/^1-2-[a-z0-9]{6}$/);
    });

    test('buildHeadingAnchorId uses indexed prefix for chinese-only headings', () => {
        const anchorId = buildHeadingAnchorId('产品定位', 3);
        expect(anchorId).toMatch(/^i3-[a-z0-9]{6}$/);
    });
});
