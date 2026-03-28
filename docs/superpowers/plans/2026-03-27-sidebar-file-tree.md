# Sidebar File Tree Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在左侧侧栏中新增目录树，并与现有 TOC 以 1:2 高度比例共存，支持目录导航、展开收起和当前路径高亮。

**Architecture:** 服务端在渲染 Markdown 页面时生成目录树 JSON 并写入模板数据。前端在页面初始化阶段读取 JSON 渲染目录树，控制展开状态和当前节点高亮。模板只做布局与样式承载，不新增额外数据请求。

**Tech Stack:** Go, html/template, Bun, TypeScript, 原生 DOM API

---

### Task 1: Add tree builder tests

**Files:**
- Create: `handlers_test.go`
- Modify: `handlers.go`

- [ ] **Step 1: Write the failing test**

覆盖这些行为：
- 只保留 Markdown 文件和包含 Markdown 的目录
- 目录点击入口对应 `index.md`
- 目录内部不重复显示 `index.md`
- 当前文件位于子目录时，树节点顺序仍为目录优先、文件随后

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./...`
Expected: FAIL because tree builder does not exist yet

- [ ] **Step 3: Write minimal implementation**

在后端增加树节点结构和构建函数，先让测试通过。

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./...`
Expected: PASS

### Task 2: Inject tree data into template

**Files:**
- Modify: `main.go`
- Modify: `handlers.go`

- [ ] **Step 1: Extend page data**

新增目录树 JSON 和当前文件相对路径字段。

- [ ] **Step 2: Build render data**

在 Markdown 渲染阶段构造目录树并写入模板数据。

- [ ] **Step 3: Verify backend behavior**

Run: `go test ./...`
Expected: PASS

### Task 3: Update sidebar layout and styles

**Files:**
- Modify: `frontend/template.html`

- [ ] **Step 1: Add Files and TOC sections**

将当前单一 TOC 区改为上下分区结构。

- [ ] **Step 2: Add directory tree containers and JSON script blocks**

为前端渲染树结构提供挂载点和内嵌 JSON。

- [ ] **Step 3: Add CSS for 1:2 layout**

实现 section 标题、树节点、箭头按钮、当前节点高亮和独立滚动区域。

### Task 4: Render and control the tree in frontend

**Files:**
- Modify: `frontend/app.ts`

- [ ] **Step 1: Read tree data from DOM**

解析服务端注入的 JSON 和当前路径数据。

- [ ] **Step 2: Render directory tree**

创建目录与文件节点 DOM，并绑定目录箭头展开事件。

- [ ] **Step 3: Auto expand active ancestors**

确保当前文档所在目录链默认展开，并高亮当前节点。

- [ ] **Step 4: Keep existing TOC behavior intact**

保证 TOC 生成、滚动高亮和 Mermaid 逻辑不受影响。

### Task 5: Verify build and runtime integration

**Files:**
- Modify: `frontend/app.ts`
- Modify: `frontend/template.html`
- Modify: `handlers.go`
- Modify: `main.go`

- [ ] **Step 1: Run Go tests**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 2: Run frontend build**

Run: `bun run build`
Workdir: `frontend`
Expected: PASS

- [ ] **Step 3: Review final diff**

确认未覆盖现有未提交改动，且目录树逻辑与用户确认方案一致。
