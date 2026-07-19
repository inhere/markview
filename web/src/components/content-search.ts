/**
 * Content Search Component
 * 提供内容区域的搜索功能，在 article.paper 之前显示搜索框和结果面板
 */

export interface SearchMatch {
    line: number;
    snippet: string;
    lines?: number[];
    context?: string[];
}

export interface SearchResult {
    file: string;
    matches: SearchMatch[];
}

export interface SearchResponse {
    query: string;
    results: SearchResult[];
    total: number;
    duration?: number;
    filesScanned?: number;
}

/** 转义 HTML 特殊字符 */
function escapeHtml(text: string): string {
    return text
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#039;');
}

/** 高亮关键词 */
function highlightKeywords(snippet: string, query: string): string {
    const escaped = escapeHtml(snippet);
    const keywords = query
        .split(/\s+/)
        .filter(word => word.length >= 2 && !word.startsWith('!'))
        .map(word => word.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'));

    if (keywords.length === 0) {
        return escaped;
    }

    const regex = new RegExp(`(${keywords.join('|')})`, 'gi');
    return escaped.replace(regex, '<mark>$1</mark>');
}

/** 渲染搜索结果 */
export function renderResults(response: SearchResponse, container: HTMLElement): void {
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
                    const content = isMatchLine ? highlightedSnippet : escapeHtml(line);
                    return `<div class="${cls}" data-line="${lineNum}"><span class="line-num">${lineNum}</span><span class="context-text">${content}</span></div>`;
                }).join('')
                : `<div class="context-line match-line" data-line="${match.line}"><span class="line-num">${match.line}</span><span class="context-text">${highlightedSnippet}</span></div>`;

            return `
                <div class="content-search-match" data-file="${result.file}" data-line="${match.line}">
                    ${contextHtml}
                </div>
            `;
        }).join('');

            return `
                <div class="content-search-result-group">
                <div class="content-search-file" data-file="${result.file}">
                    <span class="file-icon" aria-hidden="true">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round">
                            <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
                            <polyline points="14 2 14 8 20 8"></polyline>
                        </svg>
                    </span>
                    <span class="file-name">${result.file}</span>
                    <!-- 空 matches（纯 exclude 查询）显示 "file match"，否则显示匹配数量 -->
                    <span class="match-count">${result.matches.length === 0 ? 'file match' : result.matches.length}</span>
                </div>
                <div class="content-search-matches">${matchesHtml}</div>
            </div>
        `;
    }).join('');

    const statsInfo = response.duration !== undefined && response.filesScanned !== undefined
        ? ` · ${response.duration}ms · in ${response.filesScanned} files`
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

export function hasSearchTerms(query: string): boolean {
    return query
        .split(/\s+/)
        .filter(term => term && !term.startsWith('path:'))
        .join(' ')
        .length >= 2;
}

/** 执行搜索 */
async function performSearch(query: string, resultsContainer: HTMLElement): void {
    if (!hasSearchTerms(query)) {
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
    if (clearBtn && 'style' in clearBtn) {
        clearBtn.style.display = input.value.length > 0 ? 'block' : 'none';
    }
}

export function setupContentSearch(): void {
    const searchWrapper = document.getElementById('content-search');
    if (!searchWrapper) {
        return;
    }

    const trigger = document.getElementById('content-search-trigger') as HTMLButtonElement | null;
    const backdrop = searchWrapper.querySelector('.content-search-backdrop') as HTMLButtonElement | null;
    const input = searchWrapper.querySelector('.content-search-input') as HTMLInputElement;
    const clearBtn = searchWrapper.querySelector('.content-search-clear') as HTMLButtonElement;
    const resultsContainer = searchWrapper.querySelector('.content-search-results') as HTMLElement;

    if (!input || !clearBtn || !resultsContainer) {
        return;
    }

    const elementCtor = document.defaultView?.Element;
    const htmlElementCtor = document.defaultView?.HTMLElement;

    function isElement(value: EventTarget | null): value is Element {
        return !!elementCtor && value instanceof elementCtor;
    }

    function isHTMLElement(value: Element | null | undefined): value is HTMLElement {
        return !!htmlElementCtor && value instanceof htmlElementCtor;
    }

    function openSearch() {
        searchWrapper.hidden = false;
        trigger?.setAttribute('aria-expanded', 'true');
        input.focus();
    }

    function closeSearch() {
        closeResults(resultsContainer, input);
        searchWrapper.hidden = true;
        trigger?.setAttribute('aria-expanded', 'false');
        trigger?.focus();
    }

    trigger?.setAttribute('aria-controls', 'content-search');
    trigger?.setAttribute('aria-expanded', 'false');
    trigger?.addEventListener('click', openSearch);
    backdrop?.addEventListener('click', closeSearch);

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
            closeSearch();
        }
    });

    document.addEventListener('keydown', (e: KeyboardEvent) => {
        const key = e.key.toLowerCase();
        if ((e.ctrlKey || e.metaKey) && key === 'k') {
            e.preventDefault();
            openSearch();
            return;
        }
        if (e.key === 'Escape' && !searchWrapper.hidden) {
            e.preventDefault();
            closeSearch();
        }
    });

    resultsContainer.addEventListener('click', (e: MouseEvent) => {
        const target = isElement(e.target) ? e.target : null;

        const contextLine = target?.closest('.context-line');
        if (isHTMLElement(contextLine)) {
            const matchItem = contextLine.closest('.content-search-match');
            const file = isHTMLElement(matchItem) ? matchItem.dataset.file : undefined;
            const line = contextLine.dataset.line;
            if (file && line) {
                navigateToResult(file, parseInt(line, 10));
                closeSearch();
            }
            return;
        }

        const matchItem = target?.closest('.content-search-match');
        if (isHTMLElement(matchItem)) {
            const file = matchItem.dataset.file;
            const line = matchItem.dataset.line;
            if (file) {
                navigateToResult(file, line ? parseInt(line, 10) : undefined);
                closeSearch();
            }
            return;
        }

        const fileItem = target?.closest('.content-search-file');
        if (isHTMLElement(fileItem)) {
            const group = fileItem.closest('.content-search-result-group');
            if (isHTMLElement(group)) {
                group.classList.toggle('expanded');
            }
        }
    });

    input.addEventListener('focus', () => {
        if (resultsContainer.innerHTML && input.value.trim().length >= 2) {
            resultsContainer.style.display = 'block';
        }
    });
}
