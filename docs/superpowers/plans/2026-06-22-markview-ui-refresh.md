# MarkView UI Refresh Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 MarkView 默认 UI 调整为更护眼、低干扰的工程文档阅读器界面，并把内容搜索迁移为独立 Search icon + overlay。

**Architecture:** 先机械拆分 `app.css`，保持构建入口 `web/src/app.ts` 不变；再用 CSS tokens 和局部模板调整完成视觉 refresh；最后把现有 `content-search.ts` 的输入框容器迁移到 overlay，只新增打开/关闭、快捷键和焦点控制，不重写搜索 API 或结果跳转逻辑。

**Tech Stack:** Go、HTML template、TypeScript、Bun build/test、jsdom、CSS。

设计文档：[2026-06-22-markview-ui-refresh-design.md](../specs/2026-06-22-markview-ui-refresh-design.md)

预览稿：[../../ui-refresh-mockup.html](../../ui-refresh-mockup.html)

---

## File Structure

CSS 拆分后的职责：

- `web/src/style/app.css`: 样式入口，只保留 `@import` 顺序。
- `web/src/style/tokens.css`: 默认护眼色板、暗色/system/theme token、字体和尺寸变量。
- `web/src/style/layout.css`: reset、body、app shell、desktop/mobile 布局、preview active 布局。
- `web/src/style/toolbar.css`: 右上角 Search/Settings 工具区、设置面板、按钮、select。
- `web/src/style/sidebar.css`: sidebar、文件树、TOC、文件搜索框、sidebar resize。
- `web/src/style/content.css`: paper、Markdown 排版、raw link、file meta、表格和代码块阅读样式。
- `web/src/style/overlays.css`: content search overlay、preview panel、modal、toast、Mermaid/image 控制条。

TypeScript/模板职责：

- `web/template.html`: 增加 Search icon button；将 `#content-search` 调整为 overlay DOM；设置面板 DOM 保持原有控制节点和 id。
- `web/template-main.html`: 将 raw Markdown 按钮从 emoji 文案改为 SVG icon + 文案。
- `web/src/components/content-search.ts`: 复用现有搜索、渲染、跳转；新增 overlay 打开/关闭、`Ctrl/Cmd+K`、焦点回收。
- `web/src/components/content-search.test.ts`: 补搜索 overlay 行为测试，保留已有 `renderResults` 测试。
- `web/src/layout-css.test.ts`: 调整 CSS import 拆分后的文本断言，新增关键视觉和 overlay 规则断言。

提交注意：仓库当前可能已有用户改动 `Makefile`、`docs/TODO.md`。每次提交只 `git add` 本计划列出的文件，不要 `git add .`。

---

### Task 1: 机械拆分 CSS

**Files:**
- Modify: `web/src/style/app.css`
- Create: `web/src/style/tokens.css`
- Create: `web/src/style/layout.css`
- Create: `web/src/style/toolbar.css`
- Create: `web/src/style/sidebar.css`
- Create: `web/src/style/content.css`
- Create: `web/src/style/overlays.css`
- Modify: `web/src/layout-css.test.ts`

- [ ] **Step 1: 复制当前 CSS 到临时工作副本**

Run:

```powershell
Copy-Item web\src\style\app.css web\src\style\app.css.before-ui-split
```

Expected: `web/src/style/app.css.before-ui-split` 存在，仅作为本地拆分辅助文件，不提交。

- [ ] **Step 2: 创建 CSS 分文件并搬运现有块**

按当前 `app.css` 注释和选择器分区机械搬运，不改选择器、不改属性值、不改顺序语义。

搬运规则：

```text
:root, [data-color-scheme], [data-theme]       -> tokens.css
reset, html/body, app-shell, desktop/mobile    -> layout.css
.toolbar*, .toolbar-*                          -> toolbar.css
.sidebar*, .files-*, .file-tree*, .toc-*       -> sidebar.css
.content-wrapper, .paper*, markdown typography -> content.css
.content-search*, .preview*, .modal*, .toast*  -> overlays.css
```

搬运后 `web/src/style/app.css` 只保留：

```css
@import "./tokens.css";
@import "./layout.css";
@import "./toolbar.css";
@import "./sidebar.css";
@import "./content.css";
@import "./overlays.css";
```

