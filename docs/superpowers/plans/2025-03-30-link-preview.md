# Link Preview 分屏预览功能实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 Markdown 内容中的链接添加分屏预览功能，hover 显示按钮，点击后右侧 40% 面板显示目标内容。

**Architecture:** 新建 `link-preview.ts` 模块处理链接增强和面板管理，复用现有 `fetchPageSnapshot` 加载站内内容，iframe 处理站外链接。CSS 分屏布局通过替换 `content-wrapper` 为 `split-container` 实现。

**Tech Stack:** TypeScript, DOM API, fetch, iframe, CSS Flexbox

---

## File Structure

| File | Action | Responsibility |
|------|--------|----------------|
| `web/src/link-preview.ts` | Create | 核心逻辑：链接增强、面板管理、内容加载 |
| `web/src/app.ts` | Modify | 引入并调用 `setupLinkPreview()` |
| `web/template.html` | Modify | CSS 分屏布局 + 预览面板 HTML + 按钮样式 |

---

## Task 1: 创建 link-preview.ts 基础模块

**Files:**
- Create: `web/src/link-preview.ts`

- [ ] **Step 1: 创建模块骨架**

创建文件，导出 `setupLinkPreview` 入口函数，声明状态变量：

```typescript
// web/src/link-preview.ts

export function setupLinkPreview(): void {
    if (window.innerWidth < 1024) {
        return; // 移动端不启用
    }

    // 监听 ESC 键关闭面板
    document.addEventListener('keydown', handleEscapeKey);

    // 增强当前内容区域的链接
    const content = document.querySelector('#content');
    if (content instanceof HTMLElement) {
        enhanceLinksInContent(content);
    }
}

// 状态管理
let currentPreviewUrl: string | null = null;
let currentTriggerButton: HTMLElement | null = null;
let previewPanelOpen = false;

function handleEscapeKey(event: KeyboardEvent): void {
    if (event.key === 'Escape' && previewPanelOpen) {
        closePreviewPanel();
    }
}

function enhanceLinksInContent(root: HTMLElement): void {
    // TODO: 实现
}

function openPreviewPanel(url: string, triggerButton: HTMLElement): void {
    // TODO: 实现
}

function closePreviewPanel(): void {
    // TODO: 实现
}
```

- [ ] **Step 2: 验证模块可导入**

在 `web/src/app.ts` 顶部添加临时导入验证：

```typescript
import { setupLinkPreview } from './link-preview';
```

运行构建确认无语法错误：

```bash
cd web && bun run build
```

Expected: Build succeeds (可能有空函数警告，忽略)

- [ ] **Step 3: Commit**

```bash
git add web/src/link-preview.ts web/src/app.ts
git commit -m "feat(link-preview): create module skeleton with setupLinkPreview export"
```

---

## Task 2: 实现链接增强函数 enhanceLinksInContent

**Files:**
- Modify: `web/src/link-preview.ts`

- [ ] **Step 1: 实现 shouldShowPreviewButton 判断函数**

判断哪些链接应该显示预览按钮：

```typescript
// 静态资源扩展名
const STATIC_RESOURCE_EXTENSIONS = [
    '.jpg', '.jpeg', '.png', '.gif', '.svg', '.webp', '.avif',
    '.mp4', '.webm', '.mp3', '.ogg', '.wav', '.pdf', '.zip', '.tar', '.gz'
];

function shouldShowPreviewButton(anchor: HTMLAnchorElement): boolean {
    const href = anchor.getAttribute('href');
    if (!href) return false;

    // 排除锚点链接
    if (href.startsWith('#')) return false;

    // 排除静态资源
    const lowerHref = href.toLowerCase();
    for (const ext of STATIC_RESOURCE_EXTENSIONS) {
        if (lowerHref.endsWith(ext)) return false;
    }

    // 排除 download 属性
    if (anchor.hasAttribute('download')) return false;

    // 排除 target="_blank" (外站链接已有此属性)
    if (anchor.target === '_blank') {
        // 外站链接仍可预览（iframe方式）
        return true;
    }

    // 站内链接需要是 Markdown 文件
    const url = new URL(anchor.href, window.location.href);
    if (url.origin !== window.location.origin) {
        return true; // 站外链接，iframe 预览
    }

    // 站内路径：检查是否为 .md 或无扩展名
    const pathname = url.pathname;
    const lastSegment = pathname.split('/').filter(Boolean).pop() || '';

    if (lastSegment.includes('.')) {
        return lastSegment.toLowerCase().endsWith('.md');
    }

    return true; // 无扩展名的路径视为可预览
}

function isInternalLink(anchor: HTMLAnchorElement): boolean {
    const url = new URL(anchor.href, window.location.href);
    return url.origin === window.location.origin;
}
```

