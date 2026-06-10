import { describe, expect, test } from 'bun:test';
import { JSDOM } from 'jsdom';
import { PREVIEW_WIDTH_STORAGE_KEY } from '../preferences';
import { initPreviewResize } from './preview-resize';

describe('preview resize behavior', () => {
    test('dragging the left handle updates and persists preview width', () => {
        const dom = new JSDOM(`<!doctype html>
        <body>
            <aside id="preview-panel" class="preview-panel">
                <div class="preview-resize-handle" id="preview-resize-handle"></div>
            </aside>
        </body>`, {
            url: 'http://localhost/docs/page',
            pretendToBeVisual: true,
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
            const panel = document.getElementById('preview-panel') as HTMLElement;
            const handle = document.getElementById('preview-resize-handle') as HTMLElement;
            panel.getBoundingClientRect = () => ({
                width: 560,
                height: 800,
                top: 0,
                right: 1024,
                bottom: 800,
                left: 464,
                x: 464,
                y: 0,
                toJSON: () => ({}),
            });

            initPreviewResize();
            handle.dispatchEvent(new dom.window.MouseEvent('mousedown', {
                bubbles: true,
                button: 0,
                clientX: 464,
            }));
            expect(document.body.classList.contains('preview-is-resizing')).toBe(true);
            document.dispatchEvent(new dom.window.MouseEvent('mousemove', {
                bubbles: true,
                buttons: 1,
                clientX: 344,
            }));
            document.dispatchEvent(new dom.window.MouseEvent('mouseup', { bubbles: true }));

            expect(document.documentElement.style.getPropertyValue('--preview-width')).toBe('680px');
            expect(dom.window.localStorage.getItem(PREVIEW_WIDTH_STORAGE_KEY)).toBe('680');
            expect(handle.classList.contains('is-resizing')).toBe(false);
            expect(document.body.style.cursor).toBe('');
            expect(document.body.classList.contains('preview-is-resizing')).toBe(false);
        } finally {
            globalThis.window = previousWindow;
            globalThis.document = previousDocument;
            globalThis.HTMLElement = previousHTMLElement;
        }
    });

    test('window mouseup ends resize even when document misses mouseup', () => {
        const dom = new JSDOM(`<!doctype html>
        <body>
            <aside id="preview-panel" class="preview-panel">
                <div class="preview-resize-handle" id="preview-resize-handle"></div>
            </aside>
        </body>`, {
            url: 'http://localhost/docs/page',
            pretendToBeVisual: true,
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
            const panel = document.getElementById('preview-panel') as HTMLElement;
            const handle = document.getElementById('preview-resize-handle') as HTMLElement;
            panel.getBoundingClientRect = () => ({
                width: 560,
                height: 800,
                top: 0,
                right: 1024,
                bottom: 800,
                left: 464,
                x: 464,
                y: 0,
                toJSON: () => ({}),
            });

            initPreviewResize();
            handle.dispatchEvent(new dom.window.MouseEvent('mousedown', {
                bubbles: true,
                button: 0,
                clientX: 464,
            }));
            expect(document.body.classList.contains('preview-is-resizing')).toBe(true);
            dom.window.dispatchEvent(new dom.window.MouseEvent('mouseup', { bubbles: true }));

            expect(handle.classList.contains('is-resizing')).toBe(false);
            expect(document.body.style.cursor).toBe('');
            expect(document.body.classList.contains('preview-is-resizing')).toBe(false);
        } finally {
            globalThis.window = previousWindow;
            globalThis.document = previousDocument;
            globalThis.HTMLElement = previousHTMLElement;
        }
    });

    test('mousemove without pressed buttons cancels stale resize state', () => {
        const dom = new JSDOM(`<!doctype html>
        <body>
            <aside id="preview-panel" class="preview-panel">
                <div class="preview-resize-handle" id="preview-resize-handle"></div>
            </aside>
        </body>`, {
            url: 'http://localhost/docs/page',
            pretendToBeVisual: true,
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
            const panel = document.getElementById('preview-panel') as HTMLElement;
            const handle = document.getElementById('preview-resize-handle') as HTMLElement;
            panel.getBoundingClientRect = () => ({
                width: 560,
                height: 800,
                top: 0,
                right: 1024,
                bottom: 800,
                left: 464,
                x: 464,
                y: 0,
                toJSON: () => ({}),
            });

            initPreviewResize();
            handle.dispatchEvent(new dom.window.MouseEvent('mousedown', {
                bubbles: true,
                button: 0,
                clientX: 464,
            }));
            document.dispatchEvent(new dom.window.MouseEvent('mousemove', {
                bubbles: true,
                buttons: 0,
                clientX: 344,
            }));

            expect(handle.classList.contains('is-resizing')).toBe(false);
            expect(document.body.classList.contains('preview-is-resizing')).toBe(false);
        } finally {
            globalThis.window = previousWindow;
            globalThis.document = previousDocument;
            globalThis.HTMLElement = previousHTMLElement;
        }
    });
});
