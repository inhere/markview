# Files 搜索框设计文档

**日期**: 2026-04-02  
**状态**: 设计批准  
**作者**: Sisyphus  

---

## 需求概述

在 sidebar Files 标题行新增搜索框，支持实时过滤文件树，显示匹配文件及其父目录路径。

### 功能需求

| 需求项 | 描述 |
|--------|------|
| 搜索位置 | Files 文字后面、折叠按钮前面（标题行内） |
| 搜索触发 | 实时过滤（debounce 200ms） |
| 匹配显示 | 显示匹配文件 + 父目录路径 |
| 清除功能 | × 清除按钮，点击清空并恢复完整文件树 |
| 匹配规则 | 不区分大小写，部分匹配 |
| 自动展开 | 匹配项的祖先目录链自动展开 |

---

## 架构设计

### 新增组件结构

```
Files Section (sidebar-section-title)
├─ 标题部分（现有）
│  ├─ icon
│  ├─ Files 文字
│  ├─ 搜索框（新增）
│  │  ├─ input 输入框
│  │  ├─ × 清除按钮
│  └─ 折叠按钮（现有）
└─ 文件树容器（现有）
```

### 改动文件范围

| 文件 | 改动内容 | 改动量 |
|------|----------|--------|
| `web/template.html` | 添加搜索框 DOM 元素 | ~10 行 |
| `web/src/sidebar.ts` | 添加搜索逻辑函数 | ~60-80 行 |
| `web/src/style/app.css` | 添加搜索框样式 | ~20-30 行 |

### 数据流

```
用户输入 → debounce 200ms → 过滤 FileTreeNode[] → DOM 节点显示/隐藏 → 自动展开祖先链
```

---

## HTML 结构设计

### 搜索框 DOM 结构

在 `template.html` 第 107-108 行之间插入（Files 文字后的 `</span>` 和折叠按钮 `<button>` 之间）：

```html
<div class="sidebar-section-title">
  <span class="sidebar-section-label">
    <span class="sidebar-section-icon">...</span>
    <span>Files</span>
  </span>
  
  <!-- 新增：文件搜索框 -->
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
  
  <button id="files-collapse-btn">...</button>
</div>
```

### 关键属性说明

| 属性 | 作用 |
|------|------|
| `id="files-search-input"` | TypeScript 选择器入口 |
| `placeholder="搜索文件..."` | 用户引导提示（中文） |
| `autocomplete="off"` | 禁用浏览器自动完成 |
| `aria-label` | 无障碍支持 |

### 布局约束

- 搜索框宽度：自适应剩余空间（flex-grow）
- 与折叠按钮间距：8px
- 高度：28px（比标题行略小）

---

## TypeScript 搜索逻辑设计

### 核心函数设计

| 函数名 | 作用 | 参数 |
|--------|------|------|
| `initFilesSearch()` | 初始化搜索功能，绑定事件 | - |
| `filterFileTree(query: string)` | 过滤文件树，显示/隐藏节点 | 搜索关键词 |
| `findMatchingNodes(node: FileTreeNode, query: string): boolean` | 递归判断节点是否匹配 | 节点对象、关键词 |
| `expandAncestors(nodeEl: HTMLElement)` | 展开匹配项的祖先目录链 | DOM 节点元素 |
| `clearFilesSearch()` | 清除搜索，恢复完整文件树 | - |

### 搜索流程伪代码

```typescript
// 1. 初始化（在 renderFileTree 后调用）
function initFilesSearch() {
  const input = document.getElementById('files-search-input');
  const clearBtn = document.getElementById('files-search-clear');
  
  // 实时搜索（debounce 200ms）
  input.addEventListener('input', debounce((e) => {
    const query = e.target.value.trim().toLowerCase();
    if (query) {
      filterFileTree(query);
    } else {
      clearFilesSearch();
    }
  }, 200));
  
  // 清除按钮
  clearBtn.addEventListener('click', () => {
    input.value = '';
    clearFilesSearch();
  });
}

// 2. 过滤文件树
function filterFileTree(query: string) {
  const allNodes = document.querySelectorAll('.file-tree-node');
  // 从 JSON script 重新读取数据（复用 readJSONScript 工具函数）
  const fileTreeData = readJSONScript<FileTreeNode[]>('file-tree-data');
  
  allNodes.forEach(nodeEl => {
    const isMatch = findMatchingNodes(fileTreeData, query, nodeEl);
    nodeEl.classList.toggle('hidden', !isMatch);
    
    if (isMatch) {
      expandAncestors(nodeEl); // 自动展开祖先链
    }
  });
}

// 3. 递归匹配算法
function findMatchingNodes(
  node: FileTreeNode, 
  query: string,
  targetEl?: HTMLElement
): boolean {
  // 当前节点名匹配？
  if (node.name.toLowerCase().includes(query)) {
    return true;
  }
  
  // 子节点中有匹配？
  if (node.children) {
    return node.children.some(child => findMatchingNodes(child, query));
  }
  
  return false;
}

// 4. 展开祖先链
function expandAncestors(nodeEl: HTMLElement) {
  let parent = nodeEl.parentElement;
  while (parent && parent.classList.contains('file-tree-children')) {
    parent.hidden = false;  // 使用 hidden 属性，不是 collapsed 类
    const toggle = parent.previousElementSibling?.querySelector('.tree-toggle');
    if (toggle) {
      toggle.classList.add('expanded');
      toggle.setAttribute('aria-expanded', 'true');
    }
    parent = parent.parentElement?.closest('.file-tree-node')?.parentElement;
  }
}
```

### Debounce 实现

```typescript
function debounce(fn: Function, delay: number) {
  let timer: number;
  return (...args: any[]) => {
    clearTimeout(timer);
    timer = setTimeout(() => fn(...args), delay);
  };
}
```

### 匹配规则

| 规则 | 说明 |
|------|------|
| 不区分大小写 | `README.md` 和 `readme` 都匹配 |
| 部分匹配 | `util` 可匹配 `utils.ts`、`utility.js` |
| 保留父路径 | 匹配节点的所有祖先目录都显示 |

---

## CSS 样式设计

### 搜索框容器样式

```css
.files-search-box {
  flex-grow: 1;
  display: flex;
  align-items: center;
  position: relative;
  margin-left: 8px;
  margin-right: 8px;
  height: 28px;
}
```

### 输入框样式

```css
.files-search-box input {
  width: 100%;
  height: 100%;
  padding: 0 30px 0 10px;
  border: 1px solid var(--border-color, #ddd);
  border-radius: 4px;
  background: var(--bg-color, #fff);
  font-size: 13px;
  color: var(--text-color, #333);
  outline: none;
  transition: border-color 0.2s ease;
}

.files-search-box input:focus {
  border-color: var(--primary-color, #4a9eff);
  box-shadow: 0 0 0 2px rgba(74, 158, 255, 0.1);
}

.files-search-box input::placeholder {
  color: var(--placeholder-color, #999);
  font-size: 13px;
}
```

### 清除按钮样式

```css
.search-clear-btn {
  position: absolute;
  right: 6px;
  top: 50%;
  transform: translateY(-50%);
  width: 20px;
  height: 20px;
  border: none;
  background: transparent;
  color: var(--text-color, #666);
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
```

### 文件树隐藏状态

```css
.file-tree-node.hidden {
  display: none;
}

.file-tree-children.hidden {
  display: none;
}
```

### CSS 变量适配

| 变量 | 用途 | Fallback |
|------|------|----------|
| `--border-color` | 输入框边框色 | #ddd |
| `--bg-color` | 输入框背景色 | #fff |
| `--text-color` | 文字颜色 | #333 |
| `--primary-color` | 聚焦高亮色 | #4a9eff |
| `--placeholder-color` | 提示文字色 | #999 |

---

## 测试设计

### 测试场景清单

| 场景编号 | 测试场景 | 验证点 |
|----------|----------|--------|
| T1 | 空搜索输入 | 搜索框为空时，文件树完整显示 |
| T2 | 单文件匹配 | 输入 `util`，仅显示包含 util 的文件及其父目录 |
| T3 | 多文件匹配 | 输入 `.md`，显示所有 Markdown 文件 |
| T4 | 目录匹配 | 输入目录名 `docs`，显示该目录及其所有子文件 |
| T5 | 无匹配结果 | 输入不存在文件名 `zzz`，文件树全部隐藏 |
| T6 | 清除按钮 | 点击 × 按钮，搜索框清空，文件树恢复完整 |
| T7 | 实时搜索 | 输入过程中（debounce 200ms）自动触发搜索 |
| T8 | 大小写不敏感 | 输入 `README` 和 `readme` 效果一致 |
| T9 | 祖先链展开 | 匹配深层文件时，父目录自动展开 |

### 测试方法

手动测试，步骤：

1. 启动测试环境：`bun run dev`
2. 准备测试文件结构：
   ```
   docs/
     ├── api.md
     └── guide.md
   src/
     ├── utils.ts
     └── main.ts
   README.md
   CHANGELOG.md
   ```
3. 逐场景验证并记录结果

### 边界情况处理

| 边界情况 | 处理方式 |
|----------|----------|
| 特殊字符输入（`*`, `?`, `/`） | 作为普通字符串处理，不启用正则 |
| 空格输入 | Trim 后过滤，空字符串视为清除 |
| 长路径文件（10+ 层） | 递归算法确保所有祖先链展开 |
| 大文件树（100+ 文件） | Debounce 减少搜索频率，避免性能问题 |

---

## 设计约束

来自现有设计文档和架构：

| 约束项 | 说明 |
|--------|------|
| 无额外 HTTP 请求 | 复用注入的 JSON 数据源 |
| Files:TOC 高度比例 | 固定 1:2，搜索框不影响比例 |
| 避免新依赖 | 使用原生 DOM 操作 |
| 保持现有风格 | 与项目原生 TypeScript + DOM 操作风格一致 |

---

## 实现计划

详见后续实现计划文档（通过 writing-plans skill 生成）。