export function escapeHtml(value: string) {
    return value
        .replaceAll('&', '&amp;')
        .replaceAll('<', '&lt;')
        .replaceAll('>', '&gt;')
        .replaceAll('"', '&quot;')
        .replaceAll("'", '&#39;');
}

export function toDisplaySlug(value: string) {
    return value
        .trim()
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, '-')
        .replace(/(^-|-$)/g, '');
}

export function buildHeadingAnchorId(value: string, fallbackIndex?: number) {
    const trimmed = value.trim();
    if (!trimmed) {
        return fallbackIndex === undefined ? 'section' : `section-${fallbackIndex}`;
    }

    if (!containsCJK(trimmed)) {
        return toDisplaySlug(trimmed) || (fallbackIndex === undefined ? 'section' : `section-${fallbackIndex}`);
    }

    const asciiPrefix = trimmed
        .normalize('NFKD')
        .replace(/[\u0300-\u036f]/g, '')
        .replace(/[\u3400-\u9fff]/g, ' ')
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, '-')
        .replace(/(^-|-$)/g, '');

    const prefix = asciiPrefix || (fallbackIndex === undefined ? 'index' : `i${fallbackIndex}`);
    return `${prefix}-${shortHash(trimmed)}`;
}

export function ensureUniqueId(baseId: string, usedIds: Set<string>) {
    if (!usedIds.has(baseId)) {
        usedIds.add(baseId);
        return baseId;
    }

    let suffix = 2;
    while (usedIds.has(`${baseId}-${suffix}`)) {
        suffix++;
    }

    const uniqueId = `${baseId}-${suffix}`;
    usedIds.add(uniqueId);
    return uniqueId;
}

export function readJSONScript<T>(id: string): T | null {
    const element = document.getElementById(id);
    if (!element?.textContent) {
        return null;
    }

    try {
        return JSON.parse(element.textContent) as T;
    } catch (error) {
        console.warn(`Failed to parse ${id}`, error);
        return null;
    }
}

export function isInlineNavigablePath(pathname: string) {
    if (pathname === '/') {
        return true;
    }

    const lastSegment = pathname.split('/').filter(Boolean).pop() || '';
    if (!lastSegment) {
        return true;
    }

    return !lastSegment.includes('.') || lastSegment.toLowerCase().endsWith('.md');
}

export function scrollToHash(hash: string) {
    if (!hash) {
        return;
    }

    const rawId = decodeURIComponent(hash.replace(/^#/, ''));
    if (!rawId) {
        return;
    }

    const target = document.getElementById(rawId)
        || document.querySelector(`[name="${CSS.escape(rawId)}"]`);

    if (target instanceof HTMLElement) {
        target.scrollIntoView({ behavior: 'auto', block: 'start' });
    }
}

export function buildContentBaseURL(currentFilePath: string, origin = window.location.origin) {
    const normalizedPath = currentFilePath.replace(/\\/g, '/');
    const lastSlashIndex = normalizedPath.lastIndexOf('/');
    const directory = lastSlashIndex >= 0 ? normalizedPath.slice(0, lastSlashIndex + 1) : '';
    return new URL(`/${directory}`, origin);
}

export function isAlreadyAbsoluteURL(value: string) {
    const trimmed = value.trim();
    return trimmed === ''
        || trimmed.startsWith('#')
        || trimmed.startsWith('/')
        || trimmed.startsWith('//')
        || /^[a-zA-Z][a-zA-Z\d+\-.]*:/.test(trimmed);
}

export function rewriteAttributeURLs(root: HTMLElement, selector: string, attribute: 'href' | 'src', baseURL: URL) {
    root.querySelectorAll(selector).forEach(node => {
        if (!(node instanceof HTMLElement)) {
            return;
        }

        const rawValue = node.getAttribute(attribute);
        if (!rawValue || isAlreadyAbsoluteURL(rawValue)) {
            return;
        }

        try {
            const resolved = new URL(rawValue, baseURL);
            const nextValue = resolved.origin === window.location.origin
                ? `${resolved.pathname}${resolved.search}${resolved.hash}`
                : resolved.toString();
            node.setAttribute(attribute, nextValue);
        } catch (error) {
            console.warn(`Failed to rewrite ${attribute} for`, rawValue, error);
        }
    });
}

export function chevronIcon() {
    return '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><polyline points="9 18 15 12 9 6"></polyline></svg>';
}

export function folderIcon() {
    return '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V7z"></path></svg>';
}

export function fileIcon() {
    return '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path><polyline points="14 2 14 8 20 8"></polyline></svg>';
}

function containsCJK(value: string) {
    return /[\u3400-\u9fff]/.test(value);
}

function shortHash(value: string) {
    let hash = 2166136261;

    for (const char of value) {
        hash ^= char.codePointAt(0) ?? 0;
        hash = Math.imul(hash, 16777619);
    }

    return (hash >>> 0).toString(36).slice(0, 6).padStart(6, '0');
}
