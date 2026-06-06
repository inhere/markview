import { describe, expect, test } from 'bun:test';
import cssText from './style/app.css' with { type: 'text' };

function expectRule(pattern: RegExp) {
    expect(cssText).toMatch(pattern);
}

describe('layout CSS modes', () => {
    test('defines desktop rules for toc-middle and floating toc-right', () => {
        expect(cssText).toContain('@media (min-width: 1024px)');
        expectRule(/--toc-width:\s*280px;/);
        expectRule(/html\[data-layout="toc-middle"\]\s+\.app-shell\s*\{[^}]*grid-template-columns:\s*var\(--sidebar-width\)\s+minmax\(0,\s*1fr\);[^}]*grid-template-areas:\s*"files content";/s);
        expectRule(/html\[data-layout="toc-right"\]\s+\.app-shell\s*\{[^}]*grid-template-columns:\s*var\(--sidebar-width\)\s+minmax\(0,\s*1fr\);[^}]*grid-template-areas:\s*"files content";/s);
        expectRule(/html\[data-layout="toc-middle"\]\s+\.toc-pane\s*\{[^}]*position:\s*fixed;[^}]*top:\s*56px;[^}]*left:\s*var\(--sidebar-width\);[^}]*box-shadow:\s*none;[^}]*width:\s*var\(--toc-width\);/s);
        expectRule(/html\[data-layout="toc-right"\]\s+\.toc-pane\s*\{[^}]*position:\s*fixed;[^}]*border-radius:\s*0;[^}]*box-shadow:\s*none;[^}]*transform:\s*translateX\(calc\(100% - 44px\)\);[^}]*opacity:\s*1;/s);
        expectRule(/html\[data-layout="toc-right"\]\s+body\.toc-floating-open\s+\.toc-pane\s*\{[^}]*transform:\s*translateX\(0\);[^}]*opacity:\s*1;[^}]*pointer-events:\s*auto;/s);
        expectRule(/html\[data-layout="compact"\]\s+\.app-shell\s*\{[^}]*grid-template-rows:\s*minmax\(8rem,\s*1fr\)\s+minmax\(16rem,\s*2fr\);/s);
        expectRule(/html\[data-layout="compact"\]\s+\.toc-pane\s*\{[^}]*margin:\s*0\s+0\s+8px;/s);
        expectRule(/html\[data-layout="compact"\]\s+\.files-pane\.sidebar\s*\{[^}]*padding-bottom:\s*8px;/s);
        expectRule(/html\[data-layout="compact"\]\s+\.files-pane\s+\.sidebar-panels\s*\{[^}]*padding-bottom:\s*0;/s);
        expectRule(/\.toc-pane\s*\{[^}]*border-radius:\s*0;/s);
        expectRule(/\.toc-section-toggle\s*\{[^}]*display:\s*inline-flex;/s);
        expectRule(/body:not\(\.toc-floating-open\)\s+\.toc-pane\s+\.toc-section-label-text\s*\{[^}]*display:\s*none;/s);
        expectRule(/body:not\(\.toc-floating-open\)\s+\.toc-pane\s+\.toc-container\s*\{[^}]*display:\s*none;/s);
        expectRule(/html\[data-layout="compact"\]\s+body:not\(\.toc-floating-open\)\s+\.app-shell\s*\{[^}]*grid-template-rows:\s*minmax\(0,\s*1fr\)\s+44px;/s);
        expectRule(/html\[data-layout="toc-middle"\]\s+body:not\(\.toc-floating-open\)\s+\.toc-pane\s*\{[^}]*width:\s*44px;[^}]*overflow:\s*hidden;/s);
        expect(cssText).not.toMatch(/html\[data-layout="toc-right"\]\s+\.sidebar-icons\s*\{[^}]*display:\s*flex;/s);
        expect(cssText).not.toContain('.toc-toggle-button');
    });

    test('keeps mobile layout compact and supports collapsed files width', () => {
        expect(cssText).toContain('@media (max-width: 1023px)');
        expectRule(/@media \(max-width:\s*1023px\)\s*\{[\s\S]*\.app-shell\s*\{[^}]*display:\s*block;/);
        expectRule(/@media \(max-width:\s*1023px\)\s*\{[\s\S]*\.files-pane,\s*\.toc-pane\s*\{[^}]*display:\s*none;/);
        expectRule(/body\.sidebar-collapsed\s+\.files-pane\s*\{[^}]*width:\s*var\(--sidebar-collapsed-width\);/s);
        expectRule(/html\[data-layout="toc-middle"\]\s+body\.sidebar-collapsed\s+\.toc-pane\s*\{[^}]*left:\s*var\(--sidebar-collapsed-width\);/s);
        expect(cssText).not.toMatch(/(?:^|})\s*body\.sidebar-collapsed\s+\.toc-pane\s*\{\s*display:\s*none;/);
    });

    test('defines preview-active and mobile fallback layout rules', () => {
        expect(cssText).toContain('preview-active');
        expect(cssText).toContain('toc-floating-open');
        expectRule(/--preview-width:\s*clamp\(420px,\s*40vw,\s*960px\);/);
        expect(cssText).not.toContain('.toc-toggle-button');
        expect(cssText).not.toMatch(/html\[data-layout="toc-middle"\]\s+body\.preview-active\s+\.toc-pane\s*\{[^}]*display:\s*none;/s);
        expectRule(/html\[data-layout="toc-middle"\]\s+body\.preview-active\s+\.app-shell\s*\{[^}]*grid-template-columns:\s*var\(--sidebar-width\)\s+minmax\(0,\s*1fr\);[^}]*grid-template-areas:\s*"files content";[^}]*padding-right:\s*var\(--preview-width\);/s);
        expectRule(/html\[data-layout="toc-right"\]\s+body\.preview-active\s+\.toc-pane\s*\{[^}]*right:\s*calc\(var\(--preview-width\) \+ 16px\);/s);
        expectRule(/html\[data-layout="toc-right"\]\s+body\.preview-active:not\(\.toc-floating-open\)\s+\.toc-pane\s*\{[^}]*right:\s*var\(--preview-width\);[^}]*transform:\s*translateX\(calc\(100% - 44px\)\);/s);
        expectRule(/\.content-wrapper\s*\{[^}]*position:\s*relative;[^}]*overflow:\s*auto;/s);
        expectRule(/\.content-search-wrapper\s*\{[^}]*position:\s*absolute;[^}]*left:\s*30px;/s);
    });

    test('keeps toc links clean without bottom borders', () => {
        expectRule(/\.toc-link\s*\{[^}]*border-left:\s*3px solid transparent;/s);
        expectRule(/\.toc-link\s*\{[^}]*border-bottom:\s*none;/s);
    });

    test('uses containment for heavy markdown blocks to reduce long page scroll work', () => {
        expectRule(/\.mermaid-container\s*\{[^}]*content-visibility:\s*auto;[^}]*contain-intrinsic-size:\s*320px;/s);
        expectRule(/\.table-scroll-container\s*\{[^}]*content-visibility:\s*auto;[^}]*contain-intrinsic-size:\s*240px;/s);
        expectRule(/\.table-scroll-body\s*\{[^}]*overscroll-behavior:\s*contain;/s);
    });

    test('keeps desktop scrolling inside panes without root page scrollbars', () => {
        expectRule(/@media \(min-width:\s*1024px\)\s*\{[\s\S]*html,\s*body\s*\{[^}]*height:\s*100%;[^}]*overflow:\s*hidden;/);
        expectRule(/\.content-wrapper\s*\{[^}]*overflow:\s*auto;/s);
        expectRule(/\.sidebar-resize-handle\s*\{[^}]*right:\s*0;[^}]*width:\s*8px;/s);
        expect(cssText).not.toMatch(/\.sidebar-resize-handle\s*\{[^}]*right:\s*-[0-9]/s);
        expectRule(/@media \(max-width:\s*1023px\)\s*\{[\s\S]*html,\s*body\s*\{[^}]*height:\s*auto;[^}]*overflow:\s*visible;/);
    });
});