- [ ] **Step 3: 删除本地辅助副本**

Run:

```powershell
Remove-Item -LiteralPath web\src\style\app.css.before-ui-split
```

Expected: 辅助副本已删除，工作区只剩真实 CSS 文件变更。

- [ ] **Step 4: 更新 CSS 文本测试以覆盖分文件 import**

Modify `web/src/layout-css.test.ts` 顶部 import，改为读取入口 `app.css` 和分文件文本。

```ts
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
```

新增一个 import 顺序测试：

```ts
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
```

- [ ] **Step 5: 运行 CSS 相关测试**

Run:

```powershell
cd web
bun test src/layout-css.test.ts
```

Expected: all tests pass.

- [ ] **Step 6: 验证前端构建**

Run:

```powershell
cd web
bun run build
```

Expected: build exits 0 and `web/dist/app.css` is generated.

- [ ] **Step 7: 提交 CSS 拆分**

Run:

```powershell
git status --short
git add web/src/style/app.css web/src/style/tokens.css web/src/style/layout.css web/src/style/toolbar.css web/src/style/sidebar.css web/src/style/content.css web/src/style/overlays.css web/src/layout-css.test.ts
git commit -m "refactor: split frontend css by responsibility"
```

Expected: commit succeeds. Do not stage `Makefile` or `docs/TODO.md`.

---

### Task 2: 应用护眼默认主题和阅读器视觉

**Files:**
- Modify: `web/src/style/tokens.css`
- Modify: `web/src/style/layout.css`
- Modify: `web/src/style/toolbar.css`
- Modify: `web/src/style/sidebar.css`
- Modify: `web/src/style/content.css`
- Modify: `web/src/layout-css.test.ts`
- Modify: `web/template-main.html`

- [ ] **Step 1: 为护眼默认 token 写 CSS 测试**

Add to `web/src/layout-css.test.ts`:

```ts
test('uses warm low-glare default theme tokens', () => {
    expect(tokensCssText).toContain('--bg-canvas: #f3f1ea;');
    expect(tokensCssText).toContain('--bg-paper: #fffdf6;');
    expect(tokensCssText).toContain('--bg-surface: #fbf8ef;');
    expect(tokensCssText).toContain('--text-body: #3f443a;');
    expect(tokensCssText).toContain('--accent-primary: #2f6f68;');
    expect(tokensCssText).toContain('--accent-subtle: #e8f2ee;');
});
```

Add reader surface assertions:

```ts
test('keeps reader surface quiet and responsive', () => {
    expect(contentCssText).toMatch(/\.paper\s*\{[^}]*border-radius:\s*8px;[^}]*padding:\s*clamp\(34px,\s*5vw,\s*72px\);/s);
    expect(layoutCssText).toMatch(/\.content-wrapper\s*\{[^}]*padding:\s*clamp\(20px,\s*4vw,\s*52px\);/s);
    expect(toolbarCssText).not.toMatch(/\.toolbar\.expanded\s*\{[^}]*opacity:\s*0\.5;/s);
});
```

- [ ] **Step 2: Run tests to verify they fail before implementation**

Run:

```powershell
cd web
bun test src/layout-css.test.ts
```

Expected: FAIL because new tokens and reader rules are not implemented yet.

- [ ] **Step 3: Update default tokens**

Modify `web/src/style/tokens.css` default `:root` values:

```css
:root {
    --bg-canvas: #f3f1ea;
    --bg-paper: #fffdf6;
    --bg-surface: #fbf8ef;

    --border-light: #d8d1c2;
    --border-focus: #a8c8bf;

    --text-heading: #22251f;
    --text-body: #3f443a;
    --text-muted: #74786c;

    --accent-primary: #2f6f68;
    --accent-subtle: #e8f2ee;
    --accent-border: #a8c8bf;
}
```

Keep existing dark/system/theme overrides after the default block. Do not delete `github`, `one-dark`, `dracula`, or `nord`.

- [ ] **Step 4: Update content layout spacing**

Modify `web/src/style/layout.css` or `web/src/style/content.css` according to where the selectors were moved:

```css
.content-wrapper {
    padding: clamp(20px, 4vw, 52px);
}

.paper {
    border-radius: 8px;
    padding: clamp(34px, 5vw, 72px);
    box-shadow: 0 16px 40px rgba(70, 60, 38, 0.08);
}
```

Keep `width`, `max-width`, `border`, and `margin-bottom` behavior compatible with existing layout controls.

- [ ] **Step 5: Reduce toolbar visual noise**

Modify `web/src/style/toolbar.css`:

```css
.toolbar {
    background: color-mix(in srgb, var(--bg-paper) 92%, transparent);
    border-radius: 8px;
}

.toolbar.expanded {
    opacity: 1;
}

.toolbar.expanded:hover {
    opacity: 1;
}
```

If `color-mix()` causes build or browser concern during verification, use:

```css
.toolbar {
    background: var(--bg-paper);
}
```

The fallback is acceptable because the UI goal is low distraction, not glass effect.

- [ ] **Step 6: Update sidebar and TOC surface styling**

Modify `web/src/style/sidebar.css`:

```css
.sidebar-panel {
    border-radius: 8px;
    background: var(--bg-paper);
    box-shadow: none;
}

.file-tree-row {
    min-height: 32px;
}

.toc-link.active,
.file-tree-row.active,
.file-tree-node.current > .file-tree-row {
    background: var(--accent-subtle);
    color: var(--accent-primary);
}
```

Use the existing active/current class names present in the moved CSS. If the current file selector differs, update the selector to the existing class rather than adding a new class.

- [ ] **Step 7: Replace raw Markdown emoji with SVG icon**

Modify `web/template-main.html`.

Replace:

```html
<a href="{{ .CurrentFilePath }}?q=raw" class="view-raw-btn" target="_blank" title="在新标签页查看原始 Markdown">📄 查看Markdown</a>
```

With:

```html
<a href="{{ .CurrentFilePath }}?q=raw" class="view-raw-btn" target="_blank" title="在新标签页查看原始 Markdown">
    <svg aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round">
        <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
        <polyline points="14 2 14 8 20 8"></polyline>
    </svg>
    <span>查看源码</span>
</a>
```

Add or keep CSS in `web/src/style/content.css`:

```css
.view-raw-btn {
    display: inline-flex;
    align-items: center;
    gap: 7px;
}

.view-raw-btn svg {
    width: 14px;
    height: 14px;
    flex-shrink: 0;
}
```

- [ ] **Step 8: Run visual CSS tests**

Run:

```powershell
cd web
bun test src/layout-css.test.ts
```

Expected: all tests pass.

- [ ] **Step 9: Run frontend build**

Run:

```powershell
cd web
bun run build
```

Expected: build exits 0.

- [ ] **Step 10: Commit visual token refresh**

Run:

```powershell
git status --short
git add web/src/style/tokens.css web/src/style/layout.css web/src/style/toolbar.css web/src/style/sidebar.css web/src/style/content.css web/src/layout-css.test.ts web/template-main.html
git commit -m "style: refresh default reader theme"
```

Expected: commit succeeds. Do not stage unrelated files.

---

### Task 3: Move content search into overlay

**Files:**
- Modify: `web/template.html`
- Modify: `web/src/components/content-search.ts`
- Modify: `web/src/components/content-search.test.ts`
- Modify: `web/src/style/toolbar.css`
- Modify: `web/src/style/overlays.css`
- Modify: `web/src/layout-css.test.ts`

- [ ] **Step 1: Add behavior tests for content search overlay**

Append to `web/src/components/content-search.test.ts`:

```ts
import { setupContentSearch } from './content-search';

describe('setupContentSearch overlay controls', () => {
    test('opens search overlay from trigger and focuses input', () => {
        const dom = new JSDOM(`
            <!DOCTYPE html>
            <button id="content-search-trigger">Search</button>
            <div id="content-search" hidden>
                <button class="content-search-backdrop" type="button"></button>
                <section class="content-search-panel">
                    <input class="content-search-input" />
                    <button class="content-search-clear" type="button"></button>
                    <div class="content-search-results"></div>
                </section>
            </div>
        `, { url: 'http://localhost/' });
        globalThis.document = dom.window.document;
        globalThis.window = dom.window as unknown as Window & typeof globalThis;

        setupContentSearch();

        const wrapper = dom.window.document.getElementById('content-search') as HTMLElement;
        const input = dom.window.document.querySelector('.content-search-input') as HTMLInputElement;
        const trigger = dom.window.document.getElementById('content-search-trigger') as HTMLButtonElement;

        trigger.click();

        expect(wrapper.hidden).toBe(false);
        expect(dom.window.document.activeElement).toBe(input);
    });

    test('closes search overlay on Escape and restores trigger focus', () => {
        const dom = new JSDOM(`
            <!DOCTYPE html>
            <button id="content-search-trigger">Search</button>
            <div id="content-search" hidden>
                <button class="content-search-backdrop" type="button"></button>
                <section class="content-search-panel">
                    <input class="content-search-input" value="layout" />
                    <button class="content-search-clear" type="button"></button>
                    <div class="content-search-results"><div>result</div></div>
                </section>
            </div>
        `, { url: 'http://localhost/' });
        globalThis.document = dom.window.document;
        globalThis.window = dom.window as unknown as Window & typeof globalThis;

        setupContentSearch();

        const wrapper = dom.window.document.getElementById('content-search') as HTMLElement;
        const trigger = dom.window.document.getElementById('content-search-trigger') as HTMLButtonElement;

        trigger.click();
        dom.window.document.dispatchEvent(new dom.window.KeyboardEvent('keydown', { key: 'Escape' }));

        expect(wrapper.hidden).toBe(true);
        expect(dom.window.document.activeElement).toBe(trigger);
    });

    test('opens search overlay with Ctrl+K', () => {
        const dom = new JSDOM(`
            <!DOCTYPE html>
            <button id="content-search-trigger">Search</button>
            <div id="content-search" hidden>
                <button class="content-search-backdrop" type="button"></button>
                <section class="content-search-panel">
                    <input class="content-search-input" />
                    <button class="content-search-clear" type="button"></button>
                    <div class="content-search-results"></div>
                </section>
            </div>
        `, { url: 'http://localhost/' });
        globalThis.document = dom.window.document;
        globalThis.window = dom.window as unknown as Window & typeof globalThis;

        setupContentSearch();

        const wrapper = dom.window.document.getElementById('content-search') as HTMLElement;
        dom.window.document.dispatchEvent(new dom.window.KeyboardEvent('keydown', { key: 'k', ctrlKey: true }));

        expect(wrapper.hidden).toBe(false);
    });
});
```

- [ ] **Step 2: Add CSS tests for toolbar search and overlay**

Add to `web/src/layout-css.test.ts`:

```ts
test('defines content search trigger and centered overlay styles', () => {
    expect(toolbarCssText).toMatch(/\.content-search-trigger\s*\{[^}]*display:\s*inline-flex;/s);
    expect(overlaysCssText).toMatch(/\.content-search-wrapper\[hidden\]\s*\{[^}]*display:\s*none;/s);
    expect(overlaysCssText).toMatch(/\.content-search-panel\s*\{[^}]*position:\s*fixed;[^}]*top:\s*14vh;[^}]*left:\s*50%;[^}]*width:\s*min\(720px,\s*calc\(100vw - 32px\)\);[^}]*transform:\s*translateX\(-50%\);/s);
    expect(overlaysCssText).toMatch(/@media \(max-width:\s*1023px\)\s*\{[\s\S]*\.content-search-panel\s*\{[^}]*top:\s*12px;[^}]*bottom:\s*12px;/s);
});
```

- [ ] **Step 3: Run tests to verify failures before implementation**

Run:

```powershell
cd web
bun test src/components/content-search.test.ts src/layout-css.test.ts
```

Expected: FAIL because trigger, overlay behavior, and overlay CSS are not implemented yet.

- [ ] **Step 4: Update toolbar template with Search icon**

Modify `web/template.html` inside `.toolbar-header`, before `#toolbar-toggle`:

```html
<button class="toolbar-toggle-btn content-search-trigger" id="content-search-trigger" type="button" title="Search content" aria-label="Search content">
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
        <circle cx="11" cy="11" r="7"></circle>
        <path d="m20 20-3.5-3.5"></path>
    </svg>
</button>
```

Do not remove `#toolbar-toggle`; settings panel behavior remains tied to the existing button.

- [ ] **Step 5: Replace current content search DOM with overlay DOM**

Modify the current `#content-search` block in `web/template.html` to:

```html
<div class="content-search-wrapper" id="content-search" hidden>
    <button class="content-search-backdrop" type="button" aria-label="Close search"></button>
    <section class="content-search-panel" role="dialog" aria-modal="true" aria-label="Search content">
        <div class="content-search-box">
            <svg class="content-search-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                <circle cx="11" cy="11" r="7"></circle>
                <path d="m20 20-3.5-3.5"></path>
            </svg>
            <input
                type="text"
                class="content-search-input"
                id="content-search-input"
                placeholder="Search in files..."
                autocomplete="off"
            />
            <kbd class="content-search-shortcut">Esc</kbd>
            <button class="content-search-clear" type="button" aria-label="Clear search">×</button>
        </div>
        <div class="content-search-results"></div>
    </section>
</div>
```

Place this block near the top of `<main class="content-wrapper">` as it is today, or directly after the toolbar. Because the panel is `fixed`, visual position does not depend on DOM location.

- [ ] **Step 6: Implement overlay open/close in content-search.ts**

Modify `web/src/components/content-search.ts` inside `setupContentSearch()`.

Add element lookup:

```ts
const trigger = document.getElementById('content-search-trigger') as HTMLButtonElement | null;
const backdrop = searchWrapper.querySelector('.content-search-backdrop') as HTMLButtonElement | null;
```

Add helpers near existing `closeResults()` use:

```ts
function openSearch() {
    searchWrapper.hidden = false;
    trigger?.setAttribute('aria-expanded', 'true');
    requestAnimationFrame(() => input.focus());
}

function closeSearch() {
    closeResults(resultsContainer, input);
    searchWrapper.hidden = true;
    trigger?.setAttribute('aria-expanded', 'false');
    trigger?.focus();
}
```

Register events:

```ts
trigger?.setAttribute('aria-controls', 'content-search');
trigger?.setAttribute('aria-expanded', 'false');
trigger?.addEventListener('click', openSearch);
backdrop?.addEventListener('click', closeSearch);

document.addEventListener('keydown', (e: KeyboardEvent) => {
    const key = e.key.toLowerCase();
    if ((e.ctrlKey || e.metaKey) && key === 'k') {
        e.preventDefault();
        openSearch();
        return;
    }
    if (e.key === 'Escape' && !searchWrapper.hidden) {
        e.preventDefault();
        closeSearch();
    }
});
```

Update existing input Escape handler to call `closeSearch()` instead of only `closeResults(resultsContainer, input)`.

Keep existing `input` listener, clear button listener, `performSearch()`, and result click navigation.

- [ ] **Step 7: Replace search result emoji icon**

Modify `renderResults()` in `web/src/components/content-search.ts`.

Replace:

```ts
<span class="file-icon">📄</span>
```

With:

```ts
<span class="file-icon" aria-hidden="true">
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round">
        <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
        <polyline points="14 2 14 8 20 8"></polyline>
    </svg>
</span>
```

Add CSS in `web/src/style/overlays.css`:

```css
.file-icon,
.file-icon svg {
    width: 16px;
    height: 16px;
    display: inline-flex;
    flex-shrink: 0;
}
```

- [ ] **Step 8: Add overlay CSS**

Modify `web/src/style/toolbar.css`:

```css
.content-search-trigger {
    display: inline-flex;
}

.content-search-trigger[aria-expanded="true"] {
    background: var(--accent-subtle);
    color: var(--accent-primary);
}
```

Modify `web/src/style/overlays.css`:

