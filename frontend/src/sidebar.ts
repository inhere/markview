import {
    chevronIcon,
    fileIcon,
    folderIcon,
    readJSONScript,
} from './util';

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
        if (!header.id) {
            const friendlyId = (header as HTMLElement).innerText
                .toLowerCase()
                .replace(/[^a-z0-9]+/g, '-')
                .replace(/(^-|-$)/g, '');
            header.id = friendlyId || `section-${index}`;
        }

        let level = 'h1';
        if (header.tagName === 'H2') level = 'h2';
        if (header.tagName === 'H3') level = 'h3';

        const li = document.createElement('li');
        li.className = 'toc-item';

        const anchor = document.createElement('a');
        anchor.href = `#${header.id}`;
        anchor.innerText = (header as HTMLElement).innerText;
        anchor.className = `toc-link toc-${level}`;

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
    content.className = node.navigable && node.href ? 'tree-link' : 'tree-label';
    if (isActive) {
        content.classList.add('active');
    } else if (isInCurrentBranch && hasChildren) {
        content.classList.add('ancestor');
    }

    if (content instanceof HTMLAnchorElement && node.href) {
        content.href = node.href;
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
