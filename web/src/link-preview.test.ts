import { describe, expect, test } from 'bun:test';
import { JSDOM } from 'jsdom';
import {
    buildHTMLFilePreview,
    buildHighlightedFilePreview,
    configureLinkPreview,
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
        expect(isPreviewableContentPath('/pages/demo.html')).toBe(true);
        expect(isPreviewableContentPath('/assets/logo.png')).toBe(false);
    });

    test('uses provided preview extensions and ignores markdown raw content previews', () => {
        expect(isPreviewableContentPath('/config/app.ini', ['.ini'])).toBe(true);
        expect(isPreviewableContentPath('/config/app.ini', ['.json'])).toBe(false);
        expect(isPreviewableContentPath('/notes/readme.md', ['.md'])).toBe(false);
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

    test('renders html content previews as iframe markup', () => {
        const html = buildHTMLFilePreview('http://127.0.0.1/pages/demo.html?theme=dark');

        expect(html).toContain('<iframe');
        expect(html).toContain('class="preview-html-frame"');
        expect(html).toContain('src="http://127.0.0.1/pages/demo.html?theme=dark"');
        expect(html).not.toContain('<pre class="preview-file-code">');
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

    test('adds preview button for html links by default', () => {
        withDOM(`<!DOCTYPE html><body>
            <article id="content"><a href="/pages/demo.html">demo</a></article>
        </body>`, document => {
            const content = document.getElementById('content') as HTMLElement;

            enhanceLinksInContent(content);

            expect(content.querySelector('.link-preview-wrapper')).not.toBeNull();
            expect(content.querySelector('.link-preview-btn')).not.toBeNull();
        });
    });

    test('adds preview button for configured content extensions', () => {
        configureLinkPreview({ previewExts: ['.ini'] });
        try {
            withDOM(`<!DOCTYPE html><body>
                <article id="content"><a href="/config/app.ini">config</a></article>
            </body>`, document => {
                const content = document.getElementById('content') as HTMLElement;

                enhanceLinksInContent(content);

                expect(content.querySelector('.link-preview-wrapper')).not.toBeNull();
                expect(content.querySelector('.link-preview-btn')).not.toBeNull();
            });
        } finally {
            configureLinkPreview({ previewExts: ['.json', '.jsonl', '.yaml', '.yml', '.toml', '.html'] });
        }
    });

    test('enhances links idempotently after inline refresh', () => {
        withDOM(`<!DOCTYPE html><body>
            <article id="content"><a href="/docs/guide.md">guide</a></article>
        </body>`, document => {
            const content = document.getElementById('content') as HTMLElement;

            enhanceLinksInContent(content);
            enhanceLinksInContent(content);

            expect(content.querySelectorAll('.link-preview-wrapper')).toHaveLength(1);
            expect(content.querySelectorAll('.link-preview-btn')).toHaveLength(1);

            content.innerHTML = '<a href="/docs/next.md">next</a>';
            enhanceLinksInContent(content);

            expect(content.querySelectorAll('.link-preview-wrapper')).toHaveLength(1);
            expect(content.querySelectorAll('.link-preview-btn')).toHaveLength(1);
        });
    });
});
