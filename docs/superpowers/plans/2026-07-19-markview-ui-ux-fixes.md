# MarkView UI/UX Fixes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修复 UI005、UI006、UX001、UX002，并按功能点分别测试、更新 TODO 和提交。

**Architecture:** 复用现有 CSS layout、TOC 状态、内容搜索和 SSE Toast，不新增组件或依赖。UI005/UI006 通过状态选择器修复布局；UX001 在 fetch 边界前拦截 path-only 查询；UX002 使用安全 DOM 节点生成最多三个站内链接。

**Tech Stack:** TypeScript、CSS、Bun test、JSDOM、Go 1.25+。

## Global Constraints

- 不新增依赖，不重构无关代码。
- 前端依赖和测试只使用 Bun，不使用 npm。
- 每个功能点遵循 RED → GREEN → 完整前端测试 → TODO checkbox → 独立提交。
- 用户已有的其他 `docs/TODO.md` 修改必须保留，不混入无关功能提交。

## 修订记录

| 日期 | 修订人 | 变更 |
| --- | --- | --- |
| 2026-07-19 | Codex | 初版实施计划，包含放大的 TOC 展开控制图标。 |

相关文档：

- [设计文档](../specs/2026-07-19-markview-ui-ux-fixes-design.md)
- [TODO 需求](../../TODO.md)

---

### Task 1: UI005 compact 收起侧栏占满高度

**Files:**
- Modify: `web/src/layout-css.test.ts`
- Modify: `web/src/style/layout.css`
- Modify: `docs/TODO.md`

**Interfaces:**
- Consumes: `body.sidebar-collapsed` 和 `html[data-layout="compact"]`。
- Produces: compact 收起状态下跨越两行的 `.files-pane`。

- [x] **Step 1: 写失败测试**

在 `keeps mobile layout compact and supports collapsed files width` 前增加：

```ts
test('lets the compact collapsed sidebar span the full viewport height', () => {
    expectRule(/html\[data-layout="compact"\]\s+body\.sidebar-collapsed\s+\.files-pane\s*\{[^}]*grid-row:\s*1\s*\/\s*-1;[^}]*height:\s*100vh;/s);
});
```

- [x] **Step 2: 验证 RED**

Run: `cd web && bun test src/layout-css.test.ts`

Expected: FAIL，找不到 compact collapsed `.files-pane` 的 `grid-row: 1 / -1`。

- [x] **Step 3: 最小实现**

在桌面端 compact collapsed 规则附近增加：

```css
html[data-layout="compact"] body.sidebar-collapsed .files-pane {
    grid-row: 1 / -1;
    height: 100vh;
}
```

- [x] **Step 4: 验证 GREEN 并更新进度**

Run: `cd web && bun test src/layout-css.test.ts && bun test`

Expected: layout 测试和全部前端测试通过。

将 `docs/TODO.md` 的 UI005 checkbox 改为 `[x]`。

- [x] **Step 5: 提交**

```bash
git add web/src/layout-css.test.ts web/src/style/layout.css
git add -p docs/TODO.md
git commit -m "fix: fill compact collapsed sidebar height"
```

---

### Task 2: UI006 停靠收起 TOC 并放大控制图标

**Files:**
- Modify: `web/src/layout-css.test.ts`
- Modify: `web/src/style/layout.css`
- Modify: `web/src/style/sidebar.css`
- Modify: `web/src/style/overlays.css`
- Modify: `docs/TODO.md`

**Interfaces:**
- Consumes: `toc-floating-open`、`preview-active`、`--sidebar-width`、`--sidebar-collapsed-width`、`--preview-width`。
- Produces: toc-middle 左下角与 toc-right 右下角的 48px 控制按钮；展开状态 24px SVG 图标。

- [x] **Step 1: 写失败测试**

替换旧的 44px 竖栏断言，并新增图标尺寸断言：

```ts
expectRule(/html\[data-layout="toc-middle"\]\s+body:not\(\.toc-floating-open\)\s+\.toc-pane\s*\{[^}]*top:\s*auto;[^}]*left:\s*calc\(var\(--sidebar-width\) \+ 16px\);[^}]*bottom:\s*16px;[^}]*width:\s*48px;[^}]*height:\s*48px;/s);
expectRule(/html\[data-layout="toc-right"\]\s+body:not\(\.toc-floating-open\)\s+\.toc-pane\s*\{[^}]*top:\s*auto;[^}]*right:\s*16px;[^}]*bottom:\s*16px;[^}]*width:\s*48px;[^}]*height:\s*48px;[^}]*transform:\s*none;/s);
expectRule(/\.toc-section-toggle\s+svg\s*\{[^}]*width:\s*24px;[^}]*height:\s*24px;/s);
expectRule(/html\[data-layout="toc-right"\]\s+body\.preview-active:not\(\.toc-floating-open\)\s+\.toc-pane\s*\{[^}]*right:\s*calc\(var\(--preview-width\) \+ 16px\);[^}]*transform:\s*none;/s);
```

- [x] **Step 2: 验证 RED**

Run: `cd web && bun test src/layout-css.test.ts`

Expected: FAIL，旧 CSS 仍使用 44px 竖栏和 translateX。

- [x] **Step 3: 最小实现**

在桌面布局中加入收起状态：

```css
html[data-layout="toc-middle"] body:not(.toc-floating-open) .toc-pane {
    top: auto;
    left: calc(var(--sidebar-width) + 16px);
    bottom: 16px;
    width: 48px;
    min-width: 48px;
    max-width: 48px;
    height: 48px;
    overflow: hidden;
    border-radius: 8px;
}

html[data-layout="toc-middle"] body.sidebar-collapsed:not(.toc-floating-open) .toc-pane {
    left: calc(var(--sidebar-collapsed-width) + 16px);
}

html[data-layout="toc-right"] body:not(.toc-floating-open) .toc-pane {
    top: auto;
    right: 16px;
    bottom: 16px;
    width: 48px;
    min-width: 48px;
    max-width: 48px;
    height: 48px;
    transform: none;
    overflow: hidden;
    border-radius: 8px;
}

body:not(.toc-floating-open) .toc-pane .sidebar-section-title {
    height: 100%;
    padding: 0;
    justify-content: center;
}
```

删除旧的 44px collapsed 宽度和平移规则。更新 preview 避让：

```css
html[data-layout="toc-right"] body.preview-active:not(.toc-floating-open) .toc-pane {
    right: calc(var(--preview-width) + 16px);
    transform: none;
}
```

放大展开和收起共用的控制图标：

```css
.toc-section-toggle {
    width: 32px;
    height: 32px;
}

.toc-section-toggle svg {
    width: 24px;
    height: 24px;
}
```

- [x] **Step 4: 验证 GREEN 并更新进度**

Run: `cd web && bun test src/layout-css.test.ts src/components/toc-toggle.test.ts && bun test`

Expected: 目标测试和全部前端测试通过。

将 `docs/TODO.md` 的 UI006 checkbox 改为 `[x]`，保留其位置说明。

- [x] **Step 5: 提交**

```bash
git add web/src/layout-css.test.ts web/src/style/layout.css web/src/style/sidebar.css web/src/style/overlays.css
git add -p docs/TODO.md
git commit -m "fix: dock collapsed toc controls"
```

---

### Task 3: UX001 阻止 path-only 搜索请求

**Files:**
- Modify: `web/src/components/content-search.test.ts`
- Modify: `web/src/components/content-search.ts`
- Modify: `docs/TODO.md`

**Interfaces:**
- Consumes: 搜索输入框的 trimmed query。
- Produces: `hasSearchTerms(query: string): boolean`，在 `performSearch` 发请求前使用。

- [x] **Step 1: 写失败测试**

导出查询判定并测试现有语义：

```ts
import { hasSearchTerms, renderResults, setupContentSearch } from './content-search';

describe('content search query validation', () => {
    test('skips path-only queries', () => {
        expect(hasSearchTerms('path:docs')).toBe(false);
        expect(hasSearchTerms('path:docs path:api')).toBe(false);
    });

    test('keeps keyword and pure exclude queries searchable', () => {
        expect(hasSearchTerms('path:docs keyword')).toBe(true);
        expect(hasSearchTerms('!vendor')).toBe(true);
    });
});
```

- [x] **Step 2: 验证 RED**

Run: `cd web && bun test src/components/content-search.test.ts`

Expected: FAIL，`hasSearchTerms` 尚未导出。

- [x] **Step 3: 最小实现**

```ts
export function hasSearchTerms(query: string): boolean {
    return query
        .split(/\s+/)
        .filter(term => term && !term.startsWith('path:'))
        .join(' ')
        .length >= 2;
}
```

将 `performSearch` 的首个条件改为：

```ts
if (!hasSearchTerms(query)) {
    resultsContainer.innerHTML = '';
    resultsContainer.style.display = 'none';
    return;
}
```

- [x] **Step 4: 验证 GREEN 并更新进度**

Run: `cd web && bun test src/components/content-search.test.ts && bun test`

Expected: 目标测试和全部前端测试通过。

将 `docs/TODO.md` 的 UX001 checkbox 改为 `[x]`。

- [x] **Step 5: 提交**

```bash
git add web/src/components/content-search.ts web/src/components/content-search.test.ts
git add -p docs/TODO.md
git commit -m "fix: skip path-only content searches"
```

---

### Task 4: UX002 可点击文件变动通知

**Files:**
- Modify: `web/src/components/live-status.test.ts`
- Modify: `web/src/components/live-status.ts`
- Modify: `web/src/style/overlays.css`
- Modify: `web/src/layout-css.test.ts`
- Modify: `docs/TODO.md`

**Interfaces:**
- Consumes: SSE `msg.files: string[]`。
- Produces: 最多三个 `.toast-file` 站内链接，以及可选 `.toast-count` 剩余数量。

- [x] **Step 1: 写失败测试**

通过已有 `setupLiveReloadStatus` 触发 Toast，验证安全链接和数量：

```ts
test('renders at most three safe clickable file links', () => {
    const files = ['docs/<unsafe>.md', 'docs/two.md', 'docs/three.md', 'docs/four.md'];
    source.onmessage?.({ data: JSON.stringify({ type: 'reload', files }) });

    const links = [...document.querySelectorAll<HTMLAnchorElement>('.toast-file')];
    expect(links).toHaveLength(3);
    expect(links[0].textContent).toBe('docs/<unsafe>.md');
    expect(links[0].innerHTML).not.toContain('<unsafe>');
    expect(links[0].getAttribute('href')).toBe('/docs/%3Cunsafe%3E.md');
    expect(document.querySelector('.toast-count')?.textContent).toBe('还有 1 个文件');
});
```

在 CSS 测试增加：

```ts
expect(overlaysCssText).toMatch(/\.file-change-toast\s*\{[^}]*width:\s*min\(520px,\s*calc\(100vw - 48px\)\);/s);
```

- [x] **Step 2: 验证 RED**

Run: `cd web && bun test src/components/live-status.test.ts src/layout-css.test.ts`

Expected: FAIL，当前 Toast 没有链接且最大宽度为 360px。

- [x] **Step 3: 最小实现**

在 `live-status.ts` 中用 DOM API 替换 message 的 `innerHTML`：

```ts
const label = document.createElement('span');
label.className = 'toast-label';
label.textContent = '文件变动';
message.appendChild(label);

files.slice(0, 3).forEach(file => {
    const link = document.createElement('a');
    link.className = 'toast-file';
    link.href = '/' + normalizeFilePath(file).split('/').map(encodeURIComponent).join('/');
    link.textContent = normalizeFilePath(file);
    message.appendChild(link);
});

if (files.length > 3) {
    const count = document.createElement('span');
    count.className = 'toast-count';
    count.textContent = `还有 ${files.length - 3} 个文件`;
    message.appendChild(count);
}
```

在 `overlays.css` 调整宽度和链接样式：

```css
.file-change-toast {
    width: min(520px, calc(100vw - 48px));
    min-width: min(280px, calc(100vw - 48px));
}

.toast-file {
    color: var(--text-heading);
    text-decoration: none;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.toast-file:hover {
    color: var(--accent-primary);
    text-decoration: underline;
}
```

- [x] **Step 4: 验证 GREEN 并更新进度**

Run: `cd web && bun test src/components/live-status.test.ts src/layout-css.test.ts && bun test && bun run build`

Expected: 目标测试、全部前端测试和前端构建通过。

将 `docs/TODO.md` 的 UX002 checkbox 改为 `[x]`。

- [x] **Step 5: 提交**

```bash
git add web/src/components/live-status.ts web/src/components/live-status.test.ts web/src/style/overlays.css web/src/layout-css.test.ts
git add -p docs/TODO.md
git commit -m "feat: link file change notifications"
```

---

### Task 5: 完整验证与交付检查

**Files:**
- Modify: `docs/superpowers/plans/2026-07-19-markview-ui-ux-fixes.md`

**Interfaces:**
- Consumes: Task 1–4 的四个独立提交。
- Produces: 完整质量门禁证据和已完成的计划 checkbox。

- [x] **Step 1: 运行完整质量门禁**

```bash
cd web && bun test && bun run build
cd .. && go test ./...
```

Expected: 所有命令退出码为 0。

Actual: `bun test` 102/102 通过，`bun run build` 成功；`go test ./...` 保留实施前已确认并获准忽略的 4 个 dotenv/bootstrap 基线失败，没有新增失败。

- [x] **Step 2: 真实浏览器验证**

构建临时二进制并验证四个目标交互；截图失败时记录明确错误，至少保留 DOM 尺寸、位置和网络调用证据。

Actual: Browser DOM、坐标、搜索请求和 SSE 链接跳转验证通过；截图在 hard reload 后仍返回 `image readback failed`。

- [x] **Step 3: 更新计划进度并提交**

将本计划所有 checkbox 更新为 `[x]`，然后：

```bash
git add docs/superpowers/plans/2026-07-19-markview-ui-ux-fixes.md
git commit -m "docs: complete UI and UX fix plan"
```

- [x] **Step 4: 转入 FEA002 设计**

UI/UX 提交全部完成后，按 brainstorming 流程创建独立的 global server 设计文档；不在本计划中实现 FEA002。
