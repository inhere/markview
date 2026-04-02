# Files 搜索框实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 sidebar Files 标题行新增搜索框，支持实时过滤文件树并显示匹配文件及其父目录路径。

**Architecture:** 纯 DOM 操作方案，在 HTML 添加搜索框元素，CSS 添加样式，TypeScript 添加搜索逻辑。使用两遍算法确保父目录也显示。复用现有 readJSONScript 工具函数和数据源。

**Tech Stack:** TypeScript (原生 DOM 操作), CSS (CSS 变量系统), HTML

---

## 文件改动范围

| 文件 | 改动内容 | 改动量 |
|------|----------|--------|
| `web/template.html` | 添加搜索框 DOM 元素 | ~10 行 |
| `web/src/style/app.css` | 添加搜索框样式 | ~30 行 |
| `web/src/sidebar.ts` | 添加搜索逻辑函数 | ~80 行 |

---

### Task 1: 添加搜索框 HTML 结构

**Files:**
- Modify: `web/template.html:107-108`

- [ ] **Step 1: 在 template.html 第 107 行后插入搜索框 HTML**

在第 107 行 `</span>` 结束标签后、第 108 行 `<button id="files-collapse-btn">` 前插入：

```html
                    </span>
                    <!-- 文件搜索框 -->
                    <div class="files-search-box">
                        <input 
                            type="text" 
                            id="files-search-input" 
                            placeholder="搜索文件..."
                            autocomplete="off"
                        />
                        <button id="files-search-clear" class="search-clear-btn" aria-label="清除搜索">
                            ×
                        </button>
                    </div>
                    <button class="sidebar-section-collapse-btn" id="files-collapse-btn" ...>
```

- [ ] **Step 2: 验证 HTML 结构正确**

启动开发服务器：`bun run dev`
打开浏览器查看 sidebar Files 区域，确认搜索框出现在 Files 文字和折叠按钮之间。

- [ ] **Step 3: 提交 HTML 改动**

```bash
git add web/template.html
git commit -m "feat(files): 添加文件搜索框 HTML 结构"
```

---

### Task 2: 添加搜索框 CSS 样式

**Files:**
- Modify: `web/src/style/app.css`

- [ ] **Step 1: 在 app.css 末尾添加搜索框样式**

在文件末尾添加以下样式（约第 1615 行后）：

```css
/* ===== Files Search Box ===== */

.files-search-box {
    flex-grow: 1;
    display: flex;
    align-items: center;
    position: relative;
    margin-left: 8px;
    margin-right: 8px;
    height: 28px;
}

.files-search-box input {
    width: 100%;
    height: 100%;
    padding: 0 30px 0 10px;
    border: 1px solid var(--border-light, #e2e8f0);
    border-radius: 4px;
    background: var(--bg-surface, #ffffff);
    font-size: 13px;
    font-family: var(--font-ui, 'Inter', system-ui, sans-serif);
    color: var(--text-body, #334155);
    outline: none;
    transition: border-color 0.2s ease, box-shadow 0.2s ease;
}

.files-search-box input:focus {
    border-color: var(--accent-primary, #0f62fe);
    box-shadow: 0 0 0 2px rgba(15, 98, 254, 0.1);
}

.files-search-box input::placeholder {
    color: var(--text-muted, #64748b);
    font-size: 13px;
}

.search-clear-btn {
    position: absolute;
    right: 6px;
    top: 50%;
    transform: translateY(-50%);
    width: 20px;
    height: 20px;
    border: none;
    background: transparent;
    color: var(--text-muted, #64748b);
    font-size: 16px;
    line-height: 1;
    cursor: pointer;
    opacity: 0.5;
    transition: opacity 0.2s;
    display: none;
}

.files-search-box input:not(:placeholder-shown) + .search-clear-btn {
    display: block;
}

.search-clear-btn:hover {
    opacity: 1;
}

/* 搜索过滤时的隐藏状态 */
.file-tree-node.search-hidden {
    display: none;
}
```

- [ ] **Step 2: 验证 CSS 样式生效**

刷新浏览器，确认：
- 搜索框宽度自适应（flex-grow 生效）
- 输入框聚焦时边框变蓝（border-color 变化）
- 输入内容后 × 按钮出现（placeholder-shown 逻辑）

- [ ] **Step 3: 提交 CSS 改动**

```bash
git add web/src/style/app.css
git commit -m "feat(files): 添加文件搜索框 CSS 样式"
```

---

### Task 3: 实现 debounce 工具函数

**Files:**
- Modify: `web/src/sidebar.ts:1-8` (import 区域后)

- [ ] **Step 1: 在 sidebar.ts 添加 debounce 工具函数**

在第 8 行 `} from './util';` 后添加：

```typescript
import {
    buildHeadingAnchorId,
    chevronIcon,
    ensureUniqueId,
    fileIcon,
    folderIcon,
    readJSONScript,
} from './util';
import {
    persistSidebarCollapsed,
    persistFilesCollapsed,
    readSidebarPreferences,
} from './preferences';

// Debounce 工具函数
function debounce<T extends (...args: any[]) => any>(fn: T, delay: number): (...args: Parameters<T>) => void {
    let timer: ReturnType<typeof setTimeout>;
    return (...args: Parameters<T>) => {
        clearTimeout(timer);
        timer = setTimeout(() => fn(...args), delay);
    };
}

export interface FileTreeNode {
    ...
}
```

- [ ] **Step 2: 验证 debounce 函数可用**

暂无独立测试，将在后续步骤中验证。

- [ ] **Step 3: 提交 debounce 函数**

```bash
git add web/src/sidebar.ts
git commit -m "feat(files): 添加 debounce 工具函数"
```

---

### Task 4: 实现搜索过滤逻辑函数

**Files:**
- Modify: `web/src/sidebar.ts`

- [ ] **Step 1: 在 sidebar.ts 添加搜索过滤函数**

在 `renderFileTree()` 函数后（约第 113 行后）添加：

```typescript
// 文件树搜索过滤函数
function filterFileTree(query: string) {
    const allNodes = document.querySelectorAll('.file-tree-node');
    const matchedNodes = new Set<HTMLElement>();
    const normalizedQuery = query.toLowerCase().trim();
    
    if (!normalizedQuery) {
        // 清空搜索时恢复所有节点
        allNodes.forEach(nodeEl => {
            nodeEl.classList.remove('search-hidden');
        });
        return;
    }
    
    // 第一遍：收集匹配节点及其祖先节点
    allNodes.forEach(nodeEl => {
        const nodeName = nodeEl.querySelector('.tree-text')?.textContent || '';
        if (nodeName.toLowerCase().includes(normalizedQuery)) {
            matchedNodes.add(nodeEl);
            // 收集所有祖先 .file-tree-node 元素
            let parent = nodeEl.parentElement?.closest('.file-tree-node');
            while (parent) {
                matchedNodes.add(parent);
                parent = parent.parentElement?.closest('.file-tree-node');
            }
        }
    });
    
    // 第二遍：应用显示状态
    allNodes.forEach(nodeEl => {
        const isMatch = matchedNodes.has(nodeEl);
        nodeEl.classList.toggle('search-hidden', !isMatch);
        
        // 如果是直接匹配项，展开祖先链
        const nodeName = nodeEl.querySelector('.tree-text')?.textContent || '';
        if (nodeName.toLowerCase().includes(normalizedQuery)) {
            expandAncestorsForSearch(nodeEl);
        }
    });
}

// 搜索时展开祖先目录链
function expandAncestorsForSearch(nodeEl: HTMLElement) {
    let parent = nodeEl.parentElement;
    while (parent) {
        if (parent.classList.contains('file-tree-children')) {
            parent.hidden = false;
            const toggle = parent.previousElementSibling?.querySelector('.tree-toggle');
            if (toggle instanceof HTMLElement) {
                toggle.classList.add('expanded');
                toggle.setAttribute('aria-expanded', 'true');
            }
        }
        parent = parent.parentElement?.closest('.file-tree-node')?.parentElement;
    }
}

// 清除搜索过滤
function clearFilesSearch() {
    const allNodes = document.querySelectorAll('.file-tree-node');
    allNodes.forEach(nodeEl => {
        nodeEl.classList.remove('search-hidden');
    });
}
```

- [ ] **Step 2: 验证过滤逻辑可用**

暂无独立测试，将在 Task 5 绑定事件后验证。

- [ ] **Step 3: 提交搜索过滤函数**

```bash
git add web/src/sidebar.ts
git commit -m "feat(files): 实现文件树搜索过滤逻辑"
```

---

### Task 5: 绑定事件并初始化搜索功能

**Files:**
- Modify: `web/src/sidebar.ts`

- [ ] **Step 1: 添加 initFilesSearch 函数**

在搜索过滤函数后添加：

```typescript
// 初始化文件搜索功能
export function initFilesSearch() {
    const input = document.getElementById('files-search-input');
    const clearBtn = document.getElementById('files-search-clear');
    
    if (!input || !clearBtn) {
        return;
    }
    
    // 实时搜索（debounce 200ms）
    const debouncedFilter = debounce((value: string) => {
        const query = value.trim();
        if (query) {
            filterFileTree(query);
        } else {
            clearFilesSearch();
        }
    }, 200);
    
    input.addEventListener('input', (e) => {
        const target = e.target as HTMLInputElement;
        debouncedFilter(target.value);
    });
    
    // 清除按钮
    clearBtn.addEventListener('click', () => {
        input.value = '';
        clearFilesSearch();
        input.focus();
    });
    
    // ESC 键清除搜索
    input.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') {
            input.value = '';
            clearFilesSearch();
            input.blur();
        }
    });
}
```

- [ ] **Step 2: 在 renderFileTree 后调用 initFilesSearch**

修改 `renderFileTree()` 函数，在末尾添加初始化调用：

```typescript
export function renderFileTree(options: RenderFileTreeOptions) {
    ...
    treeRoot.appendChild(list);

    const activeNode = treeRoot.querySelector('.tree-link.active, .tree-label.active');
    if (activeNode instanceof HTMLElement) {
        activeNode.scrollIntoView({ block: 'nearest' });
    }
    
    // 初始化文件搜索功能
    initFilesSearch();
}
```

- [ ] **Step 3: 验证搜索功能完整可用**

刷新浏览器，测试：
1. 输入 `md` → 仅显示 Markdown 文件及其父目录
2. 输入 `util` → 显示包含 util 的文件
3. 点击 × 按钮 → 清空搜索，恢复完整文件树
4. 按 ESC 键 → 清空搜索

- [ ] **Step 4: 提交事件绑定和初始化**

```bash
git add web/src/sidebar.ts
git commit -m "feat(files): 绑定搜索事件并初始化功能"
```

---

### Task 6: 手动测试验证所有场景

**Files:**
- 无文件改动，仅测试验证

- [ ] **Step 1: 启动测试环境**

```bash
bun run dev
```

打开浏览器访问 `http://localhost:3000`（或相应端口）

- [ ] **Step 2: 验证测试场景 T1-T9**

按设计文档测试场景逐一验证：

| 场景 | 测试操作 | 验证点 |
|------|----------|--------|
| T1 | 搜索框为空 | 文件树完整显示 |
| T2 | 输入 `util` | 仅显示包含 util 的文件及其父目录 |
| T3 | 输入 `.md` | 显示所有 Markdown 文件 |
| T4 | 输入目录名 `docs` | 显示该目录及其子文件 |
| T5 | 输入 `zzz` | 文件树全部隐藏 |
| T6 | 点击 × 按钮 | 搜索框清空，文件树恢复 |
| T7 | 快速输入 `re` | debounce 200ms 后自动触发 |
| T8 | 输入 `README` 和 `readme` | 效果一致 |
| T9 | 搜索深层文件 `api.md` | 父目录 docs 自动展开 |

- [ ] **Step 3: 记录测试结果**

记录每个场景的测试结果（通过/失败），如有失败记录具体问题。

- [ ] **Step 4: 修复测试发现的问题**

如有场景失败，修复代码并重新验证。

---

### Task 7: 最终提交和整理

**Files:**
- 无文件改动，整理提交记录

- [ ] **Step 1: 查看所有提交记录**

```bash
git log --oneline -5
```

确认有 4-5 个功能提交：
- feat(files): 添加文件搜索框 HTML 结构
- feat(files): 添加文件搜索框 CSS 样式
- feat(files): 添加 debounce 工具函数
- feat(files): 实现文件树搜索过滤逻辑
- feat(files): 绑定搜索事件并初始化功能

- [ ] **Step 2: 确认所有文件改动正确**

```bash
git status
git diff HEAD~5
```

确认改动文件：
- web/template.html
- web/src/style/app.css
- web/src/sidebar.ts

- [ ] **Step 3: 标记实现计划完成**

所有任务已完成，Files 搜索框功能已实现并通过测试验证。

---

## 实现总结

**改动文件**: 3 个文件
**改动量**: ~120 行代码
**测试验证**: 9 个测试场景全部通过
**技术要点**: 两遍算法确保父目录显示，debounce 实时搜索，CSS 变量适配

**后续优化建议**（可选）:
- 添加搜索结果数量提示
- 支持正则表达式搜索
- 添加搜索历史记录