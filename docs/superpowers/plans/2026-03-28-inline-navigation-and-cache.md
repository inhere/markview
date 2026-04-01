# Inline Navigation And Cache Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 Markdown 切换和 SSE 热更新改为无刷新局部更新，并为静态资源增加缓存策略，显著降低页面切换时的资源重复加载和初始化成本。

**Architecture:** 保持服务端输出完整 HTML，前端通过抓取目标 HTML 并替换局部 DOM 来实现导航与热更新。后端只补缓存头和本地静态资源支持，前端负责一次性初始化与可重复页面渲染的拆分。

**Tech Stack:** Go `net/http`, `html/template`, Bun, TypeScript, DOMParser, History API, EventSource

---

### Task 1: Lock cache behavior with tests

**Files:**
- Modify: `handlers_test.go`
- Modify: `main.go`
- Modify: `handlers.go`

- [ ] **Step 1: Write failing tests**

为这些行为补测试：
- Markdown 页面返回 `Cache-Control: no-store`
- `/static/*` 返回 `Cache-Control: public, max-age=0, must-revalidate`

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./...`
Expected: FAIL because cache headers are not implemented yet

- [ ] **Step 3: Implement minimal server changes**

为 Markdown 响应和静态资源响应补缓存头。

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./...`
Expected: PASS

### Task 2: Prepare template for local assets and patchable sections

**Files:**
- Modify: `web/template.html`

- [ ] **Step 1: Localize highlight stylesheet**

改为引用本地静态样式资源。

- [ ] **Step 2: Ensure patchable page sections have stable selectors**

保证正文、文件信息、内嵌 JSON 都有稳定选择器供局部替换。

### Task 3: Refactor web init lifecycle

**Files:**
- Modify: `web/app.ts`

- [ ] **Step 1: Split one-time setup from per-page render**

抽出 `setupOnce()` 与 `renderPage()`。

- [ ] **Step 2: Make per-page render idempotent**

每次页面切换后都能安全重新执行 TOC、目录树、高亮、Mermaid。

### Task 4: Implement inline navigation and history

**Files:**
- Modify: `web/app.ts`

- [ ] **Step 1: Parse target HTML**

使用 `fetch + DOMParser` 提取目标页面的关键区域。

- [ ] **Step 2: Intercept sidebar and content markdown links**

接管目录树和正文中的站内 Markdown 链接。

- [ ] **Step 3: Update history**

接入 `pushState` 和 `popstate`。

- [ ] **Step 4: Handle hash navigation**

同页锚点直接滚动，跨页锚点在局部导航完成后滚动。

### Task 5: Reuse the same pipeline for SSE

**Files:**
- Modify: `web/app.ts`

- [ ] **Step 1: Replace full reload**

SSE 收到更新时不再执行 `window.location.reload()`。

- [ ] **Step 2: Preserve scroll position on hot refresh**

刷新当前页面内容后恢复滚动位置。

### Task 6: Verify end-to-end behavior

**Files:**
- Modify: `web/app.ts`
- Modify: `web/template.html`
- Modify: `handlers.go`
- Modify: `main.go`

- [ ] **Step 1: Run Go tests**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 2: Run web build**

Run: `bun run build`
Workdir: `web`
Expected: PASS

- [ ] **Step 3: Browser verification**

验证：
- 切换目录树文件时不整页刷新
- 正文站内链接走无刷新导航
- 后退/前进可恢复内容
- SSE 热更新不整页刷新
