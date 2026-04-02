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
    name: string;
    href?: string;
    matchPath?: string;
    kind: 'directory' | 'file';
    navigable: boolean;
    children?: FileTreeNode[];
}

interface RenderFileTreeOptions {
    treeRootId: string;
    treeDataId: string;
    currentFilePathDataId: string;
}

export function generateTOC(contentSelector = '#content', tocListId = 'toc-list') {
    const tocList = document.getElementById(tocListId);
    const headers = document.querySelectorAll(`${contentSelector} h1, ${contentSelector} h2, ${contentSelector} h3`);
    const usedIds = new Set<string>();

    if (!tocList) {
        return;
    }

    tocList.innerHTML = '';

    if (!headers.length) {
        const empty = document.createElement('li');
        empty.className = 'sidebar-empty';
        empty.textContent = 'No headings';
        tocList.appendChild(empty);
        return;
    }

    headers.forEach((header, index) => {
        const headingText = (header as HTMLElement).innerText.trim();
        const previousId = header.id;
        const baseId = buildHeadingAnchorId(headingText, index);
        header.id = ensureUniqueId(baseId, usedIds);
        if (previousId && previousId !== header.id) {
            document.querySelectorAll(`${contentSelector} a[href="#${CSS.escape(previousId)}"]`).forEach(link => {
                if (link instanceof HTMLAnchorElement) {
                    link.href = `#${header.id}`;
                }
            });
        }

        let level = 'h1';
        if (header.tagName === 'H2') level = 'h2';
        if (header.tagName === 'H3') level = 'h3';

        const li = document.createElement('li');
        li.className = 'toc-item';

        const anchor = document.createElement('a');
        anchor.href = `#${header.id}`;
        anchor.innerText = headingText;
        anchor.className = `toc-link toc-${level}`;
        anchor.title = headingText;
        anchor.setAttribute('aria-label', headingText);

        li.appendChild(anchor);
        tocList.appendChild(li);
    });
}

export function renderFileTree(options: RenderFileTreeOptions) {
    const treeRoot = document.getElementById(options.treeRootId);
    const treeData = readJSONScript<FileTreeNode[]>(options.treeDataId);
    const currentFilePath = readJSONScript<string>(options.currentFilePathDataId);

    if (!treeRoot) {
        return;
    }

    treeRoot.innerHTML = '';

    if (!treeData?.length) {
        const empty = document.createElement('div');
        empty.className = 'sidebar-empty';
        empty.textContent = 'No markdown files';
        treeRoot.appendChild(empty);
        return;
    }

    const list = document.createElement('ul');
    list.className = 'file-tree-list';

    treeData.forEach(node => {
        list.appendChild(createTreeNode(node, currentFilePath || ''));
    });

    treeRoot.appendChild(list);

    const activeNode = treeRoot.querySelector('.tree-link.active, .tree-label.active');
    if (activeNode instanceof HTMLElement) {
        activeNode.scrollIntoView({ block: 'nearest' });
    }
    
    initFilesSearch();
}