- [ ] **Step 2: 实现 enhanceLinksInContent 主体**

为每个符合条件的链接创建 hover 按钮：

```typescript
function enhanceLinksInContent(root: HTMLElement): void {
    const anchors = root.querySelectorAll('a[href]');

    for (const anchor of anchors) {
        if (!(anchor instanceof HTMLAnchorElement)) continue;
        if (!shouldShowPreviewButton(anchor)) continue;

        // 为链接创建包装容器（用于定位按钮）
        const wrapper = document.createElement('span');
        wrapper.className = 'link-preview-wrapper';
        anchor.parentNode?.insertBefore(wrapper, anchor);
        wrapper.appendChild(anchor);

        // 创建预览按钮
        const btn = document.createElement('button');
        btn.className = 'link-preview-btn';
        btn.innerHTML = `<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2"/><line x1="12" y1="3" x2="12" y2="21"/></svg>`;
        btn.title = '分屏预览';
        btn.type = 'button';

        // hover 显示逻辑
        wrapper.addEventListener('mouseenter', () => {
            btn.classList.add('visible');
        });
        wrapper.addEventListener('mouseleave', () => {
            btn.classList.remove('visible');
        });

        // 点击处理
        btn.addEventListener('click', (e) => {
            e.preventDefault();
            e.stopPropagation();
            openPreviewPanel(anchor.href, btn);
        });

        wrapper.appendChild(btn);
    }
}
```

- [ ] **Step 3: 验证链接增强**

临时在 `setupLinkPreview` 中添加 debug 日志：

```typescript
export function setupLinkPreview(): void {
    // ... 原有代码 ...

    const content = document.querySelector('#content');
    if (content instanceof HTMLElement) {
        enhanceLinksInContent(content);
        console.log('Link preview: enhanced links');
    }
}
```

运行服务，打开包含 Markdown 链接的页面：

```bash
cd web && bun run build && cd .. && go run ./cmd/markview
```

Expected: Console 显示 "Link preview: enhanced links"，检查 DOM 确认 `.link-preview-wrapper` 和 `.link-preview-btn` 元素存在。

- [ ] **Step 4: Commit**

```bash
git add web/src/link-preview.ts
git commit -m "feat(link-preview): implement enhanceLinksInContent with hover buttons"
```

---

## Task 3: 添加预览按钮 CSS 样式

**Files:**
- Modify: `web/template.html`

- [ ] **Step 1: 添加 .link-preview-wrapper 和 .link-preview-btn CSS**

在 template.html 的 `<style>` 区域，`.mermaid-actions` 样式附近添加：

```css
/* Link Preview Button */
.link-preview-wrapper {
    position: relative;
    display: inline;
}

.link-preview-btn {
    position: absolute;
    top: 50%;
    left: calc(100% + 4px);
    transform: translateY(-50%);
    width: 20px;
    height: 20px;
    background: white;
    border: 1px solid var(--border-light);
    border-radius: 4px;
    cursor: pointer;
    opacity: 0;
    transition: opacity 0.2s;
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 5;
    box-shadow: 0 1px 3px rgba(0,0,0,0.1);
}

