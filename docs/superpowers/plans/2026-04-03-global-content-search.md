# 全局内容搜索实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在内容区域顶部添加全局搜索框，支持搜索所有 md 文件内容，使用 AND 和排除语法。

**Architecture:** 
- 后端: Go HTTP API (`GET /api/search?q=...`)，搜索 targetDir 下所有 .md 文件，返回匹配结果
- 前端: TypeScript 组件 (`content-search.ts`)，渲染搜索框和结果下拉面板，使用内联导航跳转

**Tech Stack:** Go (net/http), TypeScript (原生 DOM API), CSS (现有变量)

---

## 文件结构

| 文件 | 责责 |
|------|------|
| `handlers.go` | 搜索 API 处理函数 + 辅助函数 |
| `main.go:135-141` | 注册搜索路由 |
| `web/src/content-search.ts` | 前端搜索组件 (搜索框 + 结果面板) |
| `web/src/app.ts` | 导入并初始化搜索组件 |

---

## Task 1: 后端搜索 API

**Files:**
- Modify: `handlers.go` (新增函数)
- Modify: `main.go:139` (新增路由)

### Step 1: 定义搜索数据结构

在 `handlers.go` 文件末尾添加数据结构定义：

```go
// 搜索术语结构
type SearchTerms struct {
    Include []string // 必须包含的关键词
    Exclude []string // 必须排除的关键词
}

// 单行匹配结果
type SearchMatch struct {
    Line    int    `json:"line"`    // 行号 (1-indexed)
    Snippet string `json:"snippet"` // 匹配片段
}

// 单文件搜索结果
type SearchResult struct {
    File    string        `json:"file"`    // 文件相对路径
    Matches []SearchMatch `json:"matches"` // 匹配行列表
}

// API 响应结构
type SearchResponse struct {
    Query   string         `json:"query"`
    Results []SearchResult `json:"results"`
    Total   int            `json:"total"` // 总匹配行数
}
```

- [ ] **Step 1: 添加数据结构**

### Step 2: 实现搜索术语解析函数

```go
// parseSearchTerms 解析搜索查询字符串
// 多关键词空格隔开 = AND, !前缀 = 排除
func parseSearchTerms(query string) SearchTerms {
    terms := SearchTerms{}
    parts := strings.Fields(query) // 按空格分割

    for _, part := range parts {
        if strings.HasPrefix(part, "!") {
            // 排除关键词
            excludeTerm := strings.TrimPrefix(part, "!")
            if excludeTerm != "" {
                terms.Exclude = append(terms.Exclude, strings.ToLower(excludeTerm))
            }
        } else {
            // 包含关键词
            terms.Include = append(terms.Include, strings.ToLower(part))
        }
    }

    return terms
}
```

- [ ] **Step 2: 实现 parseSearchTerms 函数**

### Step 3: 实现行匹配函数

```go
// lineMatchesMatch 检查一行是否匹配搜索条件
func lineMatchesMatch(line string, terms SearchTerms) bool {
    lineLower := strings.ToLower(line)

    // 检查所有包含关键词
    for _, term := range terms.Include {
        if !strings.Contains(lineLower, term) {
            return false
        }
    }

    // 检查所有排除关键词
    for _, term := range terms.Exclude {
        if strings.Contains(lineLower, term) {
            return false
        }
    }

    return len(terms.Include) > 0 // 必须有至少一个包含关键词
}
```

- [ ] **Step 3: 实现 lineMatchesMatch 函数**

### Step 4: 实现文件内容搜索函数

```go
// searchInContent 在文件内容中搜索匹配行
func searchInContent(content string, terms SearchTerms, limit int) []SearchMatch {
    lines := strings.Split(content, "\n")
    matches := []SearchMatch{}

    for i, line := range lines {
        if lineMatchesMatch(line, terms) {
            matches = append(matches, SearchMatch{
                Line:    i + 1,
                Snippet: extractSnippet(line, terms.Include),
            })
            if len(matches) >= limit {
                break
            }
        }
    }
    return matches
}

// extractSnippet 提取匹配片段（高亮第一个匹配的关键词）
func extractSnippet(line string, includeTerms []string) string {
    lineLower := strings.ToLower(line)

    for _, term := range includeTerms {
        if idx := strings.Index(lineLower, term); idx != -1 {
            start := idx - 30
            if start < 0 {
                start = 0
            }
            end := idx + len(term) + 30
            if end > len(line) {
                end = len(line)
            }
            snippet := line[start:end]
            if start > 0 {
                snippet = "..." + snippet
            }
            if end < len(line) {
                snippet = snippet + "..."
            }
            return snippet
        }
    }
    return line
}

// countMatches 统计总匹配数
func countMatches(results []SearchResult) int {
    count := 0
    for _, r := range results {
        count += len(r.Matches)
    }
    return count
}
```

- [ ] **Step 4: 实现 searchInContent 和辅助函数**

### Step 5: 实现搜索 API 处理函数

```go
// handleSearch 处理搜索 API 请求
func handleSearch(w http.ResponseWriter, r *http.Request) {
    // 只接受 GET 请求
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    // 解析查询参数
    query := r.URL.Query().Get("q")
    if query == "" {
        http.Error(w, "Missing query parameter 'q'", http.StatusBadRequest)
        return
    }

    // 解析搜索术语
    terms := parseSearchTerms(query)
    if len(terms.Include) == 0 {
        http.Error(w, "Query must include at least one positive term", http.StatusBadRequest)
        return
    }

    // 搜索结果
    results := []SearchResult{}
    maxMatchesPerFile := 10

    // 遍历 targetDir 下所有 .md 文件
    err := filepath.WalkDir(targetDir, func(path string, d fs.DirEntry, err error) error {
        if err != nil {
            return err
        }

        // 跳过非 .md 文件和目录
        if d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
            return nil
        }

        // 跳过隐藏目录和 watchSkipDirs
        relPath, err := filepath.Rel(targetDir, path)
        if err != nil {
            return nil
        }
        parts := strings.Split(relPath, string(filepath.Separator))
        for _, part := range parts {
            if strings.HasPrefix(part, ".") || slices.Contains(watchSkipDirs, part) {
                return filepath.SkipDir
            }
        }

        // 读取文件内容
        content, err := os.ReadFile(path)
        if err != nil {
            return nil
        }

        // 搜索匹配行
        matches := searchInContent(string(content), terms, maxMatchesPerFile)
        if len(matches) > 0 {
            results = append(results, SearchResult{
                File:    relPath,
                Matches: matches,
            })
        }

        return nil
    })

    if err != nil {
        http.Error(w, fmt.Sprintf("Search error: %v", err), http.StatusInternalServerError)
        return
    }

    // 按匹配数排序（多的在前）
    sort.Slice(results, func(i, j int) bool {
        return len(results[i].Matches) > len(results[j].Matches)
    })

    // 构造响应
    response := SearchResponse{
        Query:   query,
        Results: results,
        Total:   countMatches(results),
    }

    // 返回 JSON
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}
```

需要添加的 import:
```go
"encoding/json"
"io/fs"
```

- [ ] **Step 5: 实现 handleSearch 函数并添加 import**

### Step 6: 注册搜索路由

在 `main.go` 的 `newServerMux()` 函数中添加路由（在 `/sse` 之后）：

```go
func newServerMux() *http.ServeMux {
    mux := http.NewServeMux()
    mux.Handle("/static/", newStaticHandler())
    mux.HandleFunc("/sse", handleSSE)
    mux.HandleFunc("/api/search", handleSearch) // 新增搜索 API
    mux.HandleFunc("/", handleRequest)
    return mux
}
```

- [ ] **Step 6: 在 main.go 中注册搜索路由**

### Step 7: 验证后端 API

启动服务并测试 API：

```bash
# 启动服务
go run . --port 8080 --debug

# 测试搜索 API
curl "http://localhost:8080/api/search?q=api"
curl "http://localhost:8080/api/search?q=api%20!deprecated"
```

预期响应格式:
```json
{
  "query": "api !deprecated",
  "results": [
    {
      "file": "docs/api.md",
      "matches": [
        {"line": 10, "snippet": "...the api endpoint..."}
      ]
    }
  ],
  "total": 5
}
```

- [ ] **Step 7: 测试搜索 API 返回正确 JSON**

### Step 8: 提交后端代码

```bash
git add handlers.go main.go
git commit -m "feat: 添加全局内容搜索 API

- GET /api/search?q=... 支持多关键词 AND 和排除语法
- 搜索 targetDir 下所有 .md 文件
- 返回匹配文件、行号、片段"
```

- [ ] **Step 8: 提交后端代码**

---

## Task 2: 前端搜索组件

**Files:**
- Create: `web/src/content-search.ts`
- Modify: `web/src/app.ts` (导入并初始化)

### Step 1: 创建搜索组件文件

创建 `web/src/content-search.ts`：

```typescript
// content-search.ts - 全局内容搜索组件
export {};

interface SearchMatch {
    line: number;
    snippet: string;
}

interface SearchResult {
    file: string;
    matches: SearchMatch[];
}

interface SearchResponse {
    query: string;
    results: SearchResult[];
    total: number;
}

// 搜索状态
let searchAbortController: AbortController | null = null;
let searchDebounceTimer: number | null = null;
let isSearching = false;

// 创建搜索框 HTML
function createSearchHTML(): string {
    return `
<div class="content-search-wrapper">
    <div class="content-search-box">
        <input type="text" 
               class="content-search-input" 
               placeholder="搜索内容... (空格=AND, !前缀=排除)"
               autocomplete="off">
        <span class="content-search-icon">🔍</span>
        <span class="content-search-loading" style="display:none">⏳</span>
    </div>
    <div class="content-search-results" style="display:none"></div>
</div>
`;
}

// 渲染搜索结果
function renderResults(response: SearchResponse, resultsContainer: HTMLElement): void {
    if (response.results.length === 0) {
        resultsContainer.innerHTML = `
            <div class="search-result-empty">
                未找到匹配 "${response.query}" 的内容
            </div>
        `;
        return;
    }

    const html = response.results.map(result => `
        <div class="search-result-file">
            <div class="search-result-file-name">${result.file}</div>
            ${result.matches.map(match => `
                <div class="search-result-match" 
                     data-file="${result.file}" 
                     data-line="${match.line}">
                    <span class="search-result-line">L${match.line}</span>
                    <span class="search-result-snippet">${match.snippet}</span>
                </div>
            `).join('')}
        </div>
    `).join('');

    resultsContainer.innerHTML = html;
    
    // 显示总匹配数
    resultsContainer.insertAdjacentHTML('afterbegin', `
        <div class="search-result-header">
            找到 ${response.total} 处匹配
        </div>
    `);
}

// 执行搜索
async function performSearch(query: string, resultsContainer: HTMLElement, loadingEl: HTMLElement): void {
    if (!query.trim()) {
        resultsContainer.style.display = 'none';
        return;
    }

    // 取消之前的请求
    if (searchAbortController) {
        searchAbortController.abort();
    }
    searchAbortController = new AbortController();

    isSearching = true;
    loadingEl.style.display = 'inline';

    try {
        const response = await fetch(`/api/search?q=${encodeURIComponent(query)}`, {
            signal: searchAbortController.signal
        });

        if (!response.ok) {
            throw new Error(`搜索失败: ${response.status} ${response.statusText}`);
        }

        const data: SearchResponse = await response.json();
        resultsContainer.style.display = 'block';
        renderResults(data, resultsContainer);
    } catch (error) {
        if (error instanceof Error && error.name === 'AbortError') {
            // 被取消，不显示错误
            return;
        }
        resultsContainer.style.display = 'block';
        resultsContainer.innerHTML = `
            <div class="search-result-error">
                搜索出错: ${error instanceof Error ? error.message : '未知错误'}
            </div>
        `;
    } finally {
        isSearching = false;
        loadingEl.style.display = 'none';
    }
}

// 导航到结果
function navigateToResult(file: string, line: number): void {
    // 使用 MouseEvent 触发内联导航
    const link = document.createElement('a');
    link.href = `/${file}`;
    link.style.display = 'none';
    document.body.appendChild(link);
    
    const event = new MouseEvent('click', {
        bubbles: true,
        cancelable: true,
        view: window
    });
    link.dispatchEvent(event);
    document.body.removeChild(link);

    // TODO: 后续可添加滚动到指定行的逻辑
}

// 初始化搜索组件
export function setupContentSearch(): void {
    // 找到内容区域容器
    const contentWrapper = document.querySelector('.content-wrapper');
    if (!contentWrapper) {
        console.warn('Content wrapper not found, cannot setup search');
        return;
    }

    // 找到 article.paper
    const article = contentWrapper.querySelector('article.paper');
    if (!article) {
        console.warn('Article not found, cannot setup search');
        return;
    }

    // 在 article 之前插入搜索框
    const searchHTML = createSearchHTML();
    article.insertAdjacentHTML('beforebegin', searchHTML);

    // 获取元素
    const wrapper = contentWrapper.querySelector('.content-search-wrapper') as HTMLElement;
    const input = wrapper.querySelector('.content-search-input') as HTMLInputElement;
    const resultsContainer = wrapper.querySelector('.content-search-results') as HTMLElement;
    const loadingEl = wrapper.querySelector('.content-search-loading') as HTMLElement;

    // 输入事件 - debounce 300ms
    input.addEventListener('input', () => {
        if (searchDebounceTimer) {
            clearTimeout(searchDebounceTimer);
        }
        searchDebounceTimer = window.setTimeout(() => {
            performSearch(input.value, resultsContainer, loadingEl);
        }, 300);
    });

    // 点击结果项导航
    resultsContainer.addEventListener('click', (e) => {
        const target = e.target as HTMLElement;
        const matchEl = target.closest('.search-result-match') as HTMLElement;
        if (matchEl) {
            const file = matchEl.dataset.file!;
            const line = parseInt(matchEl.dataset.line!, 10);
            navigateToResult(file, line);
            // 隐藏结果面板
            resultsContainer.style.display = 'none';
            input.value = '';
        }
    });

    // ESC 键关闭结果面板
    input.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') {
            resultsContainer.style.display = 'none';
            input.blur();
        }
    });

    // 点击外部关闭结果面板
    document.addEventListener('click', (e) => {
        const target = e.target as HTMLElement;
        if (!wrapper.contains(target)) {
            resultsContainer.style.display = 'none';
        }
    });
}
```

- [ ] **Step 1: 创建 content-search.ts 文件**

### Step 2: 添加 CSS 样式

在 `web/src/style/app.css` 中添加搜索组件样式：

```css
/* 内容搜索组件 */
.content-search-wrapper {
    position: relative;
    margin-bottom: 20px;
}

.content-search-box {
    position: relative;
    display: flex;
    align-items: center;
    width: 300px;
    background: var(--bg-paper, #fff);
    border: 1px solid var(--border-light, #e0e0e0);
    border-radius: 4px;
    padding: 4px 12px;
}

.content-search-input {
    width: 100%;
    border: none;
    outline: none;
    font-size: 14px;
    padding: 4px 0;
    background: transparent;
    color: inherit;
}

.content-search-input::placeholder {
    color: var(--text-muted, #666);
}

.content-search-icon {
    margin-left: 8px;
    opacity: 0.5;
}

.content-search-loading {
    margin-left: 8px;
    animation: spin 1s linear infinite;
}

@keyframes spin {
    from { transform: rotate(0deg); }
    to { transform: rotate(360deg); }
}

/* 搜索结果面板 */
.content-search-results {
    position: absolute;
    top: 100%;
    left: 0;
    width: 500px;
    max-height: 500px;
    overflow-y: auto;
    background: var(--bg-paper, #fff);
    border: 1px solid var(--border-light, #e0e0e0);
    border-radius: 4px;
    box-shadow: 0 4px 12px rgba(0,0,0,0.15);
    z-index: 100;
    margin-top: 4px;
}

.search-result-header {
    padding: 8px 12px;
    font-size: 12px;
    color: var(--text-muted, #666);
    border-bottom: 1px solid var(--border-light, #e0e0e0);
}

.search-result-file {
    padding: 8px 12px;
}

.search-result-file-name {
    font-weight: 600;
    font-size: 13px;
    color: var(--accent-primary, #0066cc);
    margin-bottom: 4px;
}

.search-result-match {
    display: flex;
    gap: 12px;
    padding: 4px 8px;
    cursor: pointer;
    border-radius: 2px;
}

.search-result-match:hover {
    background: rgba(0,0,0,0.05);
}

.search-result-line {
    font-size: 12px;
    color: var(--text-muted, #666);
    min-width: 40px;
}

.search-result-snippet {
    font-size: 13px;
    color: inherit;
    overflow: hidden;
    text-overflow: ellipsis;
}

.search-result-empty,
.search-result-error {
    padding: 16px;
    text-align: center;
    color: var(--text-muted, #666);
}

.search-result-error {
    color: #c00;
}
```

- [ ] **Step 2: 添加 CSS 样式到 app.css**

### Step 3: 导入并初始化搜索组件

在 `web/src/app.ts` 中添加导入和初始化：

```typescript
// 在文件顶部 import 区域添加
import { setupContentSearch } from './content-search';

// 在 setupApp 函数中调用初始化
export function setupApp() {
    // ... 现有初始化代码 ...
    
    // 初始化内容搜索
    setupContentSearch();
}
```

- [ ] **Step 3: 在 app.ts 中导入并初始化搜索组件**

### Step 4: 验证前端功能

启动服务并测试前端：

```bash
# 构建前端
cd web && npm run build

# 启动服务
go run . --port 8080 --debug

# 打开浏览器访问 http://localhost:8080
# 测试搜索框:
# 1. 输入关键词，等待 300ms 后应显示结果
# 2. 点击结果项应跳转到对应文件
# 3. ESC 键应关闭结果面板
# 4. 点击外部应关闭结果面板
```

- [ ] **Step 4: 测试前端搜索功能正常**

### Step 5: 提交前端代码

```bash
git add web/src/content-search.ts web/src/style/app.css web/src/app.ts
git commit -m "feat: 添加全局内容搜索前端组件

- 搜索框位于内容区域顶部
- Debounce 300ms 延迟搜索
- 结果面板显示匹配文件、行号、片段
- 点击结果项跳转到对应文件
- ESC 键和点击外部关闭结果面板"
```

- [ ] **Step 5: 提交前端代码**

---

## Task 3: 完整功能测试

### Step 1: 测试基本搜索

```bash
# 启动服务
go run . --port 8080 --debug

# 测试场景:
# 1. 搜索单个关键词 "api"
# 2. 搜索多个关键词 "api config" (AND)
# 3. 搜索带排除 "api !deprecated"
# 4. 搜索不存在的内容 "zzzzz" (显示未找到)
# 5. 空搜索 (不触发)
```

- [ ] **Step 1: 测试基本搜索功能**

### Step 2: 测试边界情况

```bash
# 测试边界情况:
# 1. 搜索关键词包含特殊字符
# 2. 搜索结果超过 10 行匹配
# 3. 大量文件匹配时的性能
# 4. 网络错误时的错误提示
```

- [ ] **Step 2: 测试边界情况**

### Step 3: 最终提交

```bash
git status
git log --oneline -3
```

- [ ] **Step 3: 确认所有改动已提交**

---

## QA 场景

### 场景 1: 基本搜索流程
1. 用户在搜索框输入 "api"
2. 等待 300ms 后显示结果面板
3. 面板显示匹配文件和行号
4. 点击结果项跳转到文件

**验证点:** 
- API 返回正确 JSON
- 结果面板正确渲染
- 导航正常工作

### 场景 2: AND 语法搜索
1. 用户输入 "api config"
2. 结果只显示同时包含两个关键词的行

**验证点:** 
- parseSearchTerms 正确分割关键词
- lineMatchesMatch AND 逻辑正确

### 场景 3: 排除语法搜索
1. 用户输入 "api !deprecated"
2. 结果显示包含 api 但不包含 deprecated 的行

**验证点:** 
- ! 前缀识别正确
- 排除逻辑正确

### 场景 4: 错误处理
1. 用户输入不存在的关键词
2. 显示 "未找到匹配" 提示

**验证点:** 
- 空结果正确处理
- 错误信息清晰