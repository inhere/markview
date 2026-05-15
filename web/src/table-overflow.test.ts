import { describe, expect, test } from 'bun:test';
import { JSDOM } from 'jsdom';
import { enhanceTablesInContent, updateTableOverflow } from './table-overflow';

function withDOM(html: string, run: (document: Document) => void) {
    const dom = new JSDOM(html, {
        url: 'http://127.0.0.1/',
        pretendToBeVisual: true,
    });
    const previousDocument = globalThis.document;
    const previousWindow = globalThis.window;
    const previousHTMLElement = globalThis.HTMLElement;

    try {
        globalThis.document = dom.window.document;
        globalThis.window = dom.window as unknown as Window & typeof globalThis;
        globalThis.HTMLElement = dom.window.HTMLElement;
        run(dom.window.document);
    } finally {
        globalThis.document = previousDocument;
        globalThis.window = previousWindow;
        globalThis.HTMLElement = previousHTMLElement;
    }
}

describe('table overflow enhancement', () => {
    test('wraps markdown tables and expands overflowing table height on click', () => {
        withDOM(`<!DOCTYPE html><body>
            <article id="content">
                <table><tbody><tr><td>row</td></tr></tbody></table>
            </article>
        </body>`, document => {
            const content = document.getElementById('content') as HTMLElement;

            enhanceTablesInContent(content);

            const container = content.querySelector('.table-scroll-container') as HTMLElement;
            const body = container.querySelector('.table-scroll-body') as HTMLElement;
            const table = container.querySelector('table') as HTMLElement;
            const toggle = container.querySelector('.table-scroll-toggle') as HTMLButtonElement;
            expect(table.classList.contains('table-scroll-table')).toBe(true);
            expect(toggle.textContent).toBe('展开完整表格');

            Object.defineProperty(body, 'clientHeight', { value: 120, configurable: true });
            Object.defineProperty(body, 'scrollHeight', { value: 360, configurable: true });
            updateTableOverflow(container);

            expect(container.classList.contains('is-overflowing')).toBe(true);

            toggle.click();

            expect(container.classList.contains('is-expanded')).toBe(true);
            expect(container.classList.contains('is-overflowing')).toBe(false);
            expect(toggle.getAttribute('aria-expanded')).toBe('true');
            expect(toggle.textContent).toBe('收起表格');
        });
    });

    test('does not wrap an already enhanced table twice', () => {
        withDOM(`<!DOCTYPE html><body>
            <article id="content">
                <table><tbody><tr><td>row</td></tr></tbody></table>
            </article>
        </body>`, document => {
            const content = document.getElementById('content') as HTMLElement;

            enhanceTablesInContent(content);
            enhanceTablesInContent(content);

            expect(content.querySelectorAll('.table-scroll-container')).toHaveLength(1);
            expect(content.querySelectorAll('.table-scroll-body')).toHaveLength(1);
        });
    });
});