.link-preview-btn.visible {
    opacity: 1;
}

.link-preview-btn:hover {
    border-color: var(--accent-primary);
    color: var(--accent-primary);
    box-shadow: 0 2px 6px rgba(0,0,0,0.15);
}

.link-preview-btn svg {
    width: 12px;
    height: 12px;
}

/* 移动端隐藏 */
@media (max-width: 1024px) {
    .link-preview-wrapper {
        display: inline;
    }
    .link-preview-btn {
        display: none;
    }
}
```

- [ ] **Step 2: 验证按钮样式**

刷新页面，hover Markdown 链接：

Expected: 预览按钮出现在链接右侧，hover 时显示，移出时隐藏。

- [ ] **Step 3: Commit**

```bash
git add web/template.html
git commit -m "feat(link-preview): add CSS for hover preview button"
```

---

## Task 4: 实现预览面板 HTML 结构

**Files:**
- Modify: `web/template.html`

- [ ] **Step 1: 添加预览面板 HTML**

在 template.html 的 `</body>` 前，`.mermaid-modal` 之后添加：

```html
<!-- Link Preview Panel -->
<div id="preview-panel" class="preview-panel" style="display: none;">
    <div class="preview-header">
        <button class="preview-close" type="button">×</button>
    </div>
    <div class="preview-content">
        <div class="preview-loading">
            <div class="loading-spinner"></div>
            <span>加载中...</span>
        </div>
        <div class="preview-body"></div>
        <div class="preview-error">
            <span>无法预览此链接</span>
        </div>
    </div>
</div>
```

- [ ] **Step 2: 添加预览面板 CSS**

在 `.link-preview-btn` CSS 后添加：

```css
/* Preview Panel */
.preview-panel {
    position: fixed;
    top: 0;
    right: 0;
    width: 40%;
    height: 100vh;
    background: var(--bg-surface);
    border-left: 1px solid var(--border-light);
    display: flex;
    flex-direction: column;
    z-index: 50;
    box-shadow: -4px 0 12px rgba(0,0,0,0.08);
}

.preview-header {
    height: 40px;
    padding: 0 12px;
    display: flex;
    align-items: center;
    justify-content: flex-end;
    border-bottom: 1px solid var(--border-light);
    background: var(--bg-canvas);
}

.preview-close {
    width: 28px;
    height: 28px;
    background: white;
    border: 1px solid var(--border-light);
    border-radius: 6px;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 18px;
    color: var(--text-muted);
    transition: all 0.15s;
}

.preview-close:hover {
    color: var(--text-heading);
    background: var(--accent-subtle);
    border-color: var(--accent-border);
}

.preview-content {
    flex: 1;
    overflow: auto;
    padding: 20px;
}

.preview-loading {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100%;
    gap: 12px;
    color: var(--text-muted);
}

.loading-spinner {
    width: 24px;
    height: 24px;
    border: 2px solid var(--border-light);
    border-top-color: var(--accent-primary);
    border-radius: 50%;
    animation: spin 1s linear infinite;
}

@keyframes spin {
    to { transform: rotate(360deg); }
}

.preview-body {
    /* 内容区域，由 JS 控制显示 */
}

.preview-body iframe {
    width: 100%;
    height: 100%;
    border: none;
}

.preview-error {
    display: none;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: var(--text-muted);
    font-size: 14px;
}

.preview-error.visible {
    display: flex;
}
```

- [ ] **Step 3: 添加分屏布局 CSS**

在 `.content-wrapper` CSS 后添加分屏状态样式：

```css
/* Split Layout (when preview is open) */
body.preview-active .content-wrapper {
    width: 60%;
    margin-right: 40%;
}

body.preview-active .paper {
    max-width: none;
}
```

- [ ] **Step 4: 验证面板结构**

刷新页面，检查 HTML 元素存在：

Expected: `#preview-panel` 元素存在，CSS 正确渲染（默认隐藏）。

- [ ] **Step 5: Commit**

```bash
git add web/template.html
git commit -m "feat(link-preview): add preview panel HTML and CSS"
```

---

## Task 5: 实现面板开关逻辑

**Files:**
- Modify: `web/src/link-preview.ts`

- [ ] **Step 1: 实现 openPreviewPanel 和 closePreviewPanel**

```typescript
function openPreviewPanel(url: string, triggerButton: HTMLElement): void {
    const panel = document.getElementById('preview-panel');
    if (!panel) return;

    // 若点击同一链接的按钮，关闭面板
    if (previewPanelOpen && currentPreviewUrl === url) {
        closePreviewPanel();
        return;
    }

    // 更新状态
    currentPreviewUrl = url;
    currentTriggerButton = triggerButton;
    previewPanelOpen = true;

    // 显示面板
    panel.style.display = 'flex';
    document.body.classList.add('preview-active');

    // 绑定关闭按钮
    const closeBtn = panel.querySelector('.preview-close');
    if (closeBtn) {
        closeBtn.onclick = closePreviewPanel;
    }

    // 重置面板状态
    resetPanelState();

    // 加载内容
    const anchor = triggerButton.previousElementSibling;
    if (anchor instanceof HTMLAnchorElement) {
        if (isInternalLink(anchor)) {
            loadInternalContent(url);
        } else {
            loadExternalContent(url);
        }
    }
}

function closePreviewPanel(): void {
    const panel = document.getElementById('preview-panel');
    if (!panel) return;

    // 隐藏面板
    panel.style.display = 'none';
    document.body.classList.remove('preview-active');

    // 清除 iframe（如果有）
    const iframe = panel.querySelector('iframe');
    if (iframe) iframe.remove();

    // 重置状态
    currentPreviewUrl = null;
    currentTriggerButton = null;
    previewPanelOpen = false;

    resetPanelState();
}

function resetPanelState(): void {
    const panel = document.getElementById('preview-panel');
    if (!panel) return;

    const loading = panel.querySelector('.preview-loading');
    const body = panel.querySelector('.preview-body');
    const error = panel.querySelector('.preview-error');

    if (loading) loading.style.display = 'flex';
    if (body) body.innerHTML = '';
    if (error) error.classList.remove('visible');
}
```

- [ ] **Step 2: 验证面板开关**

刷新页面，点击预览按钮：

Expected:
- 面板显示在右侧 40%
- body 添加 `.preview-active` 类
- 显示加载状态

再次点击同一按钮：

Expected: 面板关闭。

- [ ] **Step 3: Commit**

```bash
git add web/src/link-preview.ts
git commit -m "feat(link-preview): implement openPreviewPanel and closePreviewPanel"
```

---

## Task 6: 实现站内内容加载

**Files:**
- Modify: `web/src/link-preview.ts`

- [ ] **Step 1: 导入 page.ts 函数**

在文件顶部添加导入：

```typescript
import {
    parsePageSnapshot,
    type PageSnapshot,
} from './page';
```

- [ ] **Step 2: 实现 loadInternalContent**

复用 `fetchPageSnapshot` 模式，只渲染 `#content` 部分：

