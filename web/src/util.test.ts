import { describe, expect, test } from 'bun:test';
import { JSDOM } from 'jsdom';
import {
    buildContentBaseURL,
    buildHeadingAnchorId,
    escapeHtml,
    getContentScrollTop,
    isAlreadyAbsoluteURL,
    isInlineNavigablePath,
    scrollContentTo,
    scrollToHash,
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
        expect(isInlineNavigablePath('/example/config.json')).toBe(false);
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

    test('scrollToHash scrolls the content wrapper when present', () => {
        const dom = new JSDOM(`<!doctype html><body>
            <main class="content-wrapper">
                <div id="target">Target</div>
            </main>
        </body>`);
        const { document } = dom.window;
        Object.defineProperty(dom.window, 'innerWidth', {
            value: 1280,
            configurable: true,
        });
        const contentWrapper = document.querySelector('.content-wrapper') as HTMLElement;
        const target = document.getElementById('target') as HTMLElement;

        let scrolledTop = -1;
        Object.defineProperty(contentWrapper, 'scrollTop', {
            value: 40,
            writable: true,
            configurable: true,
        });
        contentWrapper.scrollTo = ((options: ScrollToOptions) => {
            scrolledTop = Number(options.top ?? 0);
        }) as typeof contentWrapper.scrollTo;
        contentWrapper.getBoundingClientRect = () => ({
            top: 120,
            left: 0,
            right: 0,
            bottom: 0,
            width: 0,
            height: 0,
            x: 0,
            y: 0,
            toJSON() {
                return {};
            },
        });
        target.getBoundingClientRect = () => ({
            top: 360,
            left: 0,
            right: 0,
            bottom: 0,
            width: 0,
            height: 0,
            x: 0,
            y: 0,
            toJSON() {
                return {};
            },
        });

        const previousWindow = globalThis.window;
        const previousDocument = globalThis.document;
        const previousHTMLElement = globalThis.HTMLElement;
        const previousCSS = globalThis.CSS;
        // @ts-expect-error test env override
        globalThis.window = dom.window;
        // @ts-expect-error test env override
        globalThis.document = document;
        // @ts-expect-error test env override
        globalThis.HTMLElement = dom.window.HTMLElement;
        // @ts-expect-error test env override
        globalThis.CSS = dom.window.CSS;

        try {
            scrollToHash('#target');
            expect(scrolledTop).toBe(280);
        } finally {
            globalThis.window = previousWindow;
            globalThis.document = previousDocument;
            globalThis.HTMLElement = previousHTMLElement;
            globalThis.CSS = previousCSS;
        }
    });

    test('getContentScrollTop and scrollContentTo use the content wrapper first', () => {
        const dom = new JSDOM(`<!doctype html><body><main class="content-wrapper"></main></body>`);
        const { document } = dom.window;
        Object.defineProperty(dom.window, 'innerWidth', {
            value: 1280,
            configurable: true,
        });
        const contentWrapper = document.querySelector('.content-wrapper') as HTMLElement;
        Object.defineProperty(contentWrapper, 'scrollTop', {
            value: 128,
            writable: true,
            configurable: true,
        });

        let scrolledTop = -1;
        contentWrapper.scrollTo = ((options: ScrollToOptions) => {
            scrolledTop = Number(options.top ?? 0);
        }) as typeof contentWrapper.scrollTo;

        const previousWindow = globalThis.window;
        const previousDocument = globalThis.document;
        const previousHTMLElement = globalThis.HTMLElement;
        // @ts-expect-error test env override
        globalThis.window = dom.window;
        // @ts-expect-error test env override
        globalThis.document = document;
        // @ts-expect-error test env override
        globalThis.HTMLElement = dom.window.HTMLElement;

        try {
            expect(getContentScrollTop()).toBe(128);
            scrollContentTo(256);
            expect(scrolledTop).toBe(256);
        } finally {
            globalThis.window = previousWindow;
            globalThis.document = previousDocument;
            globalThis.HTMLElement = previousHTMLElement;
        }
    });

    test('getContentScrollTop falls back to window scroll on mobile layout', () => {
        const dom = new JSDOM(`<!doctype html><body><main class="content-wrapper"></main></body>`);
        const { document } = dom.window;
        const contentWrapper = document.querySelector('.content-wrapper') as HTMLElement;
        Object.defineProperty(dom.window, 'innerWidth', {
            value: 800,
            configurable: true,
        });
        Object.defineProperty(contentWrapper, 'scrollTop', {
            value: 128,
            writable: true,
            configurable: true,
        });
        Object.defineProperty(dom.window, 'scrollY', {
            value: 64,
            configurable: true,
        });

        const previousWindow = globalThis.window;
        const previousDocument = globalThis.document;
        const previousHTMLElement = globalThis.HTMLElement;
        // @ts-expect-error test env override
        globalThis.window = dom.window;
        // @ts-expect-error test env override
        globalThis.document = document;
        // @ts-expect-error test env override
        globalThis.HTMLElement = dom.window.HTMLElement;

        try {
            expect(getContentScrollTop()).toBe(64);
        } finally {
            globalThis.window = previousWindow;
            globalThis.document = previousDocument;
            globalThis.HTMLElement = previousHTMLElement;
        }
    });
});
