import { describe, expect, test } from 'bun:test';
import templateHtml from '../template.html' with { type: 'text' };
import appCssText from './style/app.css' with { type: 'text' };
import tokensCssText from './style/tokens.css' with { type: 'text' };
import layoutCssText from './style/layout.css' with { type: 'text' };
import toolbarCssText from './style/toolbar.css' with { type: 'text' };
import sidebarCssText from './style/sidebar.css' with { type: 'text' };
import contentCssText from './style/content.css' with { type: 'text' };
import overlaysCssText from './style/overlays.css' with { type: 'text' };

const cssText = [
    appCssText,
    tokensCssText,
    layoutCssText,
    toolbarCssText,
    sidebarCssText,
    contentCssText,
    overlaysCssText,
].join('\n');

function expectRule(pattern: RegExp) {
    expect(cssText).toMatch(pattern);
}

describe('layout CSS modes', () => {
    test('keeps app css as ordered style entrypoint', () => {
        expect(appCssText).toContain('@import "./tokens.css";');
        expect(appCssText).toContain('@import "./layout.css";');
        expect(appCssText).toContain('@import "./toolbar.css";');
        expect(appCssText).toContain('@import "./sidebar.css";');
        expect(appCssText).toContain('@import "./content.css";');
        expect(appCssText).toContain('@import "./overlays.css";');
        expect(appCssText.indexOf('tokens.css')).toBeLessThan(appCssText.indexOf('layout.css'));
        expect(appCssText.indexOf('layout.css')).toBeLessThan(appCssText.indexOf('toolbar.css'));
        expect(appCssText.indexOf('toolbar.css')).toBeLessThan(appCssText.indexOf('sidebar.css'));
        expect(appCssText.indexOf('sidebar.css')).toBeLessThan(appCssText.indexOf('content.css'));
        expect(appCssText.indexOf('content.css')).toBeLessThan(appCssText.indexOf('overlays.css'));
    });

    test('uses warm low-glare default theme tokens', () => {
        expect(tokensCssText).toContain('--bg-canvas: #f3f1ea;');
        expect(tokensCssText).toContain('--bg-paper: #fffdf6;');
        expect(tokensCssText).toContain('--bg-surface: #fbf8ef;');
        expect(tokensCssText).toContain('--text-body: #3f443a;');
        expect(tokensCssText).toContain('--accent-primary: #2f6f68;');
        expect(tokensCssText).toContain('--accent-subtle: #e8f2ee;');
    });

    test('keeps reader surface quiet and responsive', () => {
        expect(contentCssText).toMatch(/\.paper\s*\{[^}]*border-radius:\s*8px;[^}]*padding:\s*clamp\(34px,\s*5vw,\s*72px\);/s);
        expect(layoutCssText).toMatch(/\.content-wrapper\s*\{[^}]*padding:\s*clamp\(20px,\s*4vw,\s*52px\);/s);
        expect(contentCssText).toMatch(/\.content-inner\s*\{[^}]*align-self:\s*flex-start;/s);
        expect(contentCssText).toMatch(/\.content-inner::after\s*\{[^}]*content:\s*"";[^}]*display:\s*block;[^}]*height:\s*clamp\(64px,\s*9vh,\s*110px\);/s);
        expect(overlaysCssText).toMatch(/\.preview-body::after\s*\{[^}]*content:\s*"";[^}]*display:\s*block;[^}]*height:\s*clamp\(64px,\s*9vh,\s*110px\);/s);
        expect(toolbarCssText).not.toMatch(/\.toolbar\.expanded\s*\{[^}]*opacity:\s*0\.5;/s);
    });

    test('renders global navigation as its own layout row without moving TOC controls', () => {
        expect(templateHtml).toContain('{{if .GlobalMode}}');
        expect(templateHtml).toContain('class="global-topbar"');
        expect(templateHtml).toContain('aria-label="Project navigation"');
        expect(layoutCssText).toMatch(/body\.global-mode\s*\{[^}]*display:\s*grid;[^}]*grid-template-rows:\s*44px\s+minmax\(0,\s*1fr\);/s);
        expect(layoutCssText).toMatch(/body\.global-mode\s+\.app-shell\s*\{[^}]*height:\s*auto;[^}]*min-height:\s*0;/s);
        expect(layoutCssText).toMatch(/\.global-topbar\s*\{[^}]*min-width:\s*0;[^}]*border-bottom:\s*1px solid var\(--border-light\);/s);
        expectRule(/html\[data-layout="toc-middle"\]\s+body:not\(\.toc-floating-open\)\s+\.toc-pane\s*\{[^}]*bottom:\s*16px;/s);
        expectRule(/html\[data-layout="toc-right"\]\s+body:not\(\.toc-floating-open\)\s+\.toc-pane\s*\{[^}]*bottom:\s*16px;/s);
    });

    test('defines content search trigger and centered overlay styles', () => {
        expect(toolbarCssText).toMatch(/\.content-search-trigger\s*\{[^}]*display:\s*inline-flex;/s);
        expect(overlaysCssText).toMatch(/\.content-search-wrapper\[hidden\]\s*\{[^}]*display:\s*none;/s);
        expect(overlaysCssText).toMatch(/\.content-search-panel\s*\{[^}]*position:\s*fixed;[^}]*top:\s*14vh;[^}]*left:\s*50%;[^}]*width:\s*min\(720px,\s*calc\(100vw - 32px\)\);[^}]*transform:\s*translateX\(-50%\);/s);
        expect(overlaysCssText).toMatch(/\.content-search-help\s*\{[^}]*padding:\s*8px\s+16px\s+10px;/s);
        expect(templateHtml).toContain('keyword !exclude');
        expect(templateHtml).toContain('path:docs/api keyword');
        expect(overlaysCssText).toMatch(/\.context-text\s*\{[^}]*min-width:\s*0;[^}]*overflow-wrap:\s*anywhere;/s);
        expect(overlaysCssText).toMatch(/@media \(max-width:\s*1023px\)\s*\{[\s\S]*\.content-search-panel\s*\{[^}]*top:\s*12px;[^}]*bottom:\s*12px;/s);
    });

    test('gives file change notifications responsive reading width', () => {
        expect(overlaysCssText).toMatch(/\.file-change-toast\s*\{[^}]*width:\s*min\(520px,\s*calc\(100vw - 48px\)\);/s);
    });

    test('defines desktop rules for toc-middle and floating toc-right', () => {
        expect(cssText).toContain('@media (min-width: 1024px)');
        expectRule(/--toc-width:\s*280px;/);
        expectRule(/html\[data-layout="toc-middle"\]\s+\.app-shell\s*\{[^}]*grid-template-columns:\s*var\(--sidebar-width\)\s+minmax\(0,\s*1fr\);[^}]*grid-template-areas:\s*"files content";/s);
        expectRule(/html\[data-layout="toc-right"\]\s+\.app-shell\s*\{[^}]*grid-template-columns:\s*var\(--sidebar-width\)\s+minmax\(0,\s*1fr\);[^}]*grid-template-areas:\s*"files content";/s);
        expectRule(/html\[data-layout="toc-middle"\]\s+\.toc-pane\s*\{[^}]*position:\s*fixed;[^}]*top:\s*56px;[^}]*left:\s*var\(--sidebar-width\);[^}]*box-shadow:\s*none;[^}]*width:\s*var\(--toc-width\);/s);
        expectRule(/html\[data-layout="toc-right"\]\s+\.toc-pane\s*\{[^}]*position:\s*fixed;[^}]*border-radius:\s*0;[^}]*box-shadow:\s*none;[^}]*transform:\s*none;[^}]*opacity:\s*1;/s);
        expectRule(/html\[data-layout="toc-right"\]\s+body\.toc-floating-open\s+\.toc-pane\s*\{[^}]*transform:\s*translateX\(0\);[^}]*opacity:\s*1;[^}]*pointer-events:\s*auto;/s);
        expectRule(/html\[data-layout="compact"\]\s+\.app-shell\s*\{[^}]*grid-template-rows:\s*minmax\(8rem,\s*1fr\)\s+minmax\(16rem,\s*2fr\);/s);
        expectRule(/html\[data-layout="compact"\]\s+\.toc-pane\s*\{[^}]*margin:\s*0\s+0\s+8px;/s);
        expectRule(/html\[data-layout="compact"\]\s+\.files-pane\.sidebar\s*\{[^}]*padding-bottom:\s*8px;/s);
        expectRule(/html\[data-layout="compact"\]\s+\.files-pane\s+\.sidebar-panels\s*\{[^}]*padding-bottom:\s*0;/s);
        expectRule(/\.toc-pane\s*\{[^}]*border-radius:\s*0;/s);
        expectRule(/\.toc-section-toggle\s*\{[^}]*display:\s*inline-flex;[^}]*border:\s*1px solid var\(--accent-border\);/s);
        expectRule(/body:not\(\.toc-floating-open\)\s+\.toc-pane\s+\.toc-section-label-text\s*\{[^}]*display:\s*none;/s);
        expectRule(/body:not\(\.toc-floating-open\)\s+\.toc-pane\s+\.toc-container\s*\{[^}]*display:\s*none;/s);
        expectRule(/html\[data-layout="compact"\]\s+body:not\(\.toc-floating-open\)\s+\.app-shell\s*\{[^}]*grid-template-rows:\s*minmax\(0,\s*1fr\)\s+44px;/s);
        expectRule(/html\[data-layout="toc-middle"\]\s+body:not\(\.toc-floating-open\)\s+\.toc-pane\s*\{[^}]*top:\s*auto;[^}]*left:\s*calc\(var\(--sidebar-width\) \+ 16px\);[^}]*bottom:\s*16px;[^}]*width:\s*48px;[^}]*height:\s*48px;/s);
        expectRule(/html\[data-layout="toc-middle"\]\s+body\.sidebar-collapsed:not\(\.toc-floating-open\)\s+\.toc-pane\s*\{[^}]*left:\s*calc\(var\(--sidebar-collapsed-width\) \+ 16px\);/s);
        expectRule(/html\[data-layout="toc-right"\]\s+body:not\(\.toc-floating-open\)\s+\.toc-pane\s*\{[^}]*top:\s*auto;[^}]*right:\s*16px;[^}]*bottom:\s*16px;[^}]*width:\s*48px;[^}]*height:\s*48px;[^}]*transform:\s*none;/s);
        expectRule(/\.toc-section-toggle\s+svg\s*\{[^}]*width:\s*24px;[^}]*height:\s*24px;/s);
        expect(cssText).not.toMatch(/html\[data-layout="toc-right"\]\s+\.sidebar-icons\s*\{[^}]*display:\s*flex;/s);
        expect(cssText).not.toContain('.toc-toggle-button');
    });

    test('keeps mobile layout compact and supports collapsed files width', () => {
        expect(cssText).toContain('@media (max-width: 1023px)');
        expectRule(/@media \(max-width:\s*1023px\)\s*\{[\s\S]*\.app-shell\s*\{[^}]*display:\s*block;/);
        expectRule(/@media \(max-width:\s*1023px\)\s*\{[\s\S]*\.files-pane,\s*\.toc-pane\s*\{[^}]*display:\s*none;/);
        expectRule(/body\.sidebar-collapsed\s+\.files-pane\s*\{[^}]*width:\s*var\(--sidebar-collapsed-width\);/s);
        expectRule(/html\[data-layout="toc-middle"\]\s+body\.sidebar-collapsed\s+\.toc-pane\s*\{[^}]*left:\s*var\(--sidebar-collapsed-width\);/s);
        expectRule(/@media \(max-width:\s*1023px\)\s*\{[\s\S]*body\.preview-active\s+\.preview-panel\s*\{[^}]*width:\s*100%;/);
        expect(cssText).not.toMatch(/(?:^|})\s*body\.sidebar-collapsed\s+\.toc-pane\s*\{\s*display:\s*none;/);
    });

    test('lets the compact collapsed sidebar span the full viewport height', () => {
        expectRule(/html\[data-layout="compact"\]\s+body\.sidebar-collapsed\s+\.files-pane\s*\{[^}]*grid-row:\s*1\s*\/\s*-1;[^}]*height:\s*100vh;/s);
    });

    test('defines preview-active and mobile fallback layout rules', () => {
        expect(cssText).toContain('preview-active');
        expect(cssText).toContain('toc-floating-open');
        expectRule(/--preview-width:\s*clamp\(420px,\s*40vw,\s*960px\);/);
        expect(cssText).not.toContain('.toc-toggle-button');
        expect(cssText).not.toMatch(/html\[data-layout="toc-middle"\]\s+body\.preview-active\s+\.toc-pane\s*\{[^}]*display:\s*none;/s);
        expectRule(/html\[data-layout="toc-middle"\]\s+body\.preview-active\s+\.app-shell\s*\{[^}]*grid-template-columns:\s*var\(--sidebar-width\)\s+minmax\(0,\s*1fr\);[^}]*grid-template-areas:\s*"files content";[^}]*padding-right:\s*var\(--preview-width\);/s);
        expectRule(/html\[data-layout="toc-right"\]\s+body\.preview-active\s+\.toc-pane\s*\{[^}]*right:\s*calc\(var\(--preview-width\) \+ 16px\);/s);
        expectRule(/html\[data-layout="toc-right"\]\s+body\.preview-active:not\(\.toc-floating-open\)\s+\.toc-pane\s*\{[^}]*right:\s*calc\(var\(--preview-width\) \+ 16px\);[^}]*transform:\s*none;/s);
        expectRule(/\.content-wrapper\s*\{[^}]*position:\s*relative;[^}]*overflow:\s*auto;/s);
        expectRule(/\.content-search-wrapper\s*\{[^}]*position:\s*fixed;[^}]*inset:\s*0;/s);
    });

    test('lets html iframe previews fill the preview panel height', () => {
        expectRule(/\.preview-content\s*\{[^}]*display:\s*flex;[^}]*flex-direction:\s*column;[^}]*min-height:\s*0;/s);
        expectRule(/\.preview-body\s*\{[^}]*flex:\s*1;[^}]*min-height:\s*0;/s);
        expectRule(/\.preview-body\s+\.preview-html-frame\s*\{[^}]*display:\s*block;[^}]*width:\s*100%;[^}]*height:\s*100%;/s);
    });

    test('defines a left edge resize handle for the preview panel', () => {
        expectRule(/\.preview-resize-handle\s*\{[^}]*left:\s*0;[^}]*width:\s*6px;[^}]*cursor:\s*col-resize;/s);
        expectRule(/\.preview-resize-handle:hover,\s*\.preview-resize-handle\.is-resizing\s*\{[^}]*background:\s*var\(--accent-border\);/s);
        expectRule(/body\.preview-is-resizing\s+\.preview-panel\s*\{[^}]*transition:\s*none;/s);
        expectRule(/body\.preview-is-resizing\s+\.preview-content\s*\{[^}]*pointer-events:\s*none;/s);
    });

    test('keeps toolbar version collapsed until settings are open and emphasizes sidebar collapse button', () => {
        expectRule(/\.toolbar-version\s*\{[^}]*display:\s*none;/s);
        expectRule(/\.toolbar\.expanded\s+\.toolbar-version\s*\{[^}]*display:\s*inline-flex;/s);
        expectRule(/\.sidebar-collapse-btn\s*\{[^}]*width:\s*28px;[^}]*height:\s*28px;[^}]*border:\s*1px solid var\(--border-light\);[^}]*color:\s*var\(--text-heading\);/s);
        expectRule(/\.sidebar-collapse-btn svg\s*\{[^}]*width:\s*17px;[^}]*height:\s*17px;[^}]*stroke-width:\s*2\.7;/s);
    });

    test('keeps toc links clean without bottom borders', () => {
        expectRule(/\.toc-link\s*\{[^}]*border-left:\s*3px solid transparent;/s);
        expectRule(/\.toc-link\s*\{[^}]*border-bottom:\s*none;/s);
    });

    test('uses containment for heavy markdown blocks to reduce long page scroll work', () => {
        expectRule(/\.mermaid-container\s*\{[^}]*content-visibility:\s*auto;[^}]*contain-intrinsic-size:\s*320px;/s);
        expectRule(/\.table-scroll-container\s*\{[^}]*content-visibility:\s*auto;[^}]*contain-intrinsic-size:\s*240px;/s);
        expectRule(/\.table-scroll-body\s*\{[^}]*overflow-x:\s*auto;[^}]*overflow-y:\s*hidden;[^}]*overscroll-behavior:\s*auto;/s);
    });

    test('keeps markdown table frames content-sized without trapping page wheel scroll', () => {
        expectRule(/\.table-scroll-container\s*\{[^}]*width:\s*fit-content;[^}]*max-width:\s*100%;/s);
        expectRule(/\.table-scroll-body\s*\{[^}]*overflow-x:\s*auto;[^}]*overflow-y:\s*hidden;[^}]*overscroll-behavior:\s*auto;/s);
        expect(cssText).not.toMatch(/\.table-scroll-body\s*\{[^}]*overscroll-behavior:\s*contain;/s);
    });

    test('keeps desktop scrolling inside panes without root page scrollbars', () => {
        expectRule(/@media \(min-width:\s*1024px\)\s*\{[\s\S]*html,\s*body\s*\{[^}]*height:\s*100%;[^}]*overflow:\s*hidden;/);
        expectRule(/\.content-wrapper\s*\{[^}]*overflow:\s*auto;/s);
        expectRule(/\.sidebar\s*\{[^}]*position:\s*relative;/s);
        expectRule(/\.sidebar-resize-handle\s*\{[^}]*right:\s*0;[^}]*width:\s*4px;/s);
        expect(cssText).not.toMatch(/\.sidebar-resize-handle\s*\{[^}]*right:\s*-[0-9]/s);
        expectRule(/@media \(max-width:\s*1023px\)\s*\{[\s\S]*html,\s*body\s*\{[^}]*height:\s*auto;[^}]*overflow:\s*visible;/);
    });
});
