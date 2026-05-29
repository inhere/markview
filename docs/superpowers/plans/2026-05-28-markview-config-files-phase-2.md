# MarkView 配置文件支持二期 Implementation Plan

相关文档：

- [TODO 需求](../../TODO.md#新增支持全局和项目级别的配置文件-)
- [设计文档](../specs/2026-05-28-markview-config-files-design.md)
- [一期实施计划](2026-05-28-markview-config-files-phase-1.md)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在一期配置注入和 layout dataset 基础上，实现设置面板 layout 控件，以及 `compact`、`toc-middle`、`toc-right` 三种完整布局模式；其中 `toc-right` 使用右侧浮动 TOC，并支持预览面板打开时默认隐藏、手动展开跳转。

**Architecture:** 保持单一 Files DOM、单一 TOC DOM、单一正文 DOM，不复制目录树或 TOC。`web/template.html` 将页面主区域拆成 `.app-shell` 下的 `.files-pane`、`.toc-pane`、`.content-wrapper` 三个同级 pane；`web/src/style/app.css` 使用 CSS Grid 和 `html[data-layout]` 切换 `compact`、`toc-middle`、`toc-right`，其中 `toc-right` 不给 TOC 预留 grid 列，而是将 `.toc-pane` 作为右侧浮动面板。`web/src/preferences.ts` 负责 layout 偏好读写和清除；`web/src/layout-mode.ts` 负责设置面板控件状态和页面级 dataset；`web/src/toc-toggle.ts` 负责浮动 TOC 显示/隐藏和可访问状态。

**Tech Stack:** TypeScript、Bun、JSDOM、CSS Grid/Flex、Go embed 现有前端构建链路。

---

## 复审结论

一期已经完成配置文件读取、合并、页面注入、`preview_exts` 生效和 layout 基础链路。二期只做 UI/布局，不再扩大 Go 配置模型范围。

关键约束：

- 保持 `compact` 现有体验，避免默认用户打开页面后视觉突变。
- `toc-middle` 只在桌面宽度启用三栏；`toc-right` 在桌面宽度使用 files + content 主布局和右侧浮动 TOC；移动端视觉回退为 compact，不清除用户保存的 layout 偏好。
- 设置面板修改 layout 后立即更新 `document.documentElement.dataset.layout` 并持久化。
- Reset/Default 清除本地 layout 覆盖，恢复服务端注入的项目默认 layout。
- 不复制 `#toc-list`，确保 `generateTOC()`、`highlightTOC()` 仍只维护一个 TOC。
- 右侧 preview panel 打开时避免 Files + Body + TOC + Preview 四区拥挤；`toc-right` 默认隐藏浮动 TOC，但保留控制按钮，允许用户临时打开 TOC 点击跳转。

## 文件结构

- Modify: `web/template.html`
  - 设置面板新增 Layout 分段控件和 Default 按钮。
  - 新增 `.app-shell` 主布局容器。
  - 将现有 sidebar 拆为 `.files-pane.sidebar`，保留 header、Files panel、collapsed icons 和 resize handle。
  - 将现有 TOC section 移出 `.sidebar-panels`，成为 `.app-shell` 直接子项 `.toc-pane.sidebar-panel.sidebar-panel-toc#toc-panel`。
  - 新增 `toc-right` 浮动 TOC 控制按钮，例如 `#toc-toggle-button`。
  - 给正文内部 wrapper 去掉 inline style，改用 class（例如 `content-inner`）。
- Modify: `web/src/preferences.ts`
  - 增加清除 layout 偏好的 helper，例如 `clearStoredLayoutMode()`。
  - 复用一期 `normalizeLayoutMode`、`readStoredLayoutMode`、`resolveLayoutMode`、`persistLayoutMode`。
- Modify: `web/src/preferences.test.ts`
  - 覆盖清除 layout 偏好和无效值不覆盖项目默认。
- Modify: `web/src/app.ts`
  - `setupToolbar()` 接入 layout 控件。
  - 初始化控件 active/aria 状态。
  - 切换 layout 时持久化并更新 dataset。
  - Default 按钮清除本地覆盖并恢复 `appConfig.layout`。
- Add: `web/src/toc-toggle.ts`
  - 初始化 `toc-right` 浮动 TOC 开关。
  - 监听 preview 打开状态或 link preview 事件，preview 打开时默认隐藏 TOC。
  - 同步 `aria-expanded`、`hidden`/class 状态和 `body.toc-floating-open`。
- Add: `web/src/toc-toggle.test.ts`
  - 覆盖按钮显示/隐藏、preview 打开默认隐藏、手动打开后可保持 TOC 可见。
- Modify: `web/src/style/app.css`
  - 新增 layout 控件样式。
  - 将 `body` 固定 sidebar + content margin 模型改为 `.app-shell` CSS Grid。
  - 实现 `compact`、`toc-middle`、`toc-right`，其中 `toc-right` 使用浮动 TOC。
  - 处理 sidebar collapsed、preview-active、content-search、移动端回退。
- Modify: `web/src/sidebar.ts`
  - 如有 selector 假设依赖旧 sidebar/TOC 嵌套关系，调整为只控制 files pane 折叠。
- Modify: `web/src/sidebar-resize.ts`
  - 继续控制 `.files-pane.sidebar` 对应的 `--sidebar-width`，不负责 TOC 宽度。
- Add: `web/src/layout-smoke.test.ts` 或等价浏览器 smoke 脚本
  - 运行本地构建产物后，用真实浏览器/Playwright 检查三种 layout 的 bbox 顺序、浮动状态和无重叠。
- Add/Modify tests as needed:
  - `web/src/preferences.test.ts`
  - 如当前 `app.ts` 仍难以直接测试 toolbar 初始化，可新增轻量模块 `web/src/layout-mode.ts` 和 `web/src/layout-mode.test.ts` 承载纯 DOM 控件逻辑，再由 `app.ts` 调用。

## Task 1: Layout 偏好清除与控件状态基础

**Files:**
- Modify: `web/src/preferences.ts`
- Modify: `web/src/preferences.test.ts`

- [x] **Step 1: 写失败测试**

在 `web/src/preferences.test.ts` 增加：

```ts
import {
    clearStoredLayoutMode,
    LAYOUT_MODE_STORAGE_KEY,
    readStoredLayoutMode,
    resolveLayoutMode,
} from './preferences';

test('clearStoredLayoutMode removes local layout override', () => {
    const storage = new Map<string, string>([
        [LAYOUT_MODE_STORAGE_KEY, 'toc-right'],
    ]);

    clearStoredLayoutMode({
        removeItem(key: string) {
            storage.delete(key);
        },
    });

    expect(storage.has(LAYOUT_MODE_STORAGE_KEY)).toBe(false);
});

test('invalid stored layout does not override configured layout', () => {
    const stored = readStoredLayoutMode({
        getItem() {
            return 'wide';
        },
    });

    expect(resolveLayoutMode(stored, 'toc-right')).toBe('toc-right');
});
```

- [x] **Step 2: 运行测试确认失败**

Run:

```bash
cd web && bun test src/preferences.test.ts
```

Expected: FAIL，`clearStoredLayoutMode` 未定义。

- [x] **Step 3: 实现清除 helper**

在 `web/src/preferences.ts` 增加：

```ts
interface StorageRemover {
    removeItem(key: string): void;
}

export function clearStoredLayoutMode(storage: StorageRemover = window.localStorage) {
    try {
        storage.removeItem(LAYOUT_MODE_STORAGE_KEY);
    } catch {}
}
```

- [x] **Step 4: 运行测试确认通过**

Run:

```bash
cd web && bun test src/preferences.test.ts
```

Expected: PASS。

- [x] **Step 5: 提交**

```bash
git add web/src/preferences.ts web/src/preferences.test.ts
git commit -m "feat(web): add layout preference reset"
```

## Task 2: 设置面板 Layout 控件

**Files:**
- Modify: `web/template.html`
- Modify: `web/src/app.ts`
- Modify: `web/src/preferences.ts`
- Test: `web/src/preferences.test.ts` 或新增 `web/src/layout-mode.test.ts`

- [x] **Step 1: 写失败测试**

优先新增 `web/src/layout-mode.ts` 和 `web/src/layout-mode.test.ts`，把控件状态同步逻辑从大型 `app.ts` 中拆出，便于测试：

```ts
import { describe, expect, test } from 'bun:test';
import { JSDOM } from 'jsdom';
import { syncLayoutControls } from './layout-mode';

test('syncLayoutControls marks selected layout button and enables default when local override exists', () => {
    const dom = new JSDOM(`<!doctype html><body>
        <button data-layout-mode="compact"></button>
        <button data-layout-mode="toc-middle"></button>
        <button data-layout-mode="toc-right"></button>
        <button id="layout-default"></button>
    </body>`);

    syncLayoutControls(dom.window.document, 'toc-right', true);

    expect(dom.window.document.querySelector('[data-layout-mode="toc-right"]')?.classList.contains('active')).toBe(true);
    expect(dom.window.document.querySelector('[data-layout-mode="toc-right"]')?.getAttribute('aria-pressed')).toBe('true');
    expect(dom.window.document.getElementById('layout-default')?.hasAttribute('disabled')).toBe(false);
});

test('syncLayoutControls disables default when using project default', () => {
    const dom = new JSDOM(`<!doctype html><body>
        <button data-layout-mode="compact"></button>
        <button data-layout-mode="toc-middle"></button>
        <button data-layout-mode="toc-right"></button>
        <button id="layout-default"></button>
    </body>`);

    syncLayoutControls(dom.window.document, 'toc-right', false);

    expect(dom.window.document.getElementById('layout-default')?.hasAttribute('disabled')).toBe(true);
    expect(dom.window.document.getElementById('layout-default')?.classList.contains('active')).toBe(true);
});
```

再增加切换行为测试：

```ts
test('setupLayoutControls persists selected layout and clears override on default', () => {
    // 使用 JSDOM 构造 toolbar buttons、fake storage、fake documentElement。
    // 点击 data-layout-mode="toc-middle" 后：
    // - html dataset.layout === 'toc-middle'
    // - storage[markview:layout-mode] === 'toc-middle'
    // 点击 #layout-default 后：
    // - html dataset.layout === appConfig layout
    // - storage 删除 markview:layout-mode
});
```

- [x] **Step 2: 运行测试确认失败**

Run:

```bash
cd web && bun test src/layout-mode.test.ts src/preferences.test.ts
```

Expected: FAIL，`layout-mode` 模块或相关函数不存在。

- [x] **Step 3: 模板新增 Layout 控件**

在 `web/template.html` 的 `.toolbar-content` 中新增一行，建议放在 Width 行之前：

```html
<div class="toolbar-row">
    <div class="toolbar-group toolbar-layout-group" role="group" aria-label="Layout">
        <span class="toolbar-label">Layout</span>
        <button class="toolbar-btn" data-layout-mode="compact" aria-pressed="false" title="Compact layout">Compact</button>
        <button class="toolbar-btn" data-layout-mode="toc-middle" aria-pressed="false" title="Files, TOC, content">TOC Mid</button>
        <button class="toolbar-btn" data-layout-mode="toc-right" aria-pressed="false" title="Files, content, TOC">TOC Right</button>
        <button class="toolbar-btn" id="layout-default" title="Use project default layout">Default</button>
    </div>
</div>
```

说明：

- 继续使用现有 `.toolbar-btn` 风格。
- 文案保持短，避免 toolbar 过宽。
- `Default` 的语义是清除本地 override，不是强制 compact。

- [x] **Step 4: 实现 layout 控件模块**

新增 `web/src/layout-mode.ts`：

```ts
import type { AppLayout } from './app-config';
import {
    clearStoredLayoutMode,
    LAYOUT_MODE_STORAGE_KEY,
    persistLayoutMode,
} from './preferences';

export function applyLayoutMode(mode: AppLayout, documentRef: Document = document) {
    documentRef.documentElement.dataset.layout = mode;
}

export function hasStoredLayoutMode(storage: Storage = window.localStorage): boolean {
    try {
        return storage.getItem(LAYOUT_MODE_STORAGE_KEY) !== null;
    } catch {
        return false;
    }
}

export function syncLayoutControls(documentRef: Document, mode: AppLayout, hasOverride: boolean) {
    documentRef.querySelectorAll('[data-layout-mode]').forEach(button => {
        const selected = (button as HTMLElement).dataset.layoutMode === mode;
        button.classList.toggle('active', selected);
        button.setAttribute('aria-pressed', String(selected));
    });
    const defaultButton = documentRef.getElementById('layout-default') as HTMLButtonElement | null;
    if (defaultButton) {
        defaultButton.disabled = !hasOverride;
        defaultButton.classList.toggle('active', !hasOverride);
    }
}
```

再实现 `setupLayoutControls(options)`，明确签名：

```ts
interface LayoutControlOptions {
    documentRef?: Document;
    storage?: Storage;
    configuredLayout: AppLayout;
    initialLayout: AppLayout;
}

export function setupLayoutControls({
    documentRef = document,
    storage = window.localStorage,
    configuredLayout,
    initialLayout,
}: LayoutControlOptions) {
    // implementation
}
```

行为：

- 初始化 `syncLayoutControls`。
- 点击 `data-layout-mode`：
  - normalize mode。
  - `applyLayoutMode(mode, documentRef)`。
  - `persistLayoutMode(mode, storage)`。
  - `syncLayoutControls(documentRef, mode, true)`。
- 点击 `#layout-default`：
  - `clearStoredLayoutMode(storage)`。
  - `applyLayoutMode(configuredLayout, documentRef)`。
  - `syncLayoutControls(documentRef, configuredLayout, false)`。

- [x] **Step 5: app.ts 接入**

在 `web/src/app.ts`：

- 从 `layout-mode.ts` 导入 `applyLayoutMode`、`setupLayoutControls`。
- 删除本文件内已有的 `applyLayoutMode` 私有函数。
- `setupOnce()` 保留现有 layout resolution：

```ts
const appConfig = readAppConfig();
const initialLayout = resolveLayoutMode(readStoredLayoutMode(), appConfig.layout);
applyLayoutMode(initialLayout);
configureLinkPreview({ previewExts: appConfig.previewExts });
```

- 调整 `setupToolbar()` 签名，让它接收 layout 相关配置：

```ts
function setupToolbar(appConfig: AppConfig, initialLayout: AppLayout) {
    // existing setup...
    setupLayoutControls({
        documentRef: document,
        configuredLayout: appConfig.layout,
        initialLayout,
    });
}
```

注意：局部页面导航不重新应用 layout。`toc-right` 浮动 TOC 控制在 Task 5 单独接入，不放进本任务提交。

- [x] **Step 6: 运行前端测试**

Run:

```bash
cd web && bun test
```

Expected: PASS。

- [x] **Step 7: 提交**

```bash
git add web/template.html web/src/app.ts web/src/layout-mode.ts web/src/layout-mode.test.ts web/src/preferences.ts web/src/preferences.test.ts
git commit -m "feat(web): add layout selector controls"
```

## Task 3: 拆分 Files/TOC/Content 为同级 panes

**Files:**
- Modify: `web/template.html`
- Modify: `web/src/style/app.css`
- Modify: `web/src/sidebar.ts`
- Modify: `web/src/sidebar-resize.ts`
- Test: `internal/handlers/handlers_test.go` 或现有模板渲染测试

- [x] **Step 1: 写/更新失败测试**

新增 Go handler/template 测试，确认完整页面包含新的布局骨架，且 TOC 不再是 `.sidebar-panels` 的后代。测试文件优先放在 `internal/handlers/handlers_test.go`，复用现有完整页面渲染 helper。

```go
assert.StrContains(t, body, `class="app-shell"`)
assert.StrContains(t, body, `class="files-pane sidebar"`)
assert.StrContains(t, body, `class="toc-pane sidebar-panel sidebar-panel-toc" id="toc-panel"`)
assert.StrContains(t, body, `id="toc-toggle-button"`)
assert.StrContains(t, body, `aria-controls="toc-panel"`)
assert.StrContains(t, body, `class="content-inner"`)
assert.StrContains(t, body, `data-layout-mode="compact"`)
assert.True(t, strings.Index(body, `id="files-panel"`) < strings.Index(body, `id="toc-panel"`))
assert.True(t, strings.Index(body, `id="toc-panel"`) < strings.Index(body, `class="content-wrapper"`))
```

如果测试中要判断嵌套关系，优先用 Go HTML tokenizer 或 `goquery` 等结构化方式；没有现成依赖时，用明确的片段断言即可，不为测试引入重量级依赖。

- [x] **Step 2: 运行测试确认失败**

Run:

```bash
go test ./internal/handlers
```

Expected: FAIL，模板还没有 `.app-shell`、`.files-pane`、`.toc-pane` 或 TOC 仍在旧 sidebar 内。

- [x] **Step 3: 调整模板 DOM**

在 `web/template.html`：

- 用 `.app-shell` 包住主阅读区域。
- 将原 `<aside class="sidebar">` 改为 `<aside class="files-pane sidebar" id="files-pane">`，内部保留 `.sidebar-header`、`.sidebar-panels`、Files panel、collapsed icons、resize handle。
- 从 `.sidebar-panels` 中移出 TOC section，放到 `.app-shell` 下，和 files pane、content wrapper 同级。
- 保持 `#toc-list` 只有一份，不新增第二份 TOC。
- 在 `.toc-pane` 前后合适位置新增一个浮动开关按钮，默认只在 `toc-right` 桌面模式可见：

```html
<button class="toc-toggle-button" id="toc-toggle-button" type="button" aria-controls="toc-panel" aria-expanded="true" title="Toggle table of contents">
    TOC
</button>
```

- 将正文 inline wrapper 改为 `.content-inner`。

```html
<div class="app-shell">
    <aside class="files-pane sidebar" id="files-pane">
        <!-- existing sidebar header, files panel, collapsed icons, resize handle -->
    </aside>

    <aside class="toc-pane sidebar-panel sidebar-panel-toc" id="toc-panel">
        <!-- existing On This Page header and #toc-list -->
    </aside>

    <button class="toc-toggle-button" id="toc-toggle-button" type="button" aria-controls="toc-panel" aria-expanded="true">
        TOC
    </button>

    <main class="content-wrapper">
        <div class="content-inner">
            {{ .MainContent }}
        </div>
    </main>
</div>
```

- [x] **Step 4: 增加 pane 基础 CSS**

在 `web/src/style/app.css` 增加 `app-shell`、pane 和正文内部 wrapper 的基础规则。此时只做到 compact 视觉兼容，不实现完整布局切换：

```css
:root {
    --toc-width: 240px;
    --toc-min-width: 180px;
    --toc-max-width: 360px;
}

body {
    display: block;
}

.app-shell {
    min-height: 100vh;
    height: 100vh;
    display: grid;
    grid-template-columns: var(--sidebar-width) minmax(0, 1fr);
    grid-template-rows: minmax(0, 1fr) minmax(12rem, 32vh);
    grid-template-areas:
        "files content"
        "toc content";
}

.files-pane {
    grid-area: files;
}

.toc-pane {
    grid-area: toc;
}

.content-wrapper {
    grid-area: content;
    margin-left: 0;
    min-height: 0;
    overflow: auto;
}

.content-inner {
    width: 100%;
    max-width: var(--layout-max-width);
    display: flex;
    flex-direction: column;
    align-items: center;
}
```

实现时必须阅读并调整现有 `.sidebar`、`.sidebar-panels`、`.content-wrapper`、`.preview-panel`、`.content-search-wrapper` 相关规则，删除或覆盖旧的 fixed sidebar 与 `margin-left` 依赖。`compact` 下视觉目标仍然是：

- Files 和 TOC 都在左侧区域，左列总高度不超过视口高度。
- 正文在右侧，宽度控制仍由 `--layout-max-width` 生效。
- sidebar resize 继续改变左侧 files/compact sidebar 宽度。
- TOC 不是 `.sidebar-panels` 后代。

- [x] **Step 5: 调整 sidebar collapse 与 resize selector**

检查 `web/src/sidebar.ts`：

- collapse 按钮继续切换 `body.sidebar-collapsed` 和 `.files-pane.sidebar-collapsed`。
- `#files-panel.files-collapsed` 行为不变。
- 不再依赖 TOC 是 `.sidebar-panels` 子节点。

检查 `web/src/sidebar-resize.ts`：

- 继续选择 `.files-pane.sidebar` 或兼容 `.sidebar`。
- 只写 `--sidebar-width`。
- 不增加 TOC resize。

- [x] **Step 6: 运行测试**

Run:

```bash
go test ./internal/handlers
cd web && bun test
```

Expected: PASS。

- [x] **Step 7: 提交**

```bash
git add web/template.html web/src/style/app.css web/src/sidebar.ts web/src/sidebar-resize.ts internal/handlers/handlers_test.go
git commit -m "feat(web): split layout panes"
```

Implementation note: `.content-wrapper` 成为桌面阅读滚动容器后，同步修复了 hash 跳转、TOC 高亮和 inline refresh 滚动恢复，移动端仍回退到 window 滚动。

## Task 4: 桌面布局 CSS 与 toc-right 浮动面板

**Files:**
- Modify: `web/src/style/app.css`
- Modify/Add: `web/src/layout-css.test.ts`

- [x] **Step 1: 写布局断言测试**

优先新增轻量 CSS 文本断言测试 `web/src/layout-css.test.ts`，避免引入浏览器依赖：

```ts
import { describe, expect, test } from 'bun:test';
import cssText from './style/app.css' with { type: 'text' };

describe('layout CSS modes', () => {
    test('defines desktop rules for toc-middle and floating toc-right', () => {
        expect(cssText).toContain('.app-shell');
        expect(cssText).toContain('grid-template-areas');
        expect(cssText).toContain('[data-layout="toc-middle"]');
        expect(cssText).toContain('[data-layout="toc-right"]');
        expect(cssText).toContain('--toc-width');
        expect(cssText).toContain('"files toc content"');
        expect(cssText).toContain('"files content"');
        expect(cssText).toContain('.toc-toggle-button');
        expect(cssText).toContain('body.toc-floating-open .toc-pane');
    });

    test('keeps mobile layout compact', () => {
        expect(cssText).toContain('@media (max-width: 1023px)');
        expect(cssText).toContain('grid-template-columns: minmax(0, 1fr)');
    });
});
```

如果 Bun 对 CSS text import 支持不稳定，可以改为在测试里 `Bun.file(new URL('./style/app.css', import.meta.url)).text()`。

- [x] **Step 2: 运行测试确认失败**

Run:

```bash
cd web && bun test src/layout-css.test.ts
```

Expected: FAIL，CSS 还没有完整桌面布局规则。

- [x] **Step 3: 实现桌面布局规则**

采用独立 TOC pane + `.app-shell` CSS Grid 策略。不要复制 TOC DOM、不要用 JS 在 compact、toc-middle 和 toc-right 之间移动 TOC。DOM 在 Task 3 已经保证 `.files-pane`、`.toc-pane`、`.content-wrapper` 是 `.app-shell` 的直接子项。`compact` 和 `toc-middle` 通过 grid areas 调整视觉位置；`toc-right` 不给 TOC 预留 grid 列，而是用右侧浮动 TOC 覆盖在正文区域上方。

在 `web/src/style/app.css` 中新增桌面规则（建议放在 `.app-shell` 相关样式附近）：

```css
@media (min-width: 1024px) {
    html[data-layout="compact"] .app-shell {
        grid-template-columns: var(--sidebar-width) minmax(0, 1fr);
        grid-template-rows: minmax(0, 1fr) minmax(12rem, 32vh);
        grid-template-areas:
            "files content"
            "toc content";
    }

    html[data-layout="toc-middle"] .app-shell {
        grid-template-columns: var(--sidebar-width) var(--toc-width) minmax(0, 1fr);
        grid-template-rows: minmax(0, 1fr);
        grid-template-areas: "files toc content";
    }

    html[data-layout="toc-right"] .app-shell {
        grid-template-columns: var(--sidebar-width) minmax(0, 1fr);
        grid-template-rows: minmax(0, 1fr);
        grid-template-areas: "files content";
    }

    .files-pane {
        width: var(--sidebar-width);
        min-height: 0;
        overflow: auto;
    }

    .toc-pane {
        width: var(--toc-width);
        min-width: var(--toc-min-width);
        max-width: var(--toc-max-width);
        height: 100vh;
        border-radius: 0;
        overflow: auto;
    }

    html[data-layout="compact"] .toc-pane {
        width: var(--sidebar-width);
        height: auto;
        min-height: 0;
    }

    html[data-layout="toc-middle"] .files-pane,
    html[data-layout="toc-right"] .files-pane,
    html[data-layout="toc-middle"] .toc-pane {
        height: 100vh;
    }

    html[data-layout="toc-right"] .toc-pane {
        position: fixed;
        top: 64px;
        right: 16px;
        bottom: 16px;
        z-index: 35;
        height: auto;
        max-height: calc(100vh - 80px);
        box-shadow: var(--shadow-lg);
        transform: translateX(calc(100% + 24px));
        opacity: 0;
        pointer-events: none;
    }

    html[data-layout="toc-right"] body.toc-floating-open .toc-pane {
        transform: translateX(0);
        opacity: 1;
        pointer-events: auto;
    }

    .toc-toggle-button {
        display: none;
    }

    html[data-layout="toc-right"] .toc-toggle-button {
        display: inline-flex;
        position: fixed;
        top: 16px;
        right: 16px;
        z-index: 36;
    }

    .content-wrapper {
        margin-left: 0;
        margin-right: 0;
        min-width: 0;
    }
}
```

同时需要删除或覆盖旧布局中的核心假设：

- `body { display: flex; flex-direction: row; }`。
- `.sidebar { position: fixed; left: 0; }`。
- `.content-wrapper { margin-left: var(--sidebar-width); }`。
- `body.sidebar-collapsed .content-wrapper { margin-left: var(--sidebar-collapsed-width); }`。

如果为了降低改动范围需要短期保留旧规则，必须用 `.app-shell` 新规则在桌面和移动端都明确覆盖，避免旧 fixed sidebar 与 grid 同时生效。`toc-right` 的 fixed 浮动 TOC 是有意设计，只用于 `.toc-pane` overlay，不用于主页面布局。

- [x] **Step 4: 处理 sidebar collapsed**

规则：

- `compact`: 沿用当前行为。
- `toc-middle`: collapse 只影响 files pane 宽度，TOC 继续可见。
- `toc-right`: collapse 只影响 files pane 宽度；浮动 TOC 是否可见由 `.toc-toggle-button` 控制。

CSS 需要覆盖 `.app-shell` 的列宽，而不是隐藏 `.sidebar-panels`：

```css
@media (min-width: 1024px) {
    body.sidebar-collapsed .files-pane {
        width: var(--sidebar-collapsed-width);
    }

    html[data-layout="compact"] body.sidebar-collapsed .app-shell {
        grid-template-columns: var(--sidebar-collapsed-width) minmax(0, 1fr);
    }

    html[data-layout="toc-middle"] body.sidebar-collapsed .app-shell {
        grid-template-columns: var(--sidebar-collapsed-width) var(--toc-width) minmax(0, 1fr);
    }

    html[data-layout="toc-right"] body.sidebar-collapsed .app-shell {
        grid-template-columns: var(--sidebar-collapsed-width) minmax(0, 1fr);
    }

    body.sidebar-collapsed .files-pane .sidebar-panels {
        display: none;
    }
}
```

因为 TOC 已不在 `.sidebar-panels` 内，旧的 `.sidebar-collapsed .sidebar-panels { display: none; }` 不应再影响 TOC。实现后必须确认 `.sidebar-icons` 在 `toc-middle` 和 `toc-right` collapse 后仍可点击恢复 files。

同时在 `web/src/layout-css.test.ts` 中增加断言，至少覆盖：

```ts
expect(cssText).toContain('body.sidebar-collapsed .files-pane');
expect(cssText).toContain('body.sidebar-collapsed .app-shell');
```

- [x] **Step 5: 运行前端测试**

Run:

```bash
cd web && bun test
```

Expected: PASS。

- [x] **Step 6: 提交**

```bash
git add web/src/style/app.css web/src/layout-css.test.ts
git commit -m "feat(web): add desktop toc layouts"
```

## Task 5: toc-right 浮动 TOC 开关、Preview panel、搜索框和移动端适配

**Files:**
- Add: `web/src/toc-toggle.ts`
- Add: `web/src/toc-toggle.test.ts`
- Modify: `web/src/app.ts`
- Modify: `web/src/style/app.css`
- Modify/Add: `web/src/layout-css.test.ts`

- [x] **Step 1: 写浮动 TOC 状态测试**

新增 `web/src/toc-toggle.test.ts`：

```ts
import { describe, expect, test } from 'bun:test';
import { JSDOM } from 'jsdom';
import { setupTocToggle } from './toc-toggle';

function createDom(layout = 'toc-right') {
    const dom = new JSDOM(`<!doctype html>
        <html data-layout="${layout}">
            <body>
                <button id="toc-toggle-button" aria-controls="toc-panel" aria-expanded="true"></button>
                <aside id="toc-panel" class="toc-pane"></aside>
            </body>
        </html>`);
    return dom;
}

test('toc-right toggle opens and closes floating toc', () => {
    const dom = createDom();
    setupTocToggle({ documentRef: dom.window.document });

    const button = dom.window.document.getElementById('toc-toggle-button') as HTMLButtonElement;
    expect(dom.window.document.body.classList.contains('toc-floating-open')).toBe(true);
    expect(button.getAttribute('aria-expanded')).toBe('true');

    button.click();
    expect(dom.window.document.body.classList.contains('toc-floating-open')).toBe(false);
    expect(button.getAttribute('aria-expanded')).toBe('false');

    button.click();
    expect(dom.window.document.body.classList.contains('toc-floating-open')).toBe(true);
    expect(button.getAttribute('aria-expanded')).toBe('true');
});

test('preview active hides floating toc by default but manual toggle can reopen it', () => {
    const dom = createDom();
    setupTocToggle({ documentRef: dom.window.document });

    dom.window.document.body.classList.add('preview-active');
    dom.window.document.dispatchEvent(new dom.window.CustomEvent('markview:preview-state-changed'));

    const button = dom.window.document.getElementById('toc-toggle-button') as HTMLButtonElement;
    expect(dom.window.document.body.classList.contains('toc-floating-open')).toBe(false);
    expect(button.getAttribute('aria-expanded')).toBe('false');

    button.click();
    expect(dom.window.document.body.classList.contains('toc-floating-open')).toBe(true);
    expect(button.getAttribute('aria-expanded')).toBe('true');
});
```

事件名可以按实现调整，但必须有一条可测试路径能在 preview 打开时触发 TOC 默认隐藏。若现有 link preview 模块没有事件，可在切换 `body.preview-active` 的地方派发一个轻量自定义事件。

- [x] **Step 2: 运行测试确认失败**

Run:

```bash
cd web && bun test src/toc-toggle.test.ts
```

Expected: FAIL，`toc-toggle.ts` 不存在。

- [x] **Step 3: 实现 `toc-toggle.ts` 并接入 app.ts**

新增 `web/src/toc-toggle.ts`，行为要求：

- 只在 `html[data-layout="toc-right"]` 下让浮动 TOC 默认打开。
- 非 `toc-right` 下移除 `body.toc-floating-open`，按钮 `aria-expanded=false`。
- 点击 `#toc-toggle-button` 切换 `body.toc-floating-open`。
- 当 `body.preview-active` 变为 true 时，默认关闭浮动 TOC。
- preview 打开后，用户点击按钮仍可重新打开 TOC 用于跳转。
- 本状态不写入 localStorage，不新增配置项。

在 `web/src/app.ts` 的 `setupOnce()` 或 toolbar 初始化后调用：

```ts
setupTocToggle({ documentRef: document });
```

如果 `applyLayoutMode()` 切换 layout 后需要同步 TOC 状态，可从 `layout-mode.ts` 派发布局变更事件，或在 `setupLayoutControls()` 点击处理后调用同步回调。不要把 TOC 状态塞进 Go 注入的 app config。

- [x] **Step 4: 写 CSS 行为测试**

在 `web/src/layout-css.test.ts` 增加断言：

```ts
test('defines preview-active and mobile fallback layout rules', () => {
    expect(cssText).toContain('preview-active');
    expect(cssText).toContain('max-width: 1023px');
    expect(cssText).toContain('content-search-wrapper');
    expect(cssText).toContain('toc-floating-open');
    expect(cssText).toContain('.toc-toggle-button');
});
```

- [x] **Step 5: 运行测试确认失败或不完整**

Run:

```bash
cd web && bun test src/layout-css.test.ts src/toc-toggle.test.ts
```

Expected: 如果 Task 4 已加入部分规则，可能 PASS；若 PASS，需要人工检查 CSS 是否真的覆盖下面三类场景，不足则补更具体断言。

- [x] **Step 6: 处理 preview-active**

桌面布局打开右侧预览面板时，避免 Files + TOC + Body + Preview 四区过窄。

推荐规则：

- `compact`: 沿用现有右侧预览体验，CSS 可根据 grid 结构调整实现细节，但正文和 preview 仍按约 60%/40% 分配。
- `toc-middle`: preview 打开时隐藏 `.toc-pane`，`.app-shell` grid 变为 files + content。
- `toc-right`: preview 打开时默认关闭浮动 TOC，但不 `display: none`，用户点击 `.toc-toggle-button` 可重新打开 overlay。
- 预览面板宽度使用 `--preview-width: clamp(360px, 38vw, 560px)`，避免固定 40% 在窄桌面压缩正文过度。

示例方向：

```css
:root {
    --preview-width: clamp(360px, 38vw, 560px);
}

@media (min-width: 1024px) {
    html[data-layout="toc-middle"] body.preview-active .toc-pane {
        display: none;
    }

    html[data-layout="toc-middle"] body.preview-active .app-shell {
        grid-template-columns: var(--sidebar-width) minmax(0, 1fr);
        grid-template-areas: "files content";
        padding-right: var(--preview-width);
    }

    html[data-layout="toc-right"] body.preview-active .app-shell {
        grid-template-columns: var(--sidebar-width) minmax(0, 1fr);
        grid-template-areas: "files content";
        padding-right: var(--preview-width);
    }

    html[data-layout="toc-right"] body.preview-active:not(.toc-floating-open) .toc-pane {
        opacity: 0;
        pointer-events: none;
        transform: translateX(calc(100% + 24px));
    }

    html[data-layout="toc-middle"] body.sidebar-collapsed.preview-active .app-shell,
    html[data-layout="toc-right"] body.sidebar-collapsed.preview-active .app-shell {
        grid-template-columns: var(--sidebar-collapsed-width) minmax(0, 1fr);
    }
}
```

- [x] **Step 7: 处理 content search 定位**

当前 `.content-search-wrapper` 用 `left: calc(var(--sidebar-width) + 30px)`。切到 `.app-shell` grid 后，搜索框应锚定正文区域，避免压到 TOC：

- 推荐将 `.content-wrapper` 设为 `position: relative`，把 `.content-search-wrapper` 放回正文上下文内定位。
- 如果短期必须保留 fixed 搜索框，需要按 layout 设置不同 left：
  - `toc-middle`: `left: calc(var(--sidebar-width) + var(--toc-width) + 30px)`。
  - `toc-right`: `left: calc(var(--sidebar-width) + 30px)`。
  - sidebar collapsed 时同步使用 collapsed width。
- 优先选择正文相对定位方案，因为它和 grid 布局耦合更低。

- [x] **Step 8: 移动端回退**

在 `@media (max-width: 1023px)` 中确保：

- `.app-shell` 使用单列：`grid-template-columns: minmax(0, 1fr)`。
- `.files-pane` 继续 `display: none` 或沿用现有移动端行为。
- `.content-wrapper` `margin-left: 0`。
- `.toc-pane` 不作为独立列展示；`toc-right` 的浮动 TOC 在移动端默认隐藏，开关按钮也隐藏，避免覆盖正文。
- 不修改 localStorage 中保存的 layout mode。

- [x] **Step 9: 运行前端测试和构建**

Run:

```bash
cd web
bun test
bun run build
cd ..
```

Expected: PASS。

- [x] **Step 10: 提交**

```bash
git add web/src/app.ts web/src/toc-toggle.ts web/src/toc-toggle.test.ts web/src/style/app.css web/src/layout-css.test.ts
git commit -m "feat(web): add floating toc controls"
```

## Task 6: 浏览器级布局 smoke 验证

**Files:**
- Modify: `docs/superpowers/plans/2026-05-28-markview-config-files-phase-2.md`

- [ ] **Step 1: 构建并启动本地服务**

Run:

```bash
cd web
bun run build
cd ..
New-Item -ItemType Directory -Force -Path tmp | Out-Null
go build -o tmp/markview-smoke.exe .
$server = Start-Process -FilePath .\tmp\markview-smoke.exe -ArgumentList @('--no-browser', '--port', '6224', '.') -WindowStyle Hidden -PassThru
```

不要使用 `go run` 启动 smoke 服务；先构建再运行可避免编译等待和服务生命周期不稳定。
浏览器验证结束后必须清理服务进程：

```powershell
if ($server -and -not $server.HasExited) {
    Stop-Process -Id $server.Id -Force
}
```

如果 smoke 中途失败，也必须在 `finally` 或等价清理步骤中停止该进程，避免遗留端口占用。

- [ ] **Step 2: 使用浏览器检查 desktop 布局**

用浏览器打开 `http://127.0.0.1:6224/`，分别验证：

- `compact`: Files 和 TOC 在左侧 sidebar，正文右侧，视觉与一期前一致。
- `toc-middle`: Files | TOC | Body 三栏。
- `toc-right`: Files | Body 主布局，TOC 作为右侧浮动面板显示，正文不为 TOC 预留 grid 列。
- 点击 sidebar collapse 后，`toc-middle` 中 files 收起且 TOC 仍可见；`toc-right` 中 files 收起且浮动 TOC 开关仍可用。
- 打开 link preview 后，正文和 preview 不与 TOC 重叠；`toc-right` 中浮动 TOC 默认隐藏，但点击 TOC 开关可重新显示并点击目录跳转。

必须使用自动化浏览器或等价脚本断言：

- `document.documentElement.dataset.layout` 对应当前选择。
- `toc-middle` 中 `getBoundingClientRect()` 的 Files、TOC、Body left 顺序符合模式，且两两不发生水平重叠。
- `toc-right` 中 Files 和 Body 不重叠；TOC 浮动面板 overlay 不改变 Body 的 grid 宽度。
- `toc-right` + Width Full 时，正文只铺满 content pane，不侵占 preview 或浮动 TOC。
- `toc-right` 中点击 `#toc-toggle-button` 后 `body.toc-floating-open` 和 `aria-expanded` 正确切换。
- 打开 preview 后，Body 与 Preview 不重叠，`toc-right` 默认关闭浮动 TOC；手动打开浮动 TOC 后能点击 TOC 项跳转，跳转后本期建议保持 TOC 打开。
- desktop 截图无横向重叠。

- [ ] **Step 3: 使用浏览器检查 mobile 回退**

设置 viewport 宽度小于 `1024px`，验证：

- 保存 `toc-right` 后刷新，dataset 仍是 `toc-right`。
- 视觉布局回退为单正文/compact，不展示三列。
- 内容不被 toolbar、preview、TOC 或 TOC 开关遮挡。

- [ ] **Step 4: 记录 smoke 结果**

在本计划 Task 6 下方追加简短记录：

```md
Smoke result:
- Desktop compact/toc-middle/toc-right floating TOC: PASS
- Sidebar collapsed with toc-middle and toc-right floating TOC: PASS
- Preview panel with toc-right default-hidden/manual-open TOC: PASS
- Mobile fallback: PASS
```

- [ ] **Step 5: 提交**

```bash
git add docs/superpowers/plans/2026-05-28-markview-config-files-phase-2.md
git commit -m "docs: record layout smoke verification"
```

## Task 7: 文档收尾与质量门

**Files:**
- Modify: `docs/TODO.md`
- Modify: `docs/superpowers/specs/2026-05-28-markview-config-files-design.md`
- Modify: `docs/superpowers/plans/2026-05-28-markview-config-files-phase-2.md`

- [ ] **Step 1: 运行完整质量门**

Run:

```bash
go test ./...
go build ./...
cd web
bun test
bun run build
cd ..
```

Expected: 全部 PASS。

- [ ] **Step 2: 更新 TODO**

当二期实现和 smoke 都完成后，更新 `docs/TODO.md`：

```md
- [x] 新增支持全局和项目级别的配置文件 `markview.json`（详细说明见下面对应章节）
  - [x] 一期：配置文件读取/合并、页面配置注入、preview_exts 生效、layout 基础链路
  - [x] 二期：设置面板 layout 控件和完整布局模式
```

并将章节标题状态从 `⏳` 改为 `✅`。

- [ ] **Step 3: 更新设计文档链接**

确认设计文档顶部相关文档包含：

- TODO 需求
- 一期实施计划
- 二期实施计划

如果二期实现与设计有差异，补充“实施说明”而不是悄悄让文档过期。

- [ ] **Step 4: 提交收尾文档**

```bash
git add docs/TODO.md docs/superpowers/specs/2026-05-28-markview-config-files-design.md docs/superpowers/plans/2026-05-28-markview-config-files-phase-2.md
git commit -m "docs: complete config layout phase two"
```

- [ ] **Step 5: 最终状态检查**

Run:

```bash
git status --short --branch
git log --oneline -10
```

Expected: 工作区干净；本地分支包含二期提交。除非用户明确要求，不要 push。

## 最终质量门

完成所有任务后运行：

```bash
go test ./...
go build ./...
cd web
bun test
bun run build
cd ..
git diff --check
git status --short --branch
```

预期全部 PASS，且没有未提交变更。

## Plan Self-Review

- Scope: 二期只覆盖 layout 控件和三种布局视觉，不修改 Go 配置合并模型。
- Compatibility: `compact` 默认视觉保持稳定；移动端回退 compact 视觉但保留用户偏好。
- Testability: 偏好逻辑、控件状态和 CSS 关键规则有自动测试；实际布局顺序通过浏览器 smoke 验证。
- Risk: 独立 TOC pane 需要调整模板和核心布局 CSS，改动比 fixed TOC 更大；收益是 DOM 语义清晰、collapse/preview/mobile 状态更可控，后续 TOC resize 或独立折叠也更容易演进。
