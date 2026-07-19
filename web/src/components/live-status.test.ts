import { describe, expect, test } from 'bun:test';
import { JSDOM } from 'jsdom';
import { setupLiveReloadStatus } from './live-status';

describe('live reload status', () => {
    test('restores live status after reconnecting from offline', () => {
        const source: {
            onopen: null | (() => void);
            onerror: null | (() => void);
            onmessage: null | ((event: { data: string }) => void);
        } = {
            onopen: null,
            onerror: null,
            onmessage: null,
        };

        const liveDot = createFakeDot();
        const statusText = { innerText: 'Live' };

        setupLiveReloadStatus(source, liveDot, statusText, async () => {});

        source.onerror?.();
        expect(statusText.innerText).toBe('Offline');
        expect(liveDot.style.backgroundColor).toBe('var(--status-warn)');

        source.onopen?.();
        expect(statusText.innerText).toBe('Live');
        expect(liveDot.style.backgroundColor).toBe('');
    });

    test('renders at most three safe clickable file links', async () => {
        const source = createFakeEventSource();
        const files = ['docs/<unsafe>.md', 'docs/two.md', 'docs/three.md', 'docs/four.md'];
        const fileTreeJSON = JSON.stringify(files.map(file => ({
            name: file,
            href: `/${file}`,
            matchPath: file,
            kind: 'file',
            navigable: true,
        })));

        await withFileTreeDOM(async () => {
            setupLiveReloadStatus(source, null, null, async () => {});
            source.onmessage?.({
                data: JSON.stringify({ type: 'reload', files }),
            });

            const links = [...document.querySelectorAll<HTMLAnchorElement>('.toast-file')];
            expect(links).toHaveLength(3);
            expect(links[0].textContent).toBe('docs/<unsafe>.md');
            expect(links[0].innerHTML).not.toContain('<unsafe>');
            expect(links[0].getAttribute('href')).toBe('/docs/%3Cunsafe%3E.md');
            expect(document.querySelector('.toast-count')?.textContent).toBe('还有 1 个文件');
        }, { fileTreeJSON });
    });

    test('refreshes file tree after SSE reconnects from offline', async () => {
        const source = createFakeEventSource();
        let fetchCount = 0;

        await withFileTreeDOM(async () => {
            const previousFetch = globalThis.fetch;
            globalThis.fetch = async () => {
                fetchCount++;
                return new Response('[]', { status: 200 });
            };

            try {
                setupLiveReloadStatus(source, null, null, async () => {});

                source.onopen?.();
                await flushPromises();
                expect(fetchCount).toBe(0);

                source.onerror?.();
                source.onopen?.();
                await flushPromises();

                expect(fetchCount).toBe(1);
            } finally {
                globalThis.fetch = previousFetch;
            }
        });
    });

    test('refreshes file tree when SSE files include a path missing from local tree', async () => {
        const source = createFakeEventSource();
        let fetchCount = 0;
        let pageRefreshCount = 0;

        await withFileTreeDOM(async () => {
            const previousFetch = globalThis.fetch;
            globalThis.fetch = async () => {
                fetchCount++;
                return new Response('[]', { status: 200 });
            };

            try {
                setupLiveReloadStatus(source, null, null, async () => {
                    pageRefreshCount++;
                });

                source.onmessage?.({
                    data: JSON.stringify({
                        type: 'reload',
                        files: ['new.md'],
                    }),
                });
                await flushPromises();

                expect(fetchCount).toBe(1);
                expect(pageRefreshCount).toBe(0);
            } finally {
                globalThis.fetch = previousFetch;
            }
        });
    });

    test('does not refresh file tree when SSE files already exist in local tree', async () => {
        const source = createFakeEventSource();
        let fetchCount = 0;
        let pageRefreshCount = 0;

        await withFileTreeDOM(async () => {
            const previousFetch = globalThis.fetch;
            globalThis.fetch = async () => {
                fetchCount++;
                return new Response('[]', { status: 200 });
            };

            try {
                setupLiveReloadStatus(source, null, null, async () => {
                    pageRefreshCount++;
                });

                source.onmessage?.({
                    data: JSON.stringify({
                        type: 'reload',
                        files: ['current.md'],
                    }),
                });
                await flushPromises();

                expect(fetchCount).toBe(0);
                expect(pageRefreshCount).toBe(1);
            } finally {
                globalThis.fetch = previousFetch;
            }
        }, {
            fileTreeJSON: `[{"name":"current.md","href":"/current.md","matchPath":"current.md","kind":"file","navigable":true}]`,
            currentFilePathJSON: `"current.md"`,
        });
    });
});

function createFakeDot() {
    const classNames = new Set<string>();

    return {
        style: {
            backgroundColor: '',
        },
        classList: {
            add(name: string) {
                classNames.add(name);
            },
            remove(name: string) {
                classNames.delete(name);
            },
            contains(name: string) {
                return classNames.has(name);
            },
        },
    };
}

function createFakeEventSource() {
    return {
        onopen: null as null | (() => void),
        onerror: null as null | (() => void),
        onmessage: null as null | ((event: { data: string }) => void),
    };
}

async function withFileTreeDOM(
    run: () => Promise<void>,
    options: { fileTreeJSON?: string; currentFilePathJSON?: string } = {},
) {
    const dom = new JSDOM(`<!DOCTYPE html><body>
        <div id="file-tree"></div>
        <script id="file-tree-data" type="application/json">${options.fileTreeJSON ?? `[{"name":"existing.md","href":"/existing.md","matchPath":"existing.md","kind":"file","navigable":true}]`}</script>
        <script id="current-file-path-data" type="application/json">${options.currentFilePathJSON ?? `"current.md"`}</script>
    </body>`);

    const previousDocument = globalThis.document;
    const previousWindow = globalThis.window;
    const previousHTMLElement = globalThis.HTMLElement;
    const previousHTMLScriptElement = globalThis.HTMLScriptElement;
    const previousRequestAnimationFrame = globalThis.requestAnimationFrame;
    const previousSetTimeout = globalThis.setTimeout;
    const previousClearTimeout = globalThis.clearTimeout;

    try {
        globalThis.document = dom.window.document;
        globalThis.window = dom.window as unknown as Window & typeof globalThis;
        globalThis.HTMLElement = dom.window.HTMLElement;
        globalThis.HTMLScriptElement = dom.window.HTMLScriptElement;
        globalThis.requestAnimationFrame = callback => {
            callback(0);
            return 0;
        };
        globalThis.setTimeout = ((callback: TimerHandler) => {
            if (typeof callback === 'function') {
                callback();
            }
            return 0;
        }) as typeof globalThis.setTimeout;
        globalThis.clearTimeout = (() => {}) as typeof globalThis.clearTimeout;

        await run();
    } finally {
        globalThis.document = previousDocument;
        globalThis.window = previousWindow;
        globalThis.HTMLElement = previousHTMLElement;
        globalThis.HTMLScriptElement = previousHTMLScriptElement;
        globalThis.requestAnimationFrame = previousRequestAnimationFrame;
        globalThis.setTimeout = previousSetTimeout;
        globalThis.clearTimeout = previousClearTimeout;
    }
}

async function flushPromises() {
    await Promise.resolve();
    await Promise.resolve();
}