// 文件树搜索过滤函数
function filterFileTree(query: string) {
    const allNodes = document.querySelectorAll('.file-tree-node');
    const matchedNodes = new Set<HTMLElement>();
    const normalizedQuery = query.toLowerCase().trim();

    if (!normalizedQuery) {
        // 清空搜索时恢复所有节点
        allNodes.forEach(nodeEl => {
            nodeEl.classList.remove('hidden');
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
        nodeEl.classList.toggle('hidden', !isMatch);

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
            parent.classList.remove('hidden');
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
        nodeEl.classList.remove('hidden');
    });
}

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

export function highlightTOC(contentSelector = '#content', sidebarSelector = '.toc-container') {
    const scrollPos = window.scrollY + 100;
    const headers = document.querySelectorAll(`${contentSelector} h1, ${contentSelector} h2, ${contentSelector} h3`);
    let currentId = '';

    headers.forEach(header => {
        if ((header as HTMLElement).offsetTop <= scrollPos) {
            currentId = header.id;
        }
    });

    if (!currentId) {
        return;
    }

    // 同步 URL hash，保持阅读进度
    const currentHash = `#${currentId}`;
    if (window.location.hash !== currentHash) {
        history.replaceState({ markview: true }, '', currentHash);
    }

    document.querySelectorAll('.toc-link').forEach(link => {
        link.classList.remove('active');
        if (link.getAttribute('href') === `#${currentId}`) {
            link.classList.add('active');
            const sidebar = document.querySelector(sidebarSelector);
            if (!sidebar) {
                return;
            }

            const linkRect = link.getBoundingClientRect();
            const sidebarRect = sidebar.getBoundingClientRect();
            if (linkRect.top < sidebarRect.top || linkRect.bottom > sidebarRect.bottom) {
                link.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
            }
        }
    });
}

function createTreeNode(node: FileTreeNode, currentFilePath: string): HTMLLIElement {
    const item = document.createElement('li');
    item.className = 'file-tree-node';

    const row = document.createElement('div');
    row.className = 'file-tree-row';

    const hasChildren = Boolean(node.children?.length);
    const isInCurrentBranch = nodeContainsPath(node, currentFilePath);
    const shouldExpand = hasChildren && isInCurrentBranch;
    if (isInCurrentBranch && hasChildren) {
        item.classList.add('branch-active');
    }

    const toggle = document.createElement('button');
    toggle.type = 'button';
    toggle.className = 'tree-toggle';

    if (hasChildren) {
        toggle.innerHTML = chevronIcon();
        toggle.setAttribute('aria-label', shouldExpand ? 'Collapse folder' : 'Expand folder');
        toggle.setAttribute('aria-expanded', shouldExpand ? 'true' : 'false');
        if (shouldExpand) {
            toggle.classList.add('expanded');
        }
    } else {
        toggle.classList.add('placeholder');
        toggle.setAttribute('aria-hidden', 'true');
    }

    const isActive = Boolean(node.matchPath && node.matchPath === currentFilePath);
    const content = node.navigable && node.href ? document.createElement('a') : document.createElement('span');
    const tooltipText = node.matchPath || node.name;
    content.className = node.navigable && node.href ? 'tree-link' : 'tree-label';
    content.title = tooltipText;
    if (isActive) {
        content.classList.add('active');
    } else if (isInCurrentBranch && hasChildren) {
        content.classList.add('ancestor');
    }

    if (content instanceof HTMLAnchorElement && node.href) {
        content.href = node.href;
        content.setAttribute('aria-label', tooltipText);
    }

    const icon = document.createElement('span');
    icon.className = 'tree-icon';
    icon.innerHTML = node.kind === 'directory' ? folderIcon() : fileIcon();

    const text = document.createElement('span');
    text.className = 'tree-text';
    text.textContent = node.name;

    content.appendChild(icon);
    content.appendChild(text);

    row.appendChild(toggle);
    row.appendChild(content);
    item.appendChild(row);

    if (hasChildren && node.children) {
        const childList = document.createElement('ul');
        childList.className = 'file-tree-children';
        childList.hidden = !shouldExpand;

        node.children.forEach(child => {
            childList.appendChild(createTreeNode(child, currentFilePath));
        });

        toggle.addEventListener('click', event => {
            event.preventDefault();
            const expanded = toggle.getAttribute('aria-expanded') === 'true';
            const nextExpanded = !expanded;
            toggle.setAttribute('aria-expanded', nextExpanded ? 'true' : 'false');
            toggle.setAttribute('aria-label', nextExpanded ? 'Collapse folder' : 'Expand folder');
            toggle.classList.toggle('expanded', nextExpanded);
            childList.hidden = !nextExpanded;
        });

        item.appendChild(childList);
    }

    return item;
}

function nodeContainsPath(node: FileTreeNode, currentFilePath: string): boolean {
    if (!currentFilePath) {
        return false;
    }
    if (node.matchPath === currentFilePath) {
        return true;
    }
    return node.children?.some(child => nodeContainsPath(child, currentFilePath)) ?? false;
}

export function setupSidebarCollapse() {
    const collapseBtn = document.getElementById('sidebar-collapse-btn');
    const sidebar = document.querySelector('.sidebar');
    const filesPanel = document.getElementById('files-panel');

    if (!collapseBtn || !sidebar || !filesPanel) return;

    const prefs = readSidebarPreferences();

    const updateBodyClass = (collapsed: boolean) => {
        document.body.classList.toggle('sidebar-collapsed', collapsed);
    };

    // Apply initial state
    if (prefs.sidebarCollapsed) {
        sidebar.classList.add('sidebar-collapsed');
        updateBodyClass(true);
    }
    if (prefs.filesCollapsed) {
        filesPanel.classList.add('files-collapsed');
    }

    // Collapse button click
    collapseBtn.addEventListener('click', () => {
        const isCollapsed = sidebar.classList.toggle('sidebar-collapsed');
        updateBodyClass(isCollapsed);
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
            updateBodyClass(false);
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