```typescript
async function loadInternalContent(url: string): Promise<void> {
    const panel = document.getElementById('preview-panel');
    if (!panel) return;

    try {
        // 构造 URL（添加 inline navigation header）
        const targetUrl = new URL(url, window.location.href);

        // fetch 页面
        const response = await fetch(targetUrl.toString(), {
            headers: { 'X-MarkView-Navigation': 'inline' },
        });

        if (!response.ok) {
            throw new Error(`Failed to fetch: ${response.status}`);
        }

        const html = await response.text();

        // 解析页面，只提取 #content
        const parser = new DOMParser();
        const doc = parser.parseFromString(html, 'text/html');
        const content = doc.querySelector('#content');

        if (!(content instanceof HTMLElement)) {
            throw new Error('Missing #content in fetched page');
        }

        // 渲染到 preview-body
        const bodyEl = panel.querySelector('.preview-body');
        const loadingEl = panel.querySelector('.preview-loading');

        if (bodyEl) {
            bodyEl.innerHTML = content.innerHTML;
            // 添加 paper 样式给预览内容
            bodyEl.style.padding = '20px';
        }
        if (loadingEl) loadingEl.style.display = 'none';

    } catch (error) {
        console.error('Internal content load failed:', error);
        showErrorState();
    }
}

function showErrorState(): void {
    const panel = document.getElementById('preview-panel');
    if (!panel) return;

    const loading = panel.querySelector('.preview-loading');
    const error = panel.querySelector('.preview-error');

    if (loading) loading.style.display = 'none';
    if (error) {
        error.classList.add('visible');
        // 3秒后自动关闭
        setTimeout(closePreviewPanel, 3000);
    }
}
```

- [ ] **Step 3: 验证站内预览**

创建测试文件：
- `test.md` 含链接 `[other](other.md)`
- `other.md` 含一些 Markdown 内容

打开 test.md，点击预览按钮：

Expected: 右侧面板显示 other.md 的 #content 内容（无 sidebar），样式正确。

- [ ] **Step 4: Commit**

```bash
git add web/src/link-preview.ts
git commit -m "feat(link-preview): implement loadInternalContent using fetchPageSnapshot"
```

---

## Task 7: 实现站外 iframe 加载

**Files:**
- Modify: `web/src/link-preview.ts`

- [ ] **Step 1: 实现 loadExternalContent**

```typescript
const IFRAME_TIMEOUT_MS = 8000;

function loadExternalContent(url: string): void {
    const panel = document.getElementById('preview-panel');
    if (!panel) return;

    const bodyEl = panel.querySelector('.preview-body');
    const loadingEl = panel.querySelector('.preview-loading');

    if (!bodyEl) return;

    // 创建 iframe
    const iframe = document.createElement('iframe');
    iframe.src = url;
    iframe.style.width = '100%';
    iframe.style.height = '100%';
    iframe.style.border = 'none';
    iframe.title = 'Preview';

    // 超时检测
    let loaded = false;
    const timeoutId = setTimeout(() => {
        if (!loaded) {
            console.warn('iframe load timeout for:', url);
            iframe.remove();
            showErrorState();
        }
    }, IFRAME_TIMEOUT_MS);

    iframe.onload = () => {
        loaded = true;
        clearTimeout(timeoutId);
        if (loadingEl) loadingEl.style.display = 'none';
    };

    iframe.onerror = () => {
        loaded = true;
        clearTimeout(timeoutId);
        iframe.remove();
        showErrorState();
    };

    bodyEl.innerHTML = '';
    bodyEl.appendChild(iframe);
}
```

- [ ] **Step 2: 验证 iframe 预览**

测试站外链接：
- 成功案例：链接到 Wikipedia 或其他允许 iframe 的站点
- 失败案例：链接到 Google 或其他设置 X-Frame-Options 的站点

Expected:
- 成功：iframe 显示目标内容
- 失败：显示"无法预览此链接"，3秒后关闭

- [ ] **Step 3: Commit**

```bash
git add web/src/link-preview.ts
git commit -m "feat(link-preview): implement loadExternalContent with iframe and timeout"
```

---

## Task 8: 集成到 app.ts

**Files:**
- Modify: `web/src/app.ts`

- [ ] **Step 1: 导入并调用 setupLinkPreview**

在 `app.ts` 顶部添加导入：

```typescript
import { setupLinkPreview } from './link-preview';
```

在 `setupOnce()` 函数中添加调用（在 `setupMermaidModal()` 之后）：

```typescript
function setupOnce() {
    if (setupCompleted) {
        return;
    }

    setupToolbar();
    setupInlineNavigation();
    setupMermaidModal();
    setupLinkPreview(); // 新增
    // ... 后续代码 ...
}
```

