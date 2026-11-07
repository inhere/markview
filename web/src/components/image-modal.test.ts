import { describe, expect, test } from 'bun:test';
import { JSDOM } from 'jsdom';
import { enhanceImagesInContent } from './image-modal';

function withDOM(html: string, run: (document: Document) => void) {
    const dom = new JSDOM(html, {
        url: 'http://127.0.0.1/docs/readme.md',
        pretendToBeVisual: true,
    });
    const previousDocument = globalThis.document;
    const previousWindow = globalThis.window;

    try {
        globalThis.document = dom.window.document;
        globalThis.window = dom.window as unknown as Window & typeof globalThis;
        run(dom.window.document);
    } finally {
        globalThis.document = previousDocument;
        globalThis.window = previousWindow;
    }
}

describe('image enhancement', () => {
    test('renders SVG links as images with fullscreen controls', () => {
        withDOM(`<!DOCTYPE html><body>
            <article id="content">
                <a href="/assets/architecture.svg" title="Architecture">Architecture diagram</a>
                <a href="/docs/guide.md">Guide</a>
            </article>
        </body>`, document => {
            const content = document.getElementById('content') as HTMLElement;

            enhanceImagesInContent(content);

            const image = content.querySelector('.image-container img') as HTMLImageElement | null;
            expect(image).not.toBeNull();
            expect(image?.src).toBe('http://127.0.0.1/assets/architecture.svg');
            expect(image?.alt).toBe('Architecture diagram');
            expect(image?.title).toBe('Architecture');
            expect(content.querySelector('.image-fullscreen-btn')).not.toBeNull();
            expect(content.querySelector('a[href$="architecture.svg"]')).toBeNull();
            expect(content.querySelector('a[href$="guide.md"]')).not.toBeNull();
        });
    });

    test('does not replace downloadable SVG links', () => {
        withDOM(`<!DOCTYPE html><body>
            <article id="content">
                <a href="/assets/architecture.svg" download>Download SVG</a>
            </article>
        </body>`, document => {
            const content = document.getElementById('content') as HTMLElement;

            enhanceImagesInContent(content);

            expect(content.querySelector('a[download]')).not.toBeNull();
            expect(content.querySelector('.image-container')).toBeNull();
        });
    });
});
