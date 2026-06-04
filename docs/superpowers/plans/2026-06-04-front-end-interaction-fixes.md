# MarkView Frontend Interaction Fixes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修复 MarkView 前端交互回归并改善长页面滚动体验。

**Architecture:** 沿用现有模块边界，在 `sidebar-resize.ts`、`mermaid.ts`、`highlight.ts`、`link-preview.ts` 和 `app.css` 内做局部修复。测试按现有 Bun + jsdom 模式补齐，CSS 规则继续用文本断言固定关键行为。

**Tech Stack:** TypeScript、Bun test、jsdom、highlight.js、Mermaid、CSS。

设计文档：[2026-06-04-front-end-interaction-fixes-design.md](../specs/2026-06-04-front-end-interaction-fixes-design.md)

---

### Task 1: 测试锁定回归

**Files:**
- Modify: `web/src/sidebar.test.ts`
- Modify: `web/src/mermaid.test.ts`
- Modify: `web/src/link-preview.test.ts`
- Create: `web/src/highlight.test.ts`
- Modify: `web/src/layout-css.test.ts`

- [x] **Step 1: 为 sidebar 拖拽写失败测试**

覆盖鼠标拖动后 `--sidebar-width` 更新，并且鼠标释放后清理拖拽状态。

- [x] **Step 2: 为 Mermaid 复制按钮写失败测试**

覆盖 Mermaid 容器增强后出现复制按钮，点击时复制原始源码。

- [x] **Step 3: 为高亮 fallback 写失败测试**

覆盖未知语言不会抛错，并标记为已处理。

- [x] **Step 4: 为链接预览幂等增强写失败测试**

覆盖同一个内容区域重复增强不会重复按钮，并且刷新后的新 DOM 能重新生成按钮。

- [x] **Step 5: 为滚动性能 CSS 写失败测试**

覆盖 Mermaid 和表格容器包含 `content-visibility: auto` 等关键规则。

- [x] **Step 6: 运行前端测试确认失败**

Run: `bun test web/src/sidebar.test.ts web/src/mermaid.test.ts web/src/link-preview.test.ts web/src/highlight.test.ts web/src/layout-css.test.ts`

---

### Task 2: 实现前端修复

**Files:**
- Modify: `web/src/sidebar-resize.ts`
- Modify: `web/src/mermaid.ts`
- Modify: `web/src/highlight.ts`
- Modify: `web/src/app.ts`
- Modify: `web/src/link-preview.ts`
- Modify: `web/src/style/app.css`

- [x] **Step 1: 修复 sidebar 拖拽**

扩大拖拽命中、拖动中禁用过渡，并保持宽度持久化。

- [x] **Step 2: 给 Mermaid 添加复制按钮**

按钮复制 `data-source`，成功后显示短暂成功态。

- [x] **Step 3: 增加统一安全高亮入口**

未知语言 fallback 到 plaintext，常见别名注册到已有语言。

- [x] **Step 4: 修复链接预览幂等增强**

避免重复包裹，刷新后的新内容仍生成按钮。

- [x] **Step 5: 添加滚动性能 CSS**

为复杂块启用 `content-visibility` 和容器滚动限制。

- [x] **Step 6: 运行前端测试确认通过**

Run: `bun test web/src/sidebar.test.ts web/src/mermaid.test.ts web/src/link-preview.test.ts web/src/highlight.test.ts web/src/layout-css.test.ts`

---

### Task 3: 全量验证和收尾

**Files:**
- Modify: `docs/superpowers/plans/2026-06-04-front-end-interaction-fixes.md`

- [x] **Step 1: 运行全部前端测试**

Run: `bun test`

- [x] **Step 2: 运行 Go 主链路测试**

Run: `go test ./...`

- [x] **Step 3: 更新计划 checkbox**

将已完成步骤标记为 `[x]`。

- [x] **Step 4: 提交并推送**

Run: `git status --short && git add ... && git commit -m "fix: restore frontend interactions" && git pull --rebase && git push`
