import { describe, expect, test } from 'bun:test';
import cssText from './style/app.css' with { type: 'text' };

function expectRule(pattern: RegExp) {
    expect(cssText).toMatch(pattern);
}

describe('layout CSS modes', () => {
    test('defines desktop rules for toc-middle and floating toc-right', () => {
        expect(cssText).toContain('@media (min-width: 1024px)');
        expectRule(/html\[data-layout="toc-middle"\]\s+\.app-shell\s*\{[^}]*grid-template-columns:\s*var\(--sidebar-width\)\s+var\(--toc-width\)\s+minmax\(0,\s*1fr\);[^}]*grid-template-areas:\s*"files toc content";/s);
        expectRule(/html\[data-layout="toc-right"\]\s+\.app-shell\s*\{[^}]*grid-template-columns:\s*var\(--sidebar-width\)\s+minmax\(0,\s*1fr\);[^}]*grid-template-areas:\s*"files content";/s);
        expectRule(/html\[data-layout="toc-right"\]\s+\.toc-pane\s*\{[^}]*position:\s*fixed;[^}]*transform:\s*translateX\(calc\(100% \+ 24px\)\);[^}]*opacity:\s*0;[^}]*pointer-events:\s*none;/s);
        expectRule(/html\[data-layout="toc-right"\]\s+body\.toc-floating-open\s+\.toc-pane\s*\{[^}]*transform:\s*translateX\(0\);[^}]*opacity:\s*1;[^}]*pointer-events:\s*auto;/s);
        expectRule(/html\[data-layout="toc-right"\]\s+\.toc-toggle-button\s*\{[^}]*display:\s*inline-flex;[^}]*position:\s*fixed;/s);
    });

    test('keeps mobile layout compact and supports collapsed files width', () => {
        expect(cssText).toContain('@media (max-width: 1023px)');
        expectRule(/@media \(max-width:\s*1023px\)\s*\{[\s\S]*\.app-shell\s*\{[^}]*display:\s*block;/);
        expectRule(/@media \(max-width:\s*1023px\)\s*\{[\s\S]*\.files-pane,\s*\.toc-pane\s*\{[^}]*display:\s*none;/);
        expectRule(/body\.sidebar-collapsed\s+\.files-pane\s*\{[^}]*width:\s*var\(--sidebar-collapsed-width\);/s);
        expectRule(/html\[data-layout="toc-middle"\]\s+body\.sidebar-collapsed\s+\.app-shell\s*\{[^}]*grid-template-columns:\s*var\(--sidebar-collapsed-width\)\s+var\(--toc-width\)\s+minmax\(0,\s*1fr\);/s);
        expect(cssText).not.toMatch(/(?:^|})\s*body\.sidebar-collapsed\s+\.toc-pane\s*\{\s*display:\s*none;/);
    });
});
