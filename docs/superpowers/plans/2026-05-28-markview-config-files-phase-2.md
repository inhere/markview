# MarkView 配置文件支持二期 Implementation Plan

相关文档：

- [TODO 需求](../../TODO.md#新增支持全局和项目级别的配置文件-)
- [设计文档](../specs/2026-05-28-markview-config-files-design.md)
- [一期实施计划](2026-05-28-markview-config-files-phase-1.md)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在一期配置注入和 layout dataset 基础上，实现设置面板 layout 控件，以及 `compact`、`toc-middle`、`toc-right` 三种完整布局。

**Architecture:** 保持单一 Files DOM、单一 TOC DOM、单一正文 DOM，不复制目录树或 TOC。`web/src/preferences.ts` 负责 layout 偏好读写和清除；`web/src/app.ts` 负责初始化、设置面板事件和页面级 dataset；`web/template.html` 只补必要的布局容器/控件标记；`web/src/style/app.css` 通过 `html[data-layout]` 和现有 `body.sidebar-collapsed`/`body.preview-active` 状态完成视觉布局。

**Tech Stack:** TypeScript、Bun、JSDOM、CSS Grid/Flex、Go embed 现有前端构建链路。

---

## 复审结论

一期已经完成配置文件读取、合并、页面注入、`preview_exts` 生效和 layout 基础链路。二期只做 UI/布局，不再扩大 Go 配置模型范围。

关键约束：

- 保持 `compact` 现有体验，避免默认用户打开页面后视觉突变。
- `toc-middle` 和 `toc-right` 只在桌面宽度启用三栏；移动端视觉回退为 compact，不清除用户保存的 layout 偏好。
- 设置面板修改 layout 后立即更新 `document.documentElement.dataset.layout` 并持久化。
- Reset/Default 清除本地 layout 覆盖，恢复服务端注入的项目默认 layout。
- 不复制 `#toc-list`，确保 `generateTOC()`、`highlightTOC()` 仍只维护一个 TOC。
- 右侧 preview panel 打开时避免四列拥挤；三栏模式下优先隐藏独立 TOC 或压缩到不重叠的状态。

## 文件结构

- Modify: `web/template.html`
  - 设置面板新增 Layout 分段控件和 Default 按钮。
  - 给 TOC section 增加稳定 id（例如 `toc-panel`），便于 CSS/测试定位。
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
- Modify: `web/src/style/app.css`
  - 新增 layout 控件样式。
  - 改造页面布局为可响应的 grid/flex 规则。
  - 实现 `compact`、`toc-middle`、`toc-right`。
  - 处理 sidebar collapsed、preview-active、content-search、移动端回退。
- Add: `web/src/layout-smoke.test.ts` 或等价浏览器 smoke 脚本
  - 运行本地构建产物后，用真实浏览器/Playwright 检查三种 layout 的 bbox 顺序和无重叠。
- Add/Modify tests as needed:
  - `web/src/preferences.test.ts`
  - 如当前 `app.ts` 仍难以直接测试 toolbar 初始化，可新增轻量模块 `web/src/layout-mode.ts` 和 `web/src/layout-mode.test.ts` 承载纯 DOM 控件逻辑，再由 `app.ts` 调用。

## Task 1: Layout 偏好清除与控件状态基础

**Files:**
- Modify: `web/src/preferences.ts`
- Modify: `web/src/preferences.test.ts`

- [ ] **Step 1: 写失败测试**

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

- [ ] **Step 2: 运行测试确认失败**

Run:

```bash
cd web && bun test src/preferences.test.ts
```

Expected: FAIL，`clearStoredLayoutMode` 未定义。

- [ ] **Step 3: 实现清除 helper**

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

- [ ] **Step 4: 运行测试确认通过**

Run:

```bash
cd web && bun test src/preferences.test.ts
```

Expected: PASS。

- [ ] **Step 5: 提交**

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

- [ ] **Step 1: 写失败测试**

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

- [ ] **Step 2: 运行测试确认失败**

Run:

```bash
cd web && bun test src/layout-mode.test.ts src/preferences.test.ts
```

Expected: FAIL，`layout-mode` 模块或相关函数不存在。

- [ ] **Step 3: 模板新增 Layout 控件**

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

- [ ] **Step 4: 实现 layout 控件模块**

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

- [ ] **Step 5: app.ts 接入**

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

注意：局部页面导航不重新应用 layout。

- [ ] **Step 6: 运行前端测试**

Run:

```bash
cd web && bun test
```

Expected: PASS。

- [ ] **Step 7: 提交**

```bash
git add web/template.html web/src/app.ts web/src/layout-mode.ts web/src/layout-mode.test.ts web/src/preferences.ts web/src/preferences.test.ts
git commit -m "feat(web): add layout selector controls"
```

## Task 3: DOM 标记整理与 compact 兼容

**Files:**
- Modify: `web/template.html`
- Modify: `web/src/style/app.css`

- [ ] **Step 1: 写/更新测试或快照检查**

如果现有测试不适合模板结构，可以新增 Go handler 测试，确认完整页面仍包含关键结构：

```go
assert.StrContains(t, body, `id="toc-panel"`)
assert.StrContains(t, body, `class="content-inner"`)
assert.StrContains(t, body, `data-layout-mode="compact"`)
```

测试文件优先放在 `internal/handlers/handlers_test.go`，复用现有完整页面渲染 helper。

- [ ] **Step 2: 运行测试确认失败**

Run:

```bash
go test ./internal/handlers
```

Expected: FAIL，模板还没有新增标记。

- [ ] **Step 3: 调整模板标记**

在 `web/template.html`：

- 给 TOC panel 增加 id：

```html
<section class="sidebar-panel sidebar-panel-toc" id="toc-panel">
```

- 将正文 inline wrapper：

```html
<div style="width: 100%; max-width: var(--layout-max-width); display: flex; flex-direction: column; align-items: center;">
```

改为：

```html
<div class="content-inner">
```

- [ ] **Step 4: CSS 保持 compact 视觉不变**

在 `web/src/style/app.css` 增加：

```css
:root {
    --toc-width: 240px;
    --toc-min-width: 180px;
    --toc-max-width: 360px;
}

.content-inner {
    width: 100%;
    max-width: var(--layout-max-width);
    display: flex;
    flex-direction: column;
    align-items: center;
}
```

确认 `compact` 下仍然：

- `.sidebar` fixed left。
- `.content-wrapper` 使用 `margin-left: var(--sidebar-width)`。
- `.sidebar-panel-files` 和 `.sidebar-panel-toc` 保持在同一个 sidebar 内。

- [ ] **Step 5: 运行测试**

Run:

```bash
go test ./internal/handlers
cd web && bun test
```

Expected: PASS。

- [ ] **Step 6: 提交**

```bash
git add web/template.html web/src/style/app.css internal/handlers/handlers_test.go
git commit -m "feat(web): prepare layout panes"
```

## Task 4: 桌面三栏布局 CSS

**Files:**
- Modify: `web/src/style/app.css`

- [ ] **Step 1: 写布局断言测试**

优先新增轻量 CSS 文本断言测试 `web/src/layout-css.test.ts`，避免引入浏览器依赖：

```ts
import { describe, expect, test } from 'bun:test';
import cssText from './style/app.css' with { type: 'text' };

describe('layout CSS modes', () => {
    test('defines desktop rules for toc-middle and toc-right', () => {
        expect(cssText).toContain('[data-layout="toc-middle"]');
        expect(cssText).toContain('[data-layout="toc-right"]');
        expect(cssText).toContain('--toc-width');
        expect(cssText).toContain('position: fixed');
        expect(cssText).toContain('calc(var(--sidebar-width) + var(--toc-width))');
    });

    test('keeps mobile layout compact', () => {
        expect(cssText).toContain('@media (max-width: 1023px)');
        expect(cssText).toContain('[data-layout="toc-middle"] .sidebar-panel-toc');
    });
});
```

如果 Bun 对 CSS text import 支持不稳定，可以改为在测试里 `Bun.file(new URL('./style/app.css', import.meta.url)).text()`。

- [ ] **Step 2: 运行测试确认失败**

Run:

```bash
cd web && bun test src/layout-css.test.ts
```

Expected: FAIL，CSS 还没有三栏规则。

- [ ] **Step 3: 实现桌面三栏规则**

采用固定定位 TOC 策略，不移动 DOM，不使用 `display: contents`。原因：当前 TOC 是 `.sidebar > .sidebar-panels > .sidebar-panel-toc` 的后代，不是 `body` grid 直接子项；让嵌套节点参与 body grid 需要大幅 DOM 重排，风险高。二期先保持单一 TOC DOM 在当前位置，三栏模式下把 `.sidebar-panel-toc` 固定成独立列，并通过 `.content-wrapper` 的 margin 避让。

在 `web/src/style/app.css` 中新增桌面规则（建议放在 `.content-wrapper` 相关样式附近）：

```css
@media (min-width: 1024px) {
    [data-layout="toc-middle"] .sidebar,
    [data-layout="toc-right"] .sidebar {
        position: fixed;
        top: 0;
        left: 0;
        width: var(--sidebar-width);
    }

    [data-layout="toc-middle"] .sidebar-panel-toc,
    [data-layout="toc-right"] .sidebar-panel-toc {
        position: fixed;
        top: 0;
        bottom: 0;
        width: var(--toc-width);
        min-width: var(--toc-min-width);
        max-width: var(--toc-max-width);
        height: 100vh;
        z-index: 9;
        border-radius: 0;
    }

    [data-layout="toc-middle"] .sidebar-panel-toc {
        left: var(--sidebar-width);
    }

    [data-layout="toc-right"] .sidebar-panel-toc {
        right: 0;
    }

    [data-layout="toc-middle"] .content-wrapper {
        margin-left: calc(var(--sidebar-width) + var(--toc-width));
    }

    [data-layout="toc-right"] .content-wrapper {
        margin-left: var(--sidebar-width);
        margin-right: var(--toc-width);
    }
}
```

同时需要在三栏模式下让 `.sidebar-panel-files` 占满 sidebar 剩余空间，并让 `.sidebar-panel-toc` 从 `.sidebar-panels` 的正常 flex 流中脱离后不留下空白。实现前必须读完整 `.sidebar`、`.sidebar-panels`、`.content-wrapper`、`.preview-panel` 相关 CSS，避免新规则打穿现有状态。

- [ ] **Step 4: 处理 sidebar collapsed**

规则：

- `compact`: 沿用当前行为。
- `toc-middle`/`toc-right`: collapse 只影响 files pane 宽度，TOC 继续可见。

CSS 需要覆盖：

```css
@media (min-width: 1024px) {
    [data-layout="toc-middle"] body.sidebar-collapsed .sidebar-panels,
    [data-layout="toc-right"] body.sidebar-collapsed .sidebar-panels {
        display: flex;
    }

    [data-layout="toc-middle"] body.sidebar-collapsed .sidebar-panel-files,
    [data-layout="toc-right"] body.sidebar-collapsed .sidebar-panel-files {
        display: none;
    }

    [data-layout="toc-middle"] body.sidebar-collapsed .sidebar,
    [data-layout="toc-right"] body.sidebar-collapsed .sidebar {
        width: var(--sidebar-collapsed-width);
    }

    [data-layout="toc-middle"] body.sidebar-collapsed .sidebar-panel-toc {
        left: var(--sidebar-collapsed-width);
    }

    [data-layout="toc-middle"] body.sidebar-collapsed .content-wrapper {
        margin-left: calc(var(--sidebar-collapsed-width) + var(--toc-width));
    }

    [data-layout="toc-right"] body.sidebar-collapsed .content-wrapper {
        margin-left: var(--sidebar-collapsed-width);
    }
}
```

这一段必须显式覆盖现有 `.sidebar-collapsed .sidebar-panels { display: none; }`，否则 TOC 作为 `.sidebar-panels` 的后代会一起消失，违反“三栏模式 collapse 只影响 files pane”的目标。并确保 `.sidebar-icons` 在三栏模式 collapse 后仍可点击恢复 files。

同时在 `web/src/layout-css.test.ts` 中增加断言，至少覆盖：

```ts
expect(cssText).toContain('body.sidebar-collapsed .sidebar-panels');
expect(cssText).toContain('body.sidebar-collapsed .sidebar-panel-files');
```

- [ ] **Step 5: 运行前端测试**

Run:

```bash
cd web && bun test
```

Expected: PASS。

- [ ] **Step 6: 提交**

```bash
git add web/src/style/app.css web/src/layout-css.test.ts
git commit -m "feat(web): add desktop toc layouts"
```

## Task 5: Preview panel、搜索框和移动端适配

**Files:**
- Modify: `web/src/style/app.css`
- Modify/Add: `web/src/layout-css.test.ts`

- [ ] **Step 1: 写 CSS 行为测试**

在 `web/src/layout-css.test.ts` 增加断言：

```ts
test('defines preview-active and mobile fallback layout rules', () => {
    expect(cssText).toContain('preview-active');
    expect(cssText).toContain('max-width: 1023px');
    expect(cssText).toContain('content-search-wrapper');
});
```

- [ ] **Step 2: 运行测试确认失败或不完整**

Run:

```bash
cd web && bun test src/layout-css.test.ts
```

Expected: 如果 Task 4 已加入部分规则，可能 PASS；若 PASS，需要人工检查 CSS 是否真的覆盖下面三类场景，不足则补更具体断言。

- [ ] **Step 3: 处理 preview-active**

桌面三栏模式打开右侧预览面板时，避免 Files + TOC + Body + Preview 四列过窄。

推荐规则：

- `compact`: 沿用现有 `body.preview-active .content-wrapper { width: 60%; margin-right: 40%; }`。
- `toc-middle`/`toc-right`: preview 打开时隐藏独立 TOC pane 或将 TOC pane 设为 `display: none`，grid 变为 files + content。
- 预览面板仍占右侧 40%，正文区不与其重叠。

示例方向：

```css
@media (min-width: 1024px) {
    [data-layout="toc-middle"] body.preview-active .sidebar-panel-toc,
    [data-layout="toc-right"] body.preview-active .sidebar-panel-toc {
        display: none;
    }

    [data-layout="toc-middle"] body.preview-active .content-wrapper {
        margin-left: var(--sidebar-width);
        margin-right: 40%;
        width: auto;
    }

    [data-layout="toc-right"] body.preview-active .content-wrapper {
        margin-left: var(--sidebar-width);
        margin-right: 40%;
        width: auto;
    }

    [data-layout="toc-middle"] body.sidebar-collapsed.preview-active .content-wrapper,
    [data-layout="toc-right"] body.sidebar-collapsed.preview-active .content-wrapper {
        margin-left: var(--sidebar-collapsed-width);
    }
}
```

- [ ] **Step 4: 处理 content search 定位**

当前 `.content-search-wrapper` 用 `left: calc(var(--sidebar-width) + 30px)`。三栏模式需要避免搜索框压到 TOC：

- 三栏模式下优先改为相对 `.content-wrapper` 定位。
- 或按 layout 设置不同 left：
  - `toc-middle`: `left: calc(var(--sidebar-width) + var(--toc-width) + 30px)`
  - `toc-right`: `left: calc(var(--sidebar-width) + 30px)`
- sidebar collapsed 时同步使用 collapsed width。

- [ ] **Step 5: 移动端回退**

在 `@media (max-width: 1023px)` 中确保：

- `.sidebar` 继续 `display: none` 或沿用现有移动端行为。
- `.content-wrapper` `margin-left: 0`。
- `.sidebar-panel-toc` 不作为独立列展示。
- 不修改 localStorage 中保存的 layout mode。

- [ ] **Step 6: 运行前端测试和构建**

Run:

```bash
cd web
bun test
bun run build
cd ..
```

Expected: PASS。

- [ ] **Step 7: 提交**

```bash
git add web/src/style/app.css web/src/layout-css.test.ts
git commit -m "fix(web): adapt toc layouts for preview and mobile"
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
- `toc-right`: Files | Body | TOC 三栏。
- 点击 sidebar collapse 后，三栏模式中 files 收起，TOC 仍可见。
- 打开 link preview 后，正文和 preview 不与 TOC 重叠。

必须使用自动化浏览器或等价脚本断言：

- `document.documentElement.dataset.layout` 对应当前选择。
- `getBoundingClientRect()` 中 Files、TOC、Body 的 left 顺序符合模式。
- Files、TOC、Body 两两不发生水平重叠。
- 打开 preview 后，Body 与 Preview 不重叠，TOC 在三栏模式下按计划隐藏或避让。
- desktop 截图无横向重叠。

- [ ] **Step 3: 使用浏览器检查 mobile 回退**

设置 viewport 宽度小于 `1024px`，验证：

- 保存 `toc-right` 后刷新，dataset 仍是 `toc-right`。
- 视觉布局回退为单正文/compact，不展示三列。
- 内容不被 toolbar、preview、TOC 遮挡。

- [ ] **Step 4: 记录 smoke 结果**

在本计划 Task 6 下方追加简短记录：

```md
Smoke result:
- Desktop compact/toc-middle/toc-right: PASS
- Sidebar collapsed in three-column layouts: PASS
- Preview panel with three-column layouts: PASS
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
  - [x] 二期：设置面板 layout 控件和完整三栏布局
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
- Risk: `app.ts` 已较大，控件逻辑建议拆到 `layout-mode.ts`，降低继续堆叠大型初始化函数的风险。
