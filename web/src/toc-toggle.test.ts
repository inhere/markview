import { describe, expect, test } from 'bun:test';
import { JSDOM } from 'jsdom';
import { setupTocToggle } from './toc-toggle';

function createDom(layout = 'toc-right') {
    return new JSDOM(`<!doctype html>
        <html data-layout="${layout}">
            <body>
                <aside id="toc-panel" class="toc-pane">
                    <button class="toc-section-toggle" aria-controls="toc-panel" aria-expanded="true"></button>
                </aside>
            </body>
        </html>`, {
        url: 'http://localhost/',
    });
}

describe('toc floating toggle', () => {
    test('toc-right toggle opens and closes floating toc', () => {
        const dom = createDom();

        setupTocToggle({ documentRef: dom.window.document });

        const button = dom.window.document.querySelector('.toc-section-toggle') as HTMLButtonElement;

        expect(dom.window.document.body.classList.contains('toc-floating-open')).toBe(true);
        expect(button.getAttribute('aria-expanded')).toBe('true');

        button.click();
        expect(dom.window.document.body.classList.contains('toc-floating-open')).toBe(false);
        expect(button.getAttribute('aria-expanded')).toBe('false');

        button.click();
        expect(dom.window.document.body.classList.contains('toc-floating-open')).toBe(true);
        expect(button.getAttribute('aria-expanded')).toBe('true');
    });

    test('preview active hides floating toc by default but manual toggle can reopen it', () => {
        const dom = createDom();

        setupTocToggle({ documentRef: dom.window.document });

        dom.window.document.body.classList.add('preview-active');
        dom.window.document.dispatchEvent(new dom.window.CustomEvent('markview:preview-state-changed'));

        const button = dom.window.document.querySelector('.toc-section-toggle') as HTMLButtonElement;

        expect(dom.window.document.body.classList.contains('toc-floating-open')).toBe(false);
        expect(button.getAttribute('aria-expanded')).toBe('false');

        button.click();
        expect(dom.window.document.body.classList.contains('toc-floating-open')).toBe(true);
        expect(button.getAttribute('aria-expanded')).toBe('true');
    });

    test('non toc-right layout can toggle toc content', () => {
        const dom = createDom('compact');

        setupTocToggle({ documentRef: dom.window.document });

        const button = dom.window.document.querySelector('.toc-section-toggle') as HTMLButtonElement;
        expect(dom.window.document.body.classList.contains('toc-floating-open')).toBe(true);
        expect(button.getAttribute('aria-expanded')).toBe('true');

        button.click();
        expect(dom.window.document.body.classList.contains('toc-floating-open')).toBe(false);
        expect(button.getAttribute('aria-expanded')).toBe('false');

        button.click();
        expect(dom.window.document.body.classList.contains('toc-floating-open')).toBe(true);
        expect(button.getAttribute('aria-expanded')).toBe('true');
    });

    test('layout change events keep toc content available outside preview', () => {
        const dom = createDom();

        setupTocToggle({ documentRef: dom.window.document });

        const button = dom.window.document.querySelector('.toc-section-toggle') as HTMLButtonElement;
        expect(dom.window.document.body.classList.contains('toc-floating-open')).toBe(true);

        dom.window.document.documentElement.dataset.layout = 'toc-middle';
        dom.window.document.dispatchEvent(new dom.window.CustomEvent('markview:layout-mode-changed'));
        expect(dom.window.document.body.classList.contains('toc-floating-open')).toBe(true);
        expect(button.getAttribute('aria-expanded')).toBe('true');

        dom.window.document.documentElement.dataset.layout = 'toc-right';
        dom.window.document.dispatchEvent(new dom.window.CustomEvent('markview:layout-mode-changed'));
        expect(dom.window.document.body.classList.contains('toc-floating-open')).toBe(true);
        expect(button.getAttribute('aria-expanded')).toBe('true');
    });

    test('repeated preview events keep a manually reopened toc visible', () => {
        const dom = createDom();

        setupTocToggle({ documentRef: dom.window.document });

        const button = dom.window.document.querySelector('.toc-section-toggle') as HTMLButtonElement;
        dom.window.document.body.classList.add('preview-active');
        dom.window.document.dispatchEvent(new dom.window.CustomEvent('markview:preview-state-changed'));
        expect(dom.window.document.body.classList.contains('toc-floating-open')).toBe(false);

        button.click();
        expect(dom.window.document.body.classList.contains('toc-floating-open')).toBe(true);

        dom.window.document.dispatchEvent(new dom.window.CustomEvent('markview:preview-state-changed'));
        expect(dom.window.document.body.classList.contains('toc-floating-open')).toBe(true);
        expect(button.getAttribute('aria-expanded')).toBe('true');
    });

    test('same-layout events preserve a manually selected toc state', () => {
        const dom = createDom();

        setupTocToggle({ documentRef: dom.window.document });

        const button = dom.window.document.querySelector('.toc-section-toggle') as HTMLButtonElement;
        button.click();
        expect(dom.window.document.body.classList.contains('toc-floating-open')).toBe(false);

        dom.window.document.dispatchEvent(new dom.window.CustomEvent('markview:layout-mode-changed'));
        expect(dom.window.document.body.classList.contains('toc-floating-open')).toBe(false);
        expect(button.getAttribute('aria-expanded')).toBe('false');
    });
});
