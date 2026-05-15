import { describe, expect, test } from 'bun:test';
import { JSDOM } from 'jsdom';
import {
    buildHighlightedFilePreview,
    detectPreviewFileLanguage,
    enhanceLinksInContent,
    isPreviewableContentPath,
} from './link-preview';

function withDOM(html: string, run: (document: Document) => void) {
    const dom = new JSDOM(html, {
        url: 'http://127.0.0.1/docs/readme.md',
        pretendToBeVisual: true,
    });
    const previousDocument = globalThis.document;
    const previousWindow = globalThis.window;
    const previousHTMLElement = globalThis.HTMLElement;
    const previousHTMLAnchorElement = globalThis.HTMLAnchorElement;

    try {
        globalThis.document = dom.window.document;
        globalThis.window = dom.window as unknown as Window & typeof globalThis;
        globalThis.HTMLElement = dom.window.HTMLElement;
        globalThis.HTMLAnchorElement = dom.window.HTMLAnchorElement;
        run(dom.window.document);
    } finally {
        globalThis.document = previousDocument;
        globalThis.window = previousWindow;
        globalThis.HTMLElement = previousHTMLElement;
        globalThis.HTMLAnchorElement = previousHTMLAnchorElement;
    }
}

describe('link preview content files', () => {
    test('recognizes common content files for preview panel', () => {
        expect(isPreviewableContentPath('/api/schema.json')).toBe(true);
        expect(isPreviewableContentPath('/api/schema.json?env=dev')).toBe(true);
        expect(isPreviewableContentPath('/api/events.jsonl')).toBe(true);
        expect(isPreviewableContentPath('/deploy/config.yaml')).toBe(true);
        expect(isPreviewableContentPath('/deploy/config.yml')).toBe(true);
        expect(isPreviewableContentPath('/config/app.toml')).toBe(true);
        expect(isPreviewableContentPath('/assets/logo.png')).toBe(false);
    });

    test('maps preview file extension to highlight language', () => {
        expect(detectPreviewFileLanguage('/api/schema.json')).toBe('json');
        expect(detectPreviewFileLanguage('/api/events.jsonl')).toBe('json');
        expect(detectPreviewFileLanguage('/deploy/config.yaml')).toBe('yaml');
        expect(detectPreviewFileLanguage('/deploy/config.yml')).toBe('yaml');
        expect(detectPreviewFileLanguage('/config/app.toml')).toBe('toml');
        expect(detectPreviewFileLanguage('/notes/readme.md')).toBeNull();
    });

    test('renders escaped highlighted file preview markup', () => {
        const html = buildHighlightedFilePreview('{"name":"<demo>"}', 'json');

        expect(html).toContain('<pre class="preview-file-code"><code class="language-json">');
        expect(html).toContain('&lt;demo&gt;');
    });

    test('adds preview button for json links', () => {
        withDOM(`<!DOCTYPE html><body>
            <article id="content"><a href="/config/app.json">config</a></article>
        </body>`, document => {
            const content = document.getElementById('content') as HTMLElement;

            enhanceLinksInContent(content);

            expect(content.querySelector('.link-preview-wrapper')).not.toBeNull();
            expect(content.querySelector('.link-preview-btn')).not.toBeNull();
        });
    });
});
