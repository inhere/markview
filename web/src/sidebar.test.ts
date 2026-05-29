import { describe, expect, test } from 'bun:test';
import { JSDOM } from 'jsdom';
import { bindTOCScrollSpy, highlightTOC, setupSidebarCollapse } from './sidebar';

describe('sidebar scroll behavior', () => {
    test('highlightTOC reads scroll position from content wrapper', () => {
        const dom = new JSDOM(`<!doctype html><body>
            <main class="content-wrapper">
                <div id="content">
                    <h1 id="intro">Intro</h1>
                    <h2 id="details">Details</h2>
                </div>
            </main>
            <div class="toc-container">
                <a class="toc-link" href="#intro">Intro</a>
                <a class="toc-link" href="#details">Details</a>
            </div>
        </body>`, {
            url: 'http://localhost/docs/page',
        });
        const { document, history } = dom.window;
        Object.defineProperty(dom.window, 'innerWidth', {
            value: 1280,
            configurable: true,
        });
        const contentWrapper = document.querySelector('.content-wrapper') as HTMLElement;
        const intro = document.getElementById('intro') as HTMLElement;
        const details = document.getElementById('details') as HTMLElement;

        Object.defineProperty(contentWrapper, 'scrollTop', {
            value: 180,
            writable: true,
            configurable: true,
        });
        Object.defineProperty(intro, 'offsetTop', {
            value: 40,
            configurable: true,
        });
        Object.defineProperty(details, 'offsetTop', {
            value: 220,
            configurable: true,
        });

        const previousWindow = globalThis.window;
        const previousDocument = globalThis.document;
        const previousHTMLElement = globalThis.HTMLElement;
        const previousHistory = globalThis.history;
        // @ts-expect-error test env override
        globalThis.window = dom.window;
        // @ts-expect-error test env override
        globalThis.document = document;
        // @ts-expect-error test env override
        globalThis.HTMLElement = dom.window.HTMLElement;
        // @ts-expect-error test env override
        globalThis.history = history;

        try {
            highlightTOC();
            expect(document.querySelector('.toc-link[href="#details"]')?.classList.contains('active')).toBe(true);
            expect(document.querySelector('.toc-link[href="#intro"]')?.classList.contains('active')).toBe(false);
            expect(dom.window.location.hash).toBe('#details');
        } finally {
            globalThis.window = previousWindow;
            globalThis.document = previousDocument;
            globalThis.HTMLElement = previousHTMLElement;
            globalThis.history = previousHistory;
        }
    });

    test('bindTOCScrollSpy listens to both window and content wrapper', () => {
        const dom = new JSDOM(`<!doctype html><body>
            <main class="content-wrapper"></main>
        </body>`);
        const { document } = dom.window;
        const contentWrapper = document.querySelector('.content-wrapper') as HTMLElement;

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
            let calls = 0;
            bindTOCScrollSpy(() => {
                calls++;
            });

            contentWrapper.dispatchEvent(new dom.window.Event('scroll'));
            dom.window.dispatchEvent(new dom.window.Event('scroll'));

            expect(calls).toBe(2);
        } finally {
            globalThis.window = previousWindow;
            globalThis.document = previousDocument;
            globalThis.HTMLElement = previousHTMLElement;
        }
    });

    test('toc collapsed icon expands the files pane and restores body state', () => {
        const dom = new JSDOM(`<!doctype html><body class="sidebar-collapsed">
            <aside class="files-pane sidebar sidebar-collapsed">
                <button id="sidebar-collapse-btn"></button>
                <section id="files-panel"></section>
                <button class="sidebar-icon-btn" data-panel="toc"></button>
            </aside>
        </body>`, {
            url: 'http://localhost/docs/page',
        });
        const { document } = dom.window;
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
            setupSidebarCollapse();
            (document.querySelector('[data-panel="toc"]') as HTMLButtonElement).click();

            expect(document.body.classList.contains('sidebar-collapsed')).toBe(false);
            expect(document.querySelector('.files-pane')?.classList.contains('sidebar-collapsed')).toBe(false);
            expect(document.getElementById('sidebar-collapse-btn')?.getAttribute('aria-label')).toBe('Collapse sidebar');
        } finally {
            globalThis.window = previousWindow;
            globalThis.document = previousDocument;
            globalThis.HTMLElement = previousHTMLElement;
        }
    });
});
