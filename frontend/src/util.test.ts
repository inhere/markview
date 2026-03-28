import { describe, expect, test } from 'bun:test';
import {
    buildContentBaseURL,
    escapeHtml,
    isAlreadyAbsoluteURL,
    isInlineNavigablePath,
} from './util';

describe('frontend util', () => {
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
});
