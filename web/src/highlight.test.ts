import { describe, expect, test } from 'bun:test';
import { JSDOM } from 'jsdom';
import { safeHighlightElement } from './highlight';

describe('highlight fallback behavior', () => {
    test('safeHighlightElement treats unknown language as plaintext without throwing', () => {
        const dom = new JSDOM(`<!doctype html><body>
            <pre><code class="language-madeup">hello()</code></pre>
        </body>`);
        const block = dom.window.document.querySelector('code') as HTMLElement;

        expect(() => safeHighlightElement(block)).not.toThrow();
        expect(block.dataset.highlighted).toBe('yes');
        expect(block.classList.contains('language-plaintext')).toBe(true);
    });
});
