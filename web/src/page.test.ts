import { describe, expect, test } from 'bun:test';
import { JSDOM } from 'jsdom';
import { applyPageSnapshot, parsePageSnapshot, type PageMountSelectors } from './page';

const selectors: PageMountSelectors = {
    contentSelector: '#content',
    fileMetaSelector: '#file-meta',
    fileTreeDataId: 'file-tree-data',
    currentFilePathDataId: 'current-file-path-data',
};

describe('page snapshot navigation', () => {
    test('updates view raw link when applying an inline navigation snapshot', () => {
        const dom = new JSDOM(`<!DOCTYPE html><body>
            <div class="paper-wrapper">
                <a href="/old.md?q=raw" class="view-raw-btn">查看Markdown</a>
                <article id="content">Old content</article>
            </div>
            <div id="file-meta">Old meta</div>
            <script id="file-tree-data" type="application/json">[]</script>
            <script id="current-file-path-data" type="application/json">"old.md"</script>
        </body>`);

        const nextHTML = `<!DOCTYPE html><body>
            <div class="paper-wrapper">
                <a href="/next.md?q=raw" class="view-raw-btn">查看Markdown</a>
                <article id="content"><h1>Next</h1></article>
            </div>
            <div id="file-meta">Next meta</div>
            <script id="current-file-path-data" type="application/json">"next.md"</script>
        </body>`;

        const previousDocument = globalThis.document;
        const previousDOMParser = globalThis.DOMParser;
        const previousHTMLElement = globalThis.HTMLElement;
        try {
            globalThis.document = dom.window.document;
            globalThis.DOMParser = dom.window.DOMParser;
            globalThis.HTMLElement = dom.window.HTMLElement;

            const snapshot = parsePageSnapshot(nextHTML, selectors);
            applyPageSnapshot(snapshot, selectors);

            const rawLink = dom.window.document.querySelector('.view-raw-btn');
            expect(rawLink?.getAttribute('href')).toBe('/next.md?q=raw');
        } finally {
            globalThis.document = previousDocument;
            globalThis.DOMParser = previousDOMParser;
            globalThis.HTMLElement = previousHTMLElement;
        }
    });
});
