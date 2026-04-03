# 全局内容搜索设计文档

**日期**: 2026-04-03  
**状态**: 设计批准  
**作者**: Sisyphus  

---

## 需求概述

在内容区域顶部左侧添加搜索框，支持搜索所有 md 文件的**内容**（跨文件全文搜索），显示匹配结果并支持跳转。

### 功能需求

| 需求项 | 描述 |
|--------|------|
| 搜索位置 | `.paper` 内容区域顶部左侧 |
| 搜索范围 | 整个项目中所有 md 文件的内容 |
| 搜索触发 | debounce 300ms 后调用后端 API |
| 结果展示 | 浮动面板显示匹配文件 + 内容片段 |
| 匹配高亮 | 结果片段中高亮关键词 |
| 导航跳转 | 点击结果 → 内联导航到目标文件 |
| 清除功能 | × 清除按钮 + ESC 键关闭面板 |

---

## 技术方案

**方案 A：后端搜索 API + 前端结果面板**

- 后端新增 `/api/search?q=keyword` API
- 前端调用 API 并渲染结果面板
- 使用现有内联导航机制跳转

---

## 后端 API 设计

### 端点

**新增路由**: `GET /api/search?q={keyword}`

### 请求参数

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `q` | string | 是 | - | 搜索关键词（至少 2 字符） |
| `limit` | int | 否 | 5 | 每文件最大匹配数 |

### 响应格式

```json
{
  "query": "关键词",
  "totalFiles": 12,
  "totalMatches": 45,
  "results": [
    {
      "file": "docs/guide.md",
      "href": "/docs/guide.md",
      "matches": [
        {
          "line": 23,
          "snippet": "...包含 **关键词** 的上下文...",
          "contextBefore": "前面的内容",
          "contextAfter": "后面的内容"
        }
      ]
    }
  ]
}
```

### 搜索逻辑

1. 验证参数（关键词长度 ≥ 2）
2. 遍历 `targetDir` 下所有 `.md` 文件
3. 读取文件内容，逐行匹配（使用 `strings.ToLower()` 转换后匹配，实现不区分大小写）
4. 提取匹配行号 + 前后 50 字符上下文
5. 返回 JSON 响应

**后端实现细节**（`handlers.go`）：

```go
func handleSearch(w http.ResponseWriter, r *http.Request) {
    query := r.URL.Query().Get("q")
    if len(query) < 2 {
        http.Error(w, "Query too short", 400)
        return
    }

    limit := 5
    if l := r.URL.Query().Get("limit"); l != "" {
        if n, err := strconv.Atoi(l); err == nil && n > 0 {
            limit = n
        }
    }

    // 不区分大小写搜索
    queryLower := strings.ToLower(query)
    results := []SearchResult{}

    // 遍历所有 md 文件
    err := filepath.WalkDir(targetDir, func(path string, d fs.DirEntry, err error) error {
        if err != nil || d.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".md") {
            return nil
        }

        content, err := os.ReadFile(path)
        if err != nil {
            return nil
        }

        relPath, _ := filepath.Rel(targetDir, path)
        matches := searchInContent(string(content), queryLower, limit)
        if len(matches) > 0 {
            results = append(results, SearchResult{
                File:    relPath,
                Href:    "/" + strings.ReplaceAll(relPath, "\\", "/"),
                Matches: matches,
            })
        }
        return nil
    })

    // 返回 JSON 响应
    json.NewEncoder(w).Encode(SearchResponse{
        Query:        query,
        TotalFiles:   len(results),
        TotalMatches: countMatches(results),
        Results:      results,
    })
}

func searchInContent(content, queryLower string, limit int) []SearchMatch {
    lines := strings.Split(content, "\n")
    matches := []SearchMatch{}

    for i, line := range lines {
        if strings.Contains(strings.ToLower(line), queryLower) {
            matches = append(matches, SearchMatch{
                Line:    i + 1,
                Snippet: extractSnippet(line, queryLower),
            })
            if len(matches) >= limit {
                break
            }
        }
    }
    return matches
}
```

### 文件改动

| 文件 | 改动 |
|------|------|
| `handlers.go` | 新增 `handleSearch()` 函数 (~80 行) |
| `main.go` | 新增路由 `http.HandleFunc("/api/search", handleSearch)` |

---

## 前端 UI 设计

### HTML 结构

**修改文件**: `web/template-main.html`

```html
<!-- 在 article.paper 之前添加 -->
<div class="content-search-bar">
    <form class="content-search-form" role="search">
        <label for="content-search-input" class="visually-hidden">搜索文档内容</label>
        <input
            type="text"
            id="content-search-input"
            placeholder="搜索所有文档..."
            autocomplete="off"
        />
        <button id="content-search-clear" type="button" aria-label="清除搜索">
            ×
        </button>
    </form>
    <!-- 搜索结果面板 -->
    <div id="content-search-results" class="content-search-results" style="display: none;">
        <div class="search-results-header">
            <span class="search-results-count">找到 N 个结果</span>
            <button class="search-results-close" type="button">×</button>
        </div>
        <ul class="search-results-list"></ul>
    </div>
</div>

<article class="paper" id="content">
    {{ .Content }}
</article>
```

### 交互流程

```
用户输入 → debounce 300ms → fetch('/api/search?q=keyword')
    ↓
收到响应 → renderSearchResults(data) → 显示结果面板
    ↓
点击结果 → navigateTo(href) → 内联导航到目标文件
    ↓
ESC/关闭 → closeSearchResults() → 隐藏面板
```

---

## CSS 样式设计

**修改文件**: `web/src/style/app.css`

### 搜索框样式

```css
/* 搜索栏容器 */
.content-search-bar {
    width: 100%;
    max-width: var(--layout-max-width);
    margin-bottom: 20px;
    padding: 0;
}

.content-search-form {
    position: relative;
    display: flex;
    align-items: center;
    width: 300px;
}

.content-search-form input {
    width: 100%;
    height: 36px;
    padding: 0 36px 0 12px;
    border: 1px solid var(--border-light);
    border-radius: 6px;
    background: var(--bg-surface);
    font-size: 14px;
    color: var(--text-body);
    outline: none;
    transition: border-color 0.2s, box-shadow 0.2s;
}

.content-search-form input:focus {
    border-color: var(--accent-primary);
    box-shadow: 0 0 0 3px rgba(15, 98, 254, 0.1);
}

.content-search-form input::placeholder {
    color: var(--text-muted);
}

#content-search-clear {
    position: absolute;
    right: 8px;
    top: 50%;
    transform: translateY(-50%);
    width: 24px;
    height: 24px;
    border: none;
    background: transparent;
    color: var(--text-muted);
    font-size: 18px;
    cursor: pointer;
    opacity: 0;
    transition: opacity 0.2s;
}

.content-search-form input:not(:placeholder-shown) + #content-search-clear {
    opacity: 0.6;
}

#content-search-clear:hover {
    opacity: 1;
}
```

### 搜索结果面板样式

```css
.content-search-results {
    position: absolute;
    top: 100%;
    left: 0;
    width: 400px;
    max-height: 500px;
    overflow-y: auto;
    background: var(--bg-paper);
    border: 1px solid var(--border-light);
    border-radius: 8px;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.12);
    z-index: 1000;
    margin-top: 4px;
}

.search-results-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 12px 16px;
    border-bottom: 1px solid var(--border-light);
}

.search-results-count {
    font-size: 13px;
    color: var(--text-muted);
}

.search-results-close {
    width: 24px;
    height: 24px;
    border: none;
    background: transparent;
    color: var(--text-muted);
    font-size: 18px;
    cursor: pointer;
}

.search-results-list {
    list-style: none;
    padding: 0;
    margin: 0;
}

.search-result-item {
    padding: 12px 16px;
    border-bottom: 1px solid var(--border-light);
    cursor: pointer;
    transition: background 0.15s;
}

.search-result-item:hover {
    background: var(--accent-subtle);
}

.search-result-item:last-child {
    border-bottom: none;
}

.search-result-link {
    display: flex;
    align-items: baseline;
    gap: 8px;
    color: var(--text-heading);
    font-weight: 500;
}

.search-result-file {
    font-size: 14px;
}

.search-result-line {
    font-size: 12px;
    color: var(--text-muted);
}

.search-result-snippet {
    margin: 8px 0 0;
    font-size: 13px;
    color: var(--text-body);
    line-height: 1.5;
}

.search-result-snippet mark {
    background: var(--accent-subtle);
    color: var(--accent-primary);
    padding: 1px 2px;
    border-radius: 2px;
}

/* 无结果状态 */
.search-results-empty {
    padding: 24px 16px;
    text-align: center;
    color: var(--text-muted);
    font-size: 14px;
}
```

---

## TypeScript 搜索逻辑设计

### 新增文件

**文件**: `web/src/content-search.ts`

### 核心函数

| 函数名 | 作用 | 参数 |
|--------|------|------|
| `initContentSearch()` | 初始化搜索功能，绑定事件 | - |
| `performSearch(query)` | 调用 API 并渲染结果 | 搜索关键词 |
| `renderSearchResults(data)` | 渲染结果面板 | API 响应数据 |
| `highlightMatches(snippet, query)` | 在片段中高亮关键词 | 片段、关键词 |
| `closeSearchResults()` | 关闭结果面板 | - |

### 数据结构

```typescript
interface SearchResponse {
    query: string;
    totalFiles: number;
    totalMatches: number;
    results: SearchResult[];
}

interface SearchResult {
    file: string;
    href: string;
    matches: SearchMatch[];
}

interface SearchMatch {
    line: number;
    snippet: string;
    contextBefore?: string;
    contextAfter?: string;
}
```

### 核心实现

```typescript
import { debounce } from './util';

// 初始化
export function initContentSearch() {
    const input = document.getElementById('content-search-input');
    const clearBtn = document.getElementById('content-search-clear');
    const resultsPanel = document.getElementById('content-search-results');
    
    if (!input || !clearBtn || !resultsPanel) return;
    
    const debouncedSearch = debounce(performSearch, 300);
    
    // 输入事件
    input.addEventListener('input', (e) => {
        const query = (e.target as HTMLInputElement).value.trim();
        if (query.length >= 2) {
            debouncedSearch(query);
        } else {
            closeSearchResults();
        }
    });
    
    // 清除按钮
    clearBtn.addEventListener('click', () => {
        input.value = '';
        closeSearchResults();
        input.focus();
    });
    
    // ESC 键关闭
    input.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') {
            closeSearchResults();
            input.blur();
        }
    });
    
    // 关闭按钮
    resultsPanel.querySelector('.search-results-close')?.addEventListener('click', closeSearchResults);
}

// 执行搜索
async function performSearch(query: string) {
    const resultsPanel = document.getElementById('content-search-results');
    if (!resultsPanel) return;

    // 显示加载状态
    resultsPanel.innerHTML = `
        <div class="search-results-header">
            <span class="search-results-count">搜索中...</span>
        </div>
    `;
    resultsPanel.style.display = 'block';

    try {
        const response = await fetch(`/api/search?q=${encodeURIComponent(query)}`);
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }
        const data: SearchResponse = await response.json();
        renderSearchResults(data, query);
    } catch (error) {
        console.error('Search error:', error);
        const errorMsg = error instanceof Error ? error.message : '未知错误';
        resultsPanel.innerHTML = `
            <div class="search-results-header">
                <span class="search-results-count">搜索失败: ${errorMsg}</span>
            </div>
        `;
    }
}

// 渲染结果
function renderSearchResults(data: SearchResponse, query: string) {
    const resultsPanel = document.getElementById('content-search-results');
    if (!resultsPanel) return;
    
    const header = resultsPanel.querySelector('.search-results-header');
    const list = resultsPanel.querySelector('.search-results-list');
    
    if (header) {
        header.querySelector('.search-results-count')?.textContent = 
            `找到 ${data.totalMatches} 个结果（${data.totalFiles} 个文件）`;
    }
    
    if (list) {
        if (data.results.length === 0) {
            list.innerHTML = '<li class="search-results-empty">未找到匹配结果</li>';
        } else {
            list.innerHTML = data.results.map(result => `
                <li class="search-result-item" data-href="${result.href}">
                    ${result.matches.map(match => `
                        <div class="search-result-match">
                            <div class="search-result-link">
                                <span class="search-result-file">${result.file}</span>
                                <span class="search-result-line">第 ${match.line} 行</span>
                            </div>
                            <p class="search-result-snippet">${highlightMatches(match.snippet, query)}</p>
                        </div>
                    `).join('')}
                </li>
            `).join('');
            
            // 绑定点击事件
            list.querySelectorAll('.search-result-item').forEach(item => {
                item.addEventListener('click', () => {
                    const href = item.getAttribute('data-href');
                    if (href) {
                        // 使用内联导航
                        navigateToResult(href);
                    }
                });
            });
        }
    }
    
    resultsPanel.style.display = 'block';
}

// 高亮关键词
function highlightMatches(snippet: string, query: string): string {
    const regex = new RegExp(query, 'gi');
    return snippet.replace(regex, '<mark>$&</mark>');
}

// 关闭结果面板
function closeSearchResults() {
    const resultsPanel = document.getElementById('content-search-results');
    if (resultsPanel) {
        resultsPanel.style.display = 'none';
    }
}

// 导航到结果（复用现有内联导航机制）
// 说明：app.ts 中的 setupInlineNavigation() 在 document 上监听点击事件
// 我们创建真实的 <a> 元素并派发点击事件，让它被内联导航捕获
function navigateToResult(href: string) {
    closeSearchResults();

    // 创建真实的 <a> 元素
    const link = document.createElement('a');
    link.href = href;
    link.style.display = 'none';
    document.body.appendChild(link);

    // 派发点击事件（会被 setupInlineNavigation 捕获）
    const clickEvent = new MouseEvent('click', {
        bubbles: true,
        cancelable: true,
        button: 0,  // 左键
    });
    link.dispatchEvent(clickEvent);

    // 清理
    document.body.removeChild(link);
}
```

### 集成到 app.ts

```typescript
// app.ts 顶部添加 import
import { initContentSearch } from './content-search';

// 在 setupOnce() 中调用
function setupOnce() {
    // ... 其他初始化
    initContentSearch();
}
```

---

## 文件改动汇总

| 文件 | 改动内容 | 改动量 |
|------|----------|--------|
| `handlers.go` | 新增 `handleSearch()` 函数 | ~80 行 |
| `main.go` | 新增路由 | ~2 行 |
| `web/template-main.html` | 添加搜索框和结果面板 HTML | ~25 行 |
| `web/src/content-search.ts` | 新增搜索逻辑模块 | ~100 行 |
| `web/src/style/app.css` | 添加搜索框和结果面板样式 | ~70 行 |
| `web/src/app.ts` | 导入并调用 initContentSearch | ~2 行 |

**总改动量**: ~280 行

---

## 测试设计

### 测试场景

| 场景 | 测试操作 | 验证点 |
|------|----------|--------|
| T1 | 输入少于 2 字符 | 不触发搜索 |
| T2 | 输入有效关键词 | 显示匹配结果 |
| T3 | 无匹配结果 | 显示"未找到"提示 |
| T4 | 点击结果项 | 内联导航到目标文件 |
| T5 | ESC 键 | 关闭结果面板 |
| T6 | 点击关闭按钮 | 关闭结果面板 |
| T7 | 点击清除按钮 | 清空输入并关闭面板 |
| T8 | 结果片段高亮 | 关键词用 mark 标签包裹 |
| T9 | 多文件匹配 | 结果面板显示所有匹配文件 |

---

## 设计约束

| 约束项 | 说明 |
|--------|------|
| 复用现有内联导航 | 使用 app.ts 的 navigateTo 机制 |
| CSS 变量系统 | 使用 --border-light, --bg-paper 等现有变量 |
| 无新增依赖 | 使用原生 fetch 和 DOM 操作 |
| 保持现有风格 | 与项目 TypeScript + DOM 操作风格一致 |

---

## 后续优化建议（可选）

1. 搜索结果缓存（相同关键词不重复请求）
2. 支持正则表达式搜索
3. 搜索历史记录
4. 搜索结果排序（按匹配度/文件名）