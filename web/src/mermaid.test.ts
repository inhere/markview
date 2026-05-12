import { describe, expect, test } from 'bun:test';
import { select, selection } from 'd3-selection';
import { JSDOM } from 'jsdom';
import {
    buildMermaidContainerId,
    ensureD3TransitionSupport,
    parseMermaidContainerIndex,
    removeMermaidTooltipListeners,
    setupMermaidTooltipGuard,
} from './mermaid';

describe('web mermaid helpers', () => {
    test('buildMermaidContainerId uses stable prefix', () => {
        expect(buildMermaidContainerId(12)).toBe('mermaid-12');
    });

    test('parseMermaidContainerIndex extracts numeric suffix', () => {
        expect(parseMermaidContainerIndex('mermaid-7')).toBe(7);
        expect(parseMermaidContainerIndex('mermaid-invalid')).toBeNull();
    });

    test('ensureD3TransitionSupport installs d3 selection transition method', async () => {
        const selectionPrototype = selection.prototype as { transition?: unknown };
        const originalTransition = selectionPrototype.transition;
        const dom = new JSDOM('<!DOCTYPE html><body></body>');

        delete selectionPrototype.transition;
        expect(typeof select(dom.window.document.body).transition).toBe('undefined');

        try {
            await ensureD3TransitionSupport();
            expect(typeof select(dom.window.document.body).transition).toBe('function');
        } finally {
            if (originalTransition) {
                selectionPrototype.transition = originalTransition;
            } else {
                delete selectionPrototype.transition;
            }
        }
    });

    test('setupMermaidTooltipGuard handles node tooltip before mermaid d3 handler', () => {
        const dom = new JSDOM('<!DOCTYPE html><body><div id="root"><svg><g class="node" title="Node title"><rect></rect></g></svg></div></body>', {
            url: 'http://127.0.0.1/',
            pretendToBeVisual: true,
        });
        const previousDocument = globalThis.document;
        const previousWindow = globalThis.window;
        const previousElement = globalThis.Element;
        const previousHtmlElement = globalThis.HTMLElement;
        const previousNode = globalThis.Node;
        const previousSvgElement = globalThis.SVGElement;
        globalThis.document = dom.window.document;
        globalThis.window = dom.window as unknown as Window & typeof globalThis;
        globalThis.Element = dom.window.Element;
        globalThis.HTMLElement = dom.window.HTMLElement;
        globalThis.Node = dom.window.Node;
        globalThis.SVGElement = dom.window.SVGElement;

        try {
            const root = dom.window.document.getElementById('root') as HTMLElement;
            const node = root.querySelector('g.node') as SVGGElement;
            const rect = root.querySelector('rect') as SVGRectElement;
            let mermaidHandlerCalled = false;

            setupMermaidTooltipGuard(root);
            node.addEventListener('mouseover', () => {
                mermaidHandlerCalled = true;
            });

            rect.dispatchEvent(new dom.window.MouseEvent('mouseover', {
                bubbles: true,
                relatedTarget: dom.window.document.body,
            }));

            const tooltip = dom.window.document.querySelector('.mermaidTooltip') as HTMLElement | null;
            expect(mermaidHandlerCalled).toBe(false);
            expect(tooltip?.textContent).toBe('Node title');
            expect(tooltip?.style.opacity).toBe('0.9');
            expect(node.classList.contains('hover')).toBe(true);
        } finally {
            globalThis.document = previousDocument;
            globalThis.window = previousWindow;
            globalThis.Element = previousElement;
            globalThis.HTMLElement = previousHtmlElement;
            globalThis.Node = previousNode;
            globalThis.SVGElement = previousSvgElement;
        }
    });

    test('setupMermaidTooltipGuard blocks mermaid handler for empty title nodes', () => {
        const dom = new JSDOM('<!DOCTYPE html><body><div id="root"><svg><g class="node" title=""><rect></rect></g></svg></div></body>', {
            url: 'http://127.0.0.1/',
            pretendToBeVisual: true,
        });
        const previousDocument = globalThis.document;
        const previousWindow = globalThis.window;
        const previousElement = globalThis.Element;
        const previousHtmlElement = globalThis.HTMLElement;
        const previousNode = globalThis.Node;
        const previousSvgElement = globalThis.SVGElement;
        globalThis.document = dom.window.document;
        globalThis.window = dom.window as unknown as Window & typeof globalThis;
        globalThis.Element = dom.window.Element;
        globalThis.HTMLElement = dom.window.HTMLElement;
        globalThis.Node = dom.window.Node;
        globalThis.SVGElement = dom.window.SVGElement;

        try {
            const root = dom.window.document.getElementById('root') as HTMLElement;
            const node = root.querySelector('g.node') as SVGGElement;
            const rect = root.querySelector('rect') as SVGRectElement;
            let mermaidHandlerCalled = false;

            setupMermaidTooltipGuard(root);
            node.addEventListener('mouseover', () => {
                mermaidHandlerCalled = true;
            });

            rect.dispatchEvent(new dom.window.MouseEvent('mouseover', {
                bubbles: true,
                relatedTarget: dom.window.document.body,
            }));

            expect(mermaidHandlerCalled).toBe(false);
        } finally {
            globalThis.document = previousDocument;
            globalThis.window = previousWindow;
            globalThis.Element = previousElement;
            globalThis.HTMLElement = previousHtmlElement;
            globalThis.Node = previousNode;
            globalThis.SVGElement = previousSvgElement;
        }
    });

    test('removeMermaidTooltipListeners removes d3 node hover handlers', () => {
        const dom = new JSDOM('<!DOCTYPE html><body><div id="root"><svg><g class="node" title="Node title"></g></svg></div></body>');
        const previousElement = globalThis.Element;
        const previousSvgElement = globalThis.SVGElement;
        globalThis.Element = dom.window.Element;
        globalThis.SVGElement = dom.window.SVGElement;

        try {
            const root = dom.window.document.getElementById('root') as HTMLElement;
            const node = root.querySelector('g.node') as SVGGElement & {
                __on?: Array<{ type: string; listener: EventListener; options?: AddEventListenerOptions | boolean }>;
            };
            let mouseoverCalled = false;
            const mouseoverListener = () => {
                mouseoverCalled = true;
            };
            const clickListener = () => {};

            node.addEventListener('mouseover', mouseoverListener);
            node.addEventListener('click', clickListener);
            node.__on = [
                { type: 'mouseover', listener: mouseoverListener },
                { type: 'click', listener: clickListener },
            ];

            removeMermaidTooltipListeners(root);
            node.dispatchEvent(new dom.window.MouseEvent('mouseover', { bubbles: true }));

            expect(mouseoverCalled).toBe(false);
            expect(node.__on?.map(entry => entry.type)).toEqual(['click']);
        } finally {
            globalThis.Element = previousElement;
            globalThis.SVGElement = previousSvgElement;
        }
    });
});
