export type AppLayout = 'compact' | 'toc-middle' | 'toc-right';

export interface AppConfig {
    previewExts: string[];
    iframeHosts: string[];
    layout: AppLayout;
    basePath: string;
}

export const DEFAULT_APP_CONFIG: AppConfig = {
    previewExts: ['.md', '.json', '.jsonl', '.yaml', '.yml', '.toml', '.html'],
    iframeHosts: [],
    layout: 'compact',
    basePath: '',
};

const CONFIG_SCRIPT_ID = 'app-config-data';
const VALID_LAYOUTS = new Set<AppLayout>(['compact', 'toc-middle', 'toc-right']);
const PROJECT_BASE_PATH = /^\/p\/[0-9a-f]{12}$/;

export function normalizePreviewExts(value: unknown): string[] {
    if (!Array.isArray(value)) {
        return [...DEFAULT_APP_CONFIG.previewExts];
    }

    const seenExts = new Set<string>();
    const previewExts = value
        .filter((item): item is string => typeof item === 'string')
        .map(item => item.trim().toLowerCase())
        .filter(Boolean)
        .map(item => item.startsWith('.') ? item : `.${item}`)
        .filter(item => {
            if (seenExts.has(item)) {
                return false;
            }
            seenExts.add(item);
            return true;
        });

    return previewExts.length > 0 ? previewExts : [...DEFAULT_APP_CONFIG.previewExts];
}

export function normalizeIframeHosts(value: unknown): string[] {
    if (!Array.isArray(value)) {
        return [];
    }

    const seenHosts = new Set<string>();
    return value
        .filter((item): item is string => typeof item === 'string')
        .map(item => normalizeIframeHost(item))
        .filter(Boolean)
        .filter(host => {
            if (seenHosts.has(host)) {
                return false;
            }
            seenHosts.add(host);
            return true;
        });
}

function normalizeIframeHost(value: string): string {
    let rawValue = value.trim().toLowerCase();
    if (!rawValue) {
        return '';
    }

    try {
        if (rawValue.includes('://')) {
            return new URL(rawValue).host;
        }
        if (rawValue.startsWith('//')) {
            return new URL(`http:${rawValue}`).host;
        }
    } catch {
        return '';
    }

    const slashIndex = rawValue.indexOf('/');
    if (slashIndex >= 0) {
        rawValue = rawValue.slice(0, slashIndex);
    }
    return rawValue;
}

function normalizeLayout(value: unknown): AppLayout {
    if (typeof value === 'string' && VALID_LAYOUTS.has(value as AppLayout)) {
        return value as AppLayout;
    }
    return DEFAULT_APP_CONFIG.layout;
}

export function normalizeAppConfig(value: unknown): AppConfig {
    if (!value || typeof value !== 'object') {
        return {
            ...DEFAULT_APP_CONFIG,
            previewExts: [...DEFAULT_APP_CONFIG.previewExts],
            iframeHosts: [...DEFAULT_APP_CONFIG.iframeHosts],
        };
    }

    const rawConfig = value as Record<string, unknown>;
    return {
        previewExts: normalizePreviewExts(rawConfig.previewExts),
        iframeHosts: normalizeIframeHosts(rawConfig.iframeHosts),
        layout: normalizeLayout(rawConfig.layout),
        basePath: typeof rawConfig.basePath === 'string' && PROJECT_BASE_PATH.test(rawConfig.basePath)
            ? rawConfig.basePath
            : '',
    };
}

export function readAppConfig(documentRef: Document = document): AppConfig {
    const script = documentRef.getElementById(CONFIG_SCRIPT_ID);
    if (!script?.textContent) {
        return normalizeAppConfig(null);
    }

    try {
        return normalizeAppConfig(JSON.parse(script.textContent));
    } catch {
        return normalizeAppConfig(null);
    }
}