```css
.content-search-wrapper[hidden] {
    display: none;
}

.content-search-wrapper {
    position: fixed;
    inset: 0;
    z-index: 200;
}

.content-search-backdrop {
    position: fixed;
    inset: 0;
    border: 0;
    background: rgba(47, 49, 40, 0.18);
    backdrop-filter: blur(2px);
}

.content-search-panel {
    position: fixed;
    top: 14vh;
    left: 50%;
    width: min(720px, calc(100vw - 32px));
    transform: translateX(-50%);
    overflow: hidden;
    border: 1px solid var(--border-light);
    border-radius: 10px;
    background: var(--bg-paper);
    box-shadow: 0 24px 70px rgba(70, 60, 38, 0.22);
}

.content-search-box {
    width: 100%;
    height: 56px;
    display: grid;
    grid-template-columns: 24px minmax(0, 1fr) auto 28px;
    align-items: center;
    gap: 10px;
    padding: 0 16px;
    border-bottom: 1px solid var(--border-light);
}

.content-search-input {
    width: 100%;
    height: auto;
    padding: 0;
    border: 0;
    background: transparent;
    font-size: 17px;
}

.content-search-input:focus {
    box-shadow: none;
}

.content-search-results {
    position: static;
    width: 100%;
    min-width: 0;
    max-width: none;
    max-height: min(560px, 62vh);
    border: 0;
    border-radius: 0;
    box-shadow: none;
}

@media (max-width: 1023px) {
    .content-search-panel {
        top: 12px;
        bottom: 12px;
        width: calc(100vw - 24px);
        display: flex;
        flex-direction: column;
    }

    .content-search-results {
        max-height: none;
        flex: 1;
    }
}
```

Remove or override old `.content-search-wrapper { position: absolute; top: 6px; left: 30px; }` and old focus-width behavior. Search overlay width is fixed by `.content-search-panel`, not by input focus.

- [ ] **Step 9: Run content search and CSS tests**

Run:

```powershell
cd web
bun test src/components/content-search.test.ts src/layout-css.test.ts
```

Expected: all tests pass.

- [ ] **Step 10: Run frontend build**

Run:

```powershell
cd web
bun run build
```

Expected: build exits 0.

- [ ] **Step 11: Commit search overlay**

Run:

```powershell
git status --short
git add web/template.html web/src/components/content-search.ts web/src/components/content-search.test.ts web/src/style/toolbar.css web/src/style/overlays.css web/src/layout-css.test.ts
git commit -m "feat: move content search to overlay"
```

Expected: commit succeeds. Do not stage unrelated files.

---

### Task 4: Final verification and plan updates

**Files:**
- Modify: `docs/superpowers/plans/2026-06-22-markview-ui-refresh.md`

- [ ] **Step 1: Run all frontend tests**

Run:

```powershell
cd web
bun test
```

Expected: all frontend tests pass.

- [ ] **Step 2: Run frontend build**

Run:

```powershell
cd web
bun run build
```

Expected: build exits 0.

- [ ] **Step 3: Run Go tests**

Run:

```powershell
go test ./...
```

Expected: all Go packages pass.

- [ ] **Step 4: Manually inspect key UI states**

Run built app according to project convention:

```powershell
go build -o markview-ui-check.exe .
.\markview-ui-check.exe example --no-browser
```

Open the printed local URL in a browser and inspect:

```text
1440px desktop: default layout, Search icon, Settings icon, search overlay
1024px desktop: TOC right, preview panel open
390px mobile: document reading, toolbar buttons, search overlay
light/system/dark color schemes
sidebar expanded/collapsed
```

Stop the process after inspection.

- [ ] **Step 5: Update plan checkboxes**

Modify this file and mark completed steps with `[x]`. Do not mark unrun validation steps complete.

- [ ] **Step 6: Commit plan progress**

Run:

```powershell
git status --short
git add docs/superpowers/plans/2026-06-22-markview-ui-refresh.md
git commit -m "docs: update ui refresh plan progress"
```

Expected: commit succeeds if the plan file changed.

- [ ] **Step 7: Push completed work**

Run:

```powershell
git pull --rebase
git push
git status --short --branch
```

Expected: branch is up to date with origin. If rebase encounters unrelated user changes, stop and ask before resolving.

---

## Self-Review

Spec coverage:

- 护眼默认配色由 Task 2 覆盖。
- 设置面板保留和 Search icon 独立入口由 Task 3 覆盖。
- 内容搜索 overlay、`Ctrl/Cmd+K`、`Esc`、焦点回收由 Task 3 覆盖。
- CSS 拆分由 Task 1 覆盖。
- raw Markdown emoji 替换和搜索结果 emoji 替换由 Task 2、Task 3 覆盖。
- 全量测试、构建、Go 主链路和人工视觉检查由 Task 4 覆盖。

Scope guard:

- 本计划不引入新依赖。
- 本计划不重写搜索 API、SSE、inline navigation、preview resize、sidebar collapse 或主题偏好存储。
- 本计划不把键盘上下选择搜索结果放入本轮最低范围。