- [ ] **Step 2: 处理动态内容更新**

在 `renderCurrentPage()` 函数的 `enhancePageContent()` 之后添加链接增强：

```typescript
async function renderCurrentPage(options: RenderPageOptions = {}) {
    rewriteContentRelativeURLs();
    renderFileTree({
        treeRootId: 'file-tree',
        treeDataId: FILE_TREE_DATA_ID,
        currentFilePathDataId: CURRENT_FILE_PATH_DATA_ID,
    });
    generateTOC();
    await enhancePageContent();
    highlightTOC();

    // 新增：增强链接预览
    const content = document.querySelector(CONTENT_SELECTOR);
    if (content instanceof HTMLElement && window.innerWidth >= 1024) {
        enhanceLinksInContent(content);
    }

    // ... 后续代码 ...
}
```

需要在 `link-preview.ts` 中导出 `enhanceLinksInContent`：

```typescript
export function enhanceLinksInContent(root: HTMLElement): void {
    // ... 实现 ...
}
```

- [ ] **Step 3: 验证完整功能**

刷新页面，测试所有场景：

Expected:
- 页面加载后链接已增强
- inline navigation 后新页面链接也已增强
- ESC 键关闭面板
- 所有交互正常

- [ ] **Step 4: Commit**

```bash
git add web/src/link-preview.ts web/src/app.ts
git commit -m "feat(link-preview): integrate setupLinkPreview into app lifecycle"
```

---

## Task 9: QA 测试验证

**Files:**
- All modified files

- [ ] **Step 1: 创建测试 Markdown 文件**

创建以下测试文件：

```markdown
<!-- test-link-preview.md -->
# Link Preview Test

## 站内链接测试
- [其他文档](other-doc.md)
- [相对路径](./subfolder/file.md)
- [绝对路径](/docs/readme.md)

## 站外链接测试
- [Wikipedia](https://en.wikipedia.org/wiki/Markdown)
- [Google (会失败)](https://www.google.com)

## 排除测试
- [锚点链接](#section) - 应无预览按钮
- [图片链接](image.png) - 应无预览按钮
- [PDF链接](doc.pdf) - 应无预览按钮

## Section
锚点目标内容。
```

- [ ] **Step 2: 执行 QA 场景**

按 spec 文档测试场景表逐项验证：

| ID | 测试 | 结果 |
|----|------|------|
| T1 | 站内链接预览 | ✓ / ✗ |
| T2 | 站外 iframe 成功 | ✓ / ✗ |
| T3 | 站外 iframe 失败 | ✓ / ✗ |
| T4 | ESC 关闭 | ✓ / ✗ |
| T5 | 再次点击关闭 | ✓ / ✗ |
| T6 | 不同链接替换 | ✓ / ✗ |
| T7 | X 按钮关闭 | ✓ / ✗ |
| T8 | hover 显示/隐藏 | ✓ / ✗ |
| T9 | 移动端隐藏 | ✓ / ✗ |
| T10 | 锚点无预览 | ✓ / ✗ |
| T11 | 静态资源无预览 | ✓ / ✗ |
| T12 | 加载状态 | ✓ / ✗ |

- [ ] **Step 3: 修复问题**

如有测试失败，记录问题并修复。

- [ ] **Step 4: Final Commit**

```bash
git add -A
git commit -m "feat(link-preview): complete feature with QA verification"
```

---

## Notes

- **移动端处理**：`window.innerWidth < 1024` 时完全不启用功能
- **sidebar collapsed 状态**：分屏布局 CSS 使用 `body.preview-active` 控制，不影响 sidebar 折叠逻辑
- **动态内容**：每次 inline navigation 后需重新调用 `enhanceLinksInContent`
- **iframe 超时**：8秒，失败时显示友好提示
- **样式一致性**：复用 `.mermaid-actions` hover 模式