# Sidebar Resize and Collapse Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为侧边栏添加三个交互功能：(1) 点击收起为图标栏模式、(2) 拖动调整宽度、(3) Files 部分可折叠，提升阅读体验。

**Architecture:** 使用 CSS 变量控制侧边栏宽度，localStorage 存储状态，新增 sidebar-resize.ts 模块处理拖动逻辑，修改现有文件实现折叠功能。

**Tech Stack:** TypeScript, CSS Variables, localStorage, DOM Events (mousedown/mousemove/mouseup)

**Spec:** `docs/superpowers/specs/2026-03-30-sidebar-resize-and-collapse-design.md`

---

### Task 1: CSS Variables and Storage Keys Setup

**Files:**
- Modify: `frontend/template.html:18-45` (CSS variables)
- Modify: `frontend/src/preferences.ts` (storage keys)

- [ ] **Step 1: Add CSS variables for sidebar width**

在 `:root` 中添加：
```css
--sidebar-collapsed-width: 48px;
```

已有 `--sidebar-width: 280px`，将作为 JS 动态更新使用。

- [ ] **Step 2: Add storage keys to preferences.ts**

添加：
```typescript
export const SIDEBAR_COLLAPSED_STORAGE_KEY = 'markview:sidebar-collapsed';
export const SIDEBAR_WIDTH_STORAGE_KEY = 'markview:sidebar-width';
export const FILES_COLLAPSED_STORAGE_KEY = 'markview:files-collapsed';

export const DEFAULT_SIDEBAR_WIDTH = 280;
export const MIN_SIDEBAR_WIDTH = 200;
export const MAX_SIDEBAR_WIDTH = 400;
```

- [ ] **Step 3: Add helper functions for sidebar preferences**

添加读取和存储函数：
```typescript
export function normalizeSidebarWidth(value: string | null | undefined): number {
  if (!value) return DEFAULT_SIDEBAR_WIDTH;
  const parsed = Number.parseInt(value, 10);
  if (Number.isNaN(parsed)) return DEFAULT_SIDEBAR_WIDTH;
  return Math.min(MAX_SIDEBAR_WIDTH, Math.max(MIN_SIDEBAR_WIDTH, parsed));
}

export function normalizeSidebarCollapsed(value: string | null | undefined): boolean {
  return value === 'true';
}

export function readSidebarPreferences(storage: StorageReader = window.localStorage) {
  try {
    return {
      sidebarWidth: normalizeSidebarWidth(storage.getItem(SIDEBAR_WIDTH_STORAGE_KEY)),
      sidebarCollapsed: normalizeSidebarCollapsed(storage.getItem(SIDEBAR_COLLAPSED_STORAGE_KEY)),
      filesCollapsed: normalizeSidebarCollapsed(storage.getItem(FILES_COLLAPSED_STORAGE_KEY)),
    };
  } catch {
    return {
      sidebarWidth: DEFAULT_SIDEBAR_WIDTH,
      sidebarCollapsed: false,
      filesCollapsed: false,
    };
  }
}

export function persistSidebarWidth(value: number, storage: StorageWriter = window.localStorage) {
  try {
    storage.setItem(SIDEBAR_WIDTH_STORAGE_KEY, String(value));
  } catch {}
}

export function persistSidebarCollapsed(value: boolean, storage: StorageWriter = window.localStorage) {
  try {
    storage.setItem(SIDEBAR_COLLAPSED_STORAGE_KEY, String(value));
  } catch {}
}

export function persistFilesCollapsed(value: boolean, storage: StorageWriter = window.localStorage) {
  try {
    storage.setItem(FILES_COLLAPSED_STORAGE_KEY, String(value));
  } catch {}
}
```

- [ ] **Step 4: Run frontend build**

Run: `bun run build`
Workdir: `frontend`
Expected: PASS

---

### Task 2: Sidebar Collapse CSS Styles

**Files:**
- Modify: `frontend/template.html:139-190` (sidebar CSS)

- [ ] **Step 1: Add sidebar collapsed state styles**

在 `.sidebar` 样式后添加：
```css
.sidebar.sidebar-collapsed {
  width: var(--sidebar-collapsed-width);
}

.sidebar-collapsed .sidebar-panels {
  display: none;
}

.sidebar-collapsed .sidebar-icons {
  display: flex;
}

.sidebar-icons {
  display: none;
  flex-direction: column;
  gap: 8px;
  padding: 12px 10px;
  margin-top: auto;
}

.sidebar-icon-btn {
  width: 28px;
  height: 28px;
  border: 1px solid transparent;
  background: transparent;
  border-radius: 6px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-muted);
  transition: all 0.15s;
}

.sidebar-icon-btn:hover {
  background: var(--accent-subtle);
  color: var(--accent-primary);
  border-color: var(--accent-border);
}

.sidebar-icon-btn svg {
  width: 16px;
  height: 16px;
}
```

- [ ] **Step 2: Add collapse button styles**

```css
.sidebar-collapse-btn {
  width: 24px;
  height: 24px;
  border: none;
  background: transparent;
  border-radius: 4px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-muted);
  transition: all 0.15s;
  margin-left: auto;
}

.sidebar-collapse-btn:hover {
  background: var(--bg-canvas);
  color: var(--text-heading);
}

.sidebar-collapse-btn svg {
  width: 16px;
  height: 16px;
  transition: transform 0.15s;
}

.sidebar-collapsed .sidebar-collapse-btn svg {
  transform: rotate(180deg);
}
```

- [ ] **Step 3: Add resize handle styles**

```css
.sidebar-resize-handle {
  position: absolute;
  top: 0;
  right: 0;
  width: 4px;
  height: 100%;
  cursor: col-resize;
  background: transparent;
  transition: background 0.15s;
}

.sidebar-resize-handle:hover {
  background: var(--border-focus);
}

.sidebar-resize-handle.is-resizing {
  background: var(--accent-primary);
}

.sidebar.sidebar-collapsed .sidebar-resize-handle {
  display: none;
}
```

- [ ] **Step 4: Add Files collapse button styles**

```css
.sidebar-section-collapse-btn {
  width: 20px;
  height: 20px;
  border: none;
  background: transparent;
  border-radius: 4px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-muted);
  transition: all 0.15s;
}

.sidebar-section-collapse-btn:hover {
  background: var(--bg-canvas);
  color: var(--text-heading);
}

.sidebar-section-collapse-btn svg {
  width: 14px;
  height: 14px;
  transition: transform 0.15s;
}

.files-collapsed .sidebar-section-collapse-btn svg {
  transform: rotate(-90deg);
}

.files-collapsed .sidebar-scroll {
  display: none;
}
```

- [ ] **Step 5: Update sidebar CSS for position relative**

为 `.sidebar` 添加 `position: relative` 以支持 resize handle：
```css
.sidebar {
  ...
  position: fixed;
  top: 0;
  left: 0;
  z-index: 10;
  /* Add this: */
  transition: width 0.15s ease;
}
```

- [ ] **Step 6: Run frontend build**

Run: `bun run build`
Workdir: `frontend`
Expected: PASS

---

### Task 3: Add HTML Structure for Collapse and Resize

**Files:**
- Modify: `frontend/template.html:942-982` (sidebar HTML)

- [ ] **Step 1: Add collapse button to sidebar header**

修改 `.sidebar-header`：
```html
<div class="sidebar-header">
    <div class="doc-meta">
        <div class="brand-lockup" aria-label="MarkView">
            <img src="/static/logo.svg" alt="" class="brand-logo">
            <span class="brand-wordmark">MarkView</span>
        </div>
        <span class="brand-divider" aria-hidden="true"></span>
        <div class="doc-status">
        <div class="live-dot" id="live-dot"></div>
        <span id="status-text">Live</span>
        </div>
    </div>
    <button class="sidebar-collapse-btn" id="sidebar-collapse-btn" title="Collapse sidebar" aria-label="Collapse sidebar">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="15 18 9 12 15 6"></polyline></svg>
    </button>
</div>
```

- [ ] **Step 2: Add resize handle**

在 `.sidebar` 开始标签后添加：
```html
<aside class="sidebar">
    <div class="sidebar-resize-handle" id="sidebar-resize-handle"></div>
    ...
```

- [ ] **Step 3: Add sidebar icons for collapsed state**

在 `.sidebar-panels` 后添加：
```html
<div class="sidebar-icons" id="sidebar-icons">
    <button class="sidebar-icon-btn" data-panel="files" title="Files">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path><polyline points="14 2 14 8 20 8"></polyline></svg>
    </button>
    <button class="sidebar-icon-btn" data-panel="toc" title="Table of Contents">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round"><line x1="8" y1="6" x2="20" y2="6"></line><line x1="8" y1="12" x2="20" y2="12"></line><line x1="8" y1="18" x2="20" y2="18"></line><circle cx="4" cy="6" r="1.2" fill="currentColor" stroke="none"></circle><circle cx="4" cy="12" r="1.2" fill="currentColor" stroke="none"></circle><circle cx="4" cy="18" r="1.2" fill="currentColor" stroke="none"></circle></svg>
    </button>
</div>
```

- [ ] **Step 4: Add Files collapse button**

修改 Files panel 的 `.sidebar-section-title`：
```html
<div class="sidebar-section-title">
    <span class="sidebar-section-label">
        <span class="sidebar-section-icon" aria-hidden="true">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path><polyline points="14 2 14 8 20 8"></polyline></svg>
        </span>
        <span>Files</span>
    </span>
    <button class="sidebar-section-collapse-btn" id="files-collapse-btn" title="Collapse Files" aria-label="Collapse Files section">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="6 9 12 15 18 9"></polyline></svg>
    </button>
</div>
```

- [ ] **Step 5: Add id to Files panel for collapse targeting**

修改 Files panel：
```html
<section class="sidebar-panel sidebar-panel-files" id="files-panel">
```

- [ ] **Step 6: Run frontend build**

Run: `bun run build`
Workdir: `frontend`
Expected: PASS

---

### Task 4: Create sidebar-resize.ts Module

**Files:**
- Create: `frontend/src/sidebar-resize.ts`

- [ ] **Step 1: Create the resize module**

```typescript
import {
    MIN_SIDEBAR_WIDTH,
    MAX_SIDEBAR_WIDTH,
    DEFAULT_SIDEBAR_WIDTH,
    persistSidebarWidth,
} from './preferences';

const SIDEBAR_WIDTH_VAR = '--sidebar-width';
const RESIZE_HANDLE_ID = 'sidebar-resize-handle';
const SIDEBAR_SELECTOR = '.sidebar';

let isResizing = false;
let startX = 0;
let startWidth = 0;

export function initSidebarResize() {
    const handle = document.getElementById(RESIZE_HANDLE_ID);
    if (!handle) return;

    handle.addEventListener('mousedown', startResize);
    document.addEventListener('mousemove', doResize);
    document.addEventListener('mouseup', endResize);
}

function startResize(event: MouseEvent) {
    if (event.button !== 0) return;
    
    isResizing = true;
    startX = event.clientX;
    
    const sidebar = document.querySelector(SIDEBAR_SELECTOR);
    if (sidebar) {
        const rect = sidebar.getBoundingClientRect();
        startWidth = rect.width;
    }
    
    const handle = document.getElementById(RESIZE_HANDLE_ID);
    if (handle) {
        handle.classList.add('is-resizing');
    }
    
    document.body.style.cursor = 'col-resize';
    document.body.style.userSelect = 'none';
    
    event.preventDefault();
}

function doResize(event: MouseEvent) {
    if (!isResizing) return;
    
    const deltaX = startX - event.clientX;
    const newWidth = Math.max(MIN_SIDEBAR_WIDTH, Math.min(MAX_SIDEBAR_WIDTH, startWidth - deltaX));
    
    document.documentElement.style.setProperty(SIDEBAR_WIDTH_VAR, `${newWidth}px`);
}

function endResize() {
    if (!isResizing) return;
    
    isResizing = false;
    
    const handle = document.getElementById(RESIZE_HANDLE_ID);
    if (handle) {
        handle.classList.remove('is-resizing');
    }
    
    document.body.style.cursor = '';
    document.body.style.userSelect = '';
    
    const currentWidth = getCurrentSidebarWidth();
    persistSidebarWidth(currentWidth);
}

export function setSidebarWidth(width: number) {
    const clampedWidth = Math.max(MIN_SIDEBAR_WIDTH, Math.min(MAX_SIDEBAR_WIDTH, width));
    document.documentElement.style.setProperty(SIDEBAR_WIDTH_VAR, `${clampedWidth}px`);
    persistSidebarWidth(clampedWidth);
}

export function getCurrentSidebarWidth(): number {
    const value = document.documentElement.style.getPropertyValue(SIDEBAR_WIDTH_VAR);
    if (!value) return DEFAULT_SIDEBAR_WIDTH;
    
    const parsed = Number.parseInt(value.replace('px', ''), 10);
    if (Number.isNaN(parsed)) return DEFAULT_SIDEBAR_WIDTH;
    
    return parsed;
}

export function applyInitialSidebarWidth(width: number) {
    document.documentElement.style.setProperty(SIDEBAR_WIDTH_VAR, `${width}px`);
}
```

- [ ] **Step 2: Export from sidebar.ts (optional)**

不需要额外导出，sidebar-resize.ts 将独立导入到 app.ts。

- [ ] **Step 3: Run frontend build**

Run: `bun run build`
Workdir: `frontend`
Expected: PASS

---

### Task 5: Add Sidebar Collapse Logic

**Files:**
- Modify: `frontend/src/sidebar.ts`

- [ ] **Step 1: Import preferences**

```typescript
import {
    persistSidebarCollapsed,
    persistFilesCollapsed,
    readSidebarPreferences,
} from './preferences';
```

- [ ] **Step 2: Add chevron icons to util.ts**

在 `util.ts` 中添加：
```typescript
export function chevronLeftIcon() {
    return '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="15 18 9 12 15 6"></polyline></svg>';
}

export function chevronDownIcon() {
    return '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="6 9 12 15 18 9"></polyline></svg>';
}
```

- [ ] **Step 3: Add sidebar collapse setup function**

在 `sidebar.ts` 中添加：
```typescript
export function setupSidebarCollapse() {
    const collapseBtn = document.getElementById('sidebar-collapse-btn');
    const sidebar = document.querySelector('.sidebar');
    const filesPanel = document.getElementById('files-panel');
    
    if (!collapseBtn || !sidebar || !filesPanel) return;
    
    const prefs = readSidebarPreferences();
    
    // Apply initial state
    if (prefs.sidebarCollapsed) {
        sidebar.classList.add('sidebar-collapsed');
    }
    if (prefs.filesCollapsed) {
        filesPanel.classList.add('files-collapsed');
    }
    
    // Collapse button click
    collapseBtn.addEventListener('click', () => {
        const isCollapsed = sidebar.classList.toggle('sidebar-collapsed');
        persistSidebarCollapsed(isCollapsed);
        
        // Update aria-label
        collapseBtn.setAttribute('aria-label', isCollapsed ? 'Expand sidebar' : 'Collapse sidebar');
        collapseBtn.setAttribute('title', isCollapsed ? 'Expand sidebar' : 'Collapse sidebar');
    });
    
    // Files collapse button
    const filesCollapseBtn = document.getElementById('files-collapse-btn');
    if (filesCollapseBtn) {
        filesCollapseBtn.addEventListener('click', () => {
            const isCollapsed = filesPanel.classList.toggle('files-collapsed');
            persistFilesCollapsed(isCollapsed);
            
            filesCollapseBtn.setAttribute('aria-label', isCollapsed ? 'Expand Files section' : 'Collapse Files section');
            filesCollapseBtn.setAttribute('title', isCollapsed ? 'Expand Files' : 'Collapse Files');
        });
    }
    
    // Sidebar icon buttons (for collapsed state)
    const iconButtons = document.querySelectorAll('.sidebar-icon-btn');
    iconButtons.forEach(btn => {
        btn.addEventListener('click', () => {
            // Expand sidebar
            sidebar.classList.remove('sidebar-collapsed');
            persistSidebarCollapsed(false);
            
            collapseBtn.setAttribute('aria-label', 'Collapse sidebar');
            collapseBtn.setAttribute('title', 'Collapse sidebar');
            
            // If Files button clicked, ensure Files is expanded
            const panel = btn.getAttribute('data-panel');
            if (panel === 'files') {
                filesPanel.classList.remove('files-collapsed');
                persistFilesCollapsed(false);
            }
        });
    });
}
```

- [ ] **Step 4: Run frontend build**

Run: `bun run build`
Workdir: `frontend`
Expected: PASS

---

### Task 6: Integrate into app.ts

**Files:**
- Modify: `frontend/src/app.ts`

- [ ] **Step 1: Import new modules**

添加导入：
```typescript
import {
    readSidebarPreferences,
} from './preferences';
import {
    setupSidebarCollapse,
} from './sidebar';
import {
    applyInitialSidebarWidth,
    initSidebarResize,
} from './sidebar-resize';
```

- [ ] **Step 2: Add sidebar setup to setupOnce**

在 `setupOnce()` 函数中添加：
```typescript
function setupOnce() {
    if (setupCompleted) {
        return;
    }

    setupToolbar();
    setupInlineNavigation();
    setupMermaidModal();
    
    // Sidebar collapse and resize
    const sidebarPrefs = readSidebarPreferences();
    applyInitialSidebarWidth(sidebarPrefs.sidebarWidth);
    initSidebarResize();
    setupSidebarCollapse();
    
    window.addEventListener('scroll', () => {
        highlightTOC();
    }, { passive: true });

    const evtSource = new EventSource('/sse');
    const liveDot = document.getElementById('live-dot');
    const statusText = document.getElementById('status-text');
    setupLiveReloadStatus(evtSource, liveDot, statusText, refreshCurrentPage);

    setupCompleted = true;
}
```

- [ ] **Step 3: Run frontend build**

Run: `bun run build`
Workdir: `frontend`
Expected: PASS

---

### Task 7: Verify End-to-End Behavior

**Files:**
- All modified files

- [ ] **Step 1: Run frontend build**

Run: `bun run build`
Workdir: `frontend`
Expected: PASS (no TypeScript errors)

- [ ] **Step 2: Manual browser verification**

验证功能：
- 侧边栏折叠按钮点击可收起/展开
- 收起状态下显示图标按钮，点击可展开
- 拖动把手可调整侧边栏宽度
- Files 折叠按钮可折叠/展开文件列表
- 刷新页面后状态恢复（收起状态、宽度、Files 折叠）

- [ ] **Step 3: Verify responsive behavior**

在小屏幕下测试（如果需要）：
- 确认拖动把手在合理范围内工作
- 确认收起状态不会导致布局问题

- [ ] **Step 4: Commit changes**

```bash
git add frontend/src/sidebar-resize.ts frontend/src/sidebar.ts frontend/src/preferences.ts frontend/src/app.ts frontend/src/util.ts frontend/template.html
git commit -m "feat: add sidebar collapse, resize, and files collapse features"
```