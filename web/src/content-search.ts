/**
 * Content Search Component
 * 提供内容区域的搜索功能，在 article.paper 之前显示搜索框和结果面板
 */

interface SearchMatch {
    line: number;
    snippet: string;
    lines?: number[];
    context?: string[];
}

interface SearchResult {
    file: string;
    matches: SearchMatch[];
}

interface SearchResponse {
    query: string;
    results: SearchResult[];
    total: number;
    duration?: number;
    filesScanned?: number;
}

/** 高亮关键词 */
function highlightKeywords(snippet: string, query: string): string {
    // 解析查询词（支持 !前缀过滤词）
    const keywords = query
        .split(/\s+/)
        .filter(word => word.length >= 2 && !word.startsWith('!'))
        .map(word => word.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')); // 转义正则特殊字符
    
    if (keywords.length === 0) {
        return snippet;
    }
    
    const regex = new RegExp(`(${keywords.join('|')})`, 'gi');
    return snippet.replace(regex, '<mark>$1</mark>');
}

/** 渲染搜索结果 */
function renderResults(response: SearchResponse, container: HTMLElement): void {
    if (response.total === 0) {
        container.innerHTML = `
            <div class="content-search-empty">
                <span>No results found for "${response.query}"</span>
            </div>
        `;
        return;
    }

    const html = response.results.map(result => {
        const matchesHtml = result.matches.map(match => {
            const highlightedSnippet = highlightKeywords(match.snippet, response.query);
            const contextHtml = match.context && match.context.length > 1
                ? match.context.map((line, idx) => {
                    const lineNum = match.lines ? match.lines[idx] : match.line;
                    const isMatchLine = lineNum === match.line;
                    const cls = isMatchLine ? 'context-line match-line' : 'context-line';
                    const content = isMatchLine ? highlightedSnippet : line;
                    return `<div class="${cls}" data-line="${lineNum}"><span class="line-num">${lineNum}</span>${content}</div>`;
                }).join('')
                : `<div class="context-line match-line" data-line="${match.line}"><span class="line-num">${match.line}</span>${highlightedSnippet}</div>`;

            return `
                <div class="content-search-match" data-file="${result.file}" data-line="${match.line}">
                    ${contextHtml}
                </div>
            `;
        }).join('');

        return `
            <div class="content-search-result-group">
                <div class="content-search-file" data-file="${result.file}">
                    <span class="file-icon">📄</span>
                    <span class="file-name">${result.file}</span>
                    <span class="match-count">${result.matches.length}</span>
                </div>
                <div class="content-search-matches">${matchesHtml}</div>
            </div>
        `;
    }).join('');

    const statsInfo = response.duration !== undefined && response.filesScanned !== undefined
        ? ` · ${response.duration}ms · ${response.filesScanned} files`
        : '';

    container.innerHTML = `
        <div class="content-search-header">
            <span>${response.total} results${statsInfo}</span>
        </div>
        <div class="content-search-list">${html}</div>
    `;
}

/** Debounce 函数 */
function debounce<T extends (...args: unknown[]) => unknown>(
    fn: T,
    delay: number
): (...args: Parameters<T>) => void {
    let timeoutId: ReturnType<typeof setTimeout> | null = null;
    
    return (...args: Parameters<T>) => {
        if (timeoutId) {
            clearTimeout(timeoutId);
        }
        timeoutId = setTimeout(() => {
            fn(...args);
            timeoutId = null;
        }, delay);
    };
}

/** 执行搜索 */
async function performSearch(query: string, resultsContainer: HTMLElement): void {
    if (query.length < 2) {
        resultsContainer.innerHTML = '';
        resultsContainer.style.display = 'none';
        return;
    }
    
    resultsContainer.style.display = 'block';
    resultsContainer.innerHTML = `
        <div class="content-search-loading">
            <span>Searching...</span>
        </div>
    `;
    
    try {
        const response = await fetch(`/api/search?q=${encodeURIComponent(query)}`);
        if (!response.ok) {
            throw new Error(`Search failed: ${response.status}`);
        }
        
        const data: SearchResponse = await response.json();
        renderResults(data, resultsContainer);
    } catch (error) {
        resultsContainer.innerHTML = `
            <div class="content-search-error">
                <span>Search error: ${error instanceof Error ? error.message : 'Unknown error'}</span>
            </div>
        `;
    }
}

function navigateToResult(file: string, line?: number): void {
    const url = new URL(window.location.href);
    url.pathname = `/${file}`;
    url.hash = line ? `#L${line}` : '';
    
    // 使用虚拟链接触发内联导航（经过 app.ts 的导航处理）
    const anchor = document.createElement('a');
    anchor.href = url.toString();
    anchor.dispatchEvent(new MouseEvent('click', {
        bubbles: true,
        cancelable: true,
        view: window,
        button: 0,
    }));
}

/** 关闭结果面板 */
function closeResults(resultsContainer: HTMLElement, input: HTMLInputElement): void {
    resultsContainer.style.display = 'none';
    resultsContainer.innerHTML = '';
    input.value = '';
    updateClearButton(input);
}

/** 更新清除按钮显示状态 */
function updateClearButton(input: HTMLInputElement): void {
    const clearBtn = input.parentElement?.querySelector('.content-search-clear');
    if (clearBtn instanceof HTMLElement) {
        clearBtn.style.display = input.value.length > 0 ? 'block' : 'none';
    }
}

export function setupContentSearch(): void {
    const searchWrapper = document.getElementById('content-search');
    if (!searchWrapper) {
        return;
    }
    
    const input = searchWrapper.querySelector('.content-search-input') as HTMLInputElement;
    const clearBtn = searchWrapper.querySelector('.content-search-clear') as HTMLButtonElement;
    const resultsContainer = searchWrapper.querySelector('.content-search-results') as HTMLElement;
    
    if (!input || !clearBtn || !resultsContainer) {
        return;
    }
    
    const debouncedSearch = debounce((query: string) => {
        performSearch(query, resultsContainer);
    }, 300);
    
    input.addEventListener('input', () => {
        const query = input.value.trim();
        updateClearButton(input);
        debouncedSearch(query);
    });
    
    clearBtn.addEventListener('click', () => {
        closeResults(resultsContainer, input);
    });
    
    input.addEventListener('keydown', (e: KeyboardEvent) => {
        if (e.key === 'Escape') {
            closeResults(resultsContainer, input);
            input.blur();
        }
    });
    
    resultsContainer.addEventListener('click', (e: MouseEvent) => {
        const target = e.target instanceof Element ? e.target : null;

        const contextLine = target?.closest('.context-line');
        if (contextLine instanceof HTMLElement) {
            const matchItem = contextLine.closest('.content-search-match');
            const file = matchItem instanceof HTMLElement ? matchItem.dataset.file : undefined;
            const line = contextLine.dataset.line;
            if (file && line) {
                navigateToResult(file, parseInt(line, 10));
                closeResults(resultsContainer, input);
            }
            return;
        }

        const matchItem = target?.closest('.content-search-match');
        if (matchItem instanceof HTMLElement) {
            const file = matchItem.dataset.file;
            const line = matchItem.dataset.line;
            if (file) {
                navigateToResult(file, line ? parseInt(line, 10) : undefined);
                closeResults(resultsContainer, input);
            }
            return;
        }

        const fileItem = target?.closest('.content-search-file');
        if (fileItem instanceof HTMLElement) {
            const group = fileItem.closest('.content-search-result-group');
            if (group instanceof HTMLElement) {
                group.classList.toggle('expanded');
            }
        }
    });
    
    document.addEventListener('click', (e: MouseEvent) => {
        const target = e.target instanceof Element ? e.target : null;
        if (!searchWrapper.contains(target)) {
            resultsContainer.style.display = 'none';
        }
    });
    
    input.addEventListener('focus', () => {
        if (resultsContainer.innerHTML && input.value.trim().length >= 2) {
            resultsContainer.style.display = 'block';
        }
    });
}