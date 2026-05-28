export type AppLayout = 'compact' | 'toc-middle' | 'toc-right';

export interface AppConfig {
    previewExts: string[];
    layout: AppLayout;
}

export const DEFAULT_APP_CONFIG: AppConfig = {
    previewExts: ['.md', '.json', '.jsonl', '.yaml', '.yml', '.toml'],
    layout: 'compact',
};

const CONFIG_SCRIPT_ID = 'app-config-data';
const VALID_LAYOUTS = new Set<AppLayout>(['compact', 'toc-middle', 'toc-right']);

export function normalizePreviewExts(value: unknown): string[] {
    if (!Array.isArray(value)) {
        return [...DEFAULT_APP_CONFIG.previewExts];
    }

    const previewExts = value
        .filter((item): item is string => typeof item === 'string')
        .map(item => item.trim().toLowerCase())
        .filter(Boolean)
        .map(item => item.startsWith('.') ? item : `.${item}`);

    return previewExts.length > 0 ? previewExts : [...DEFAULT_APP_CONFIG.previewExts];
}

function normalizeLayout(value: unknown): AppLayout {
    if (typeof value === 'string' && VALID_LAYOUTS.has(value as AppLayout)) {
        return value as AppLayout;
    }
    return DEFAULT_APP_CONFIG.layout;
}

export function normalizeAppConfig(value: unknown): AppConfig {
    if (!value || typeof value !== 'object') {
        return { ...DEFAULT_APP_CONFIG, previewExts: [...DEFAULT_APP_CONFIG.previewExts] };
    }

    const rawConfig = value as Record<string, unknown>;
    return {
        previewExts: normalizePreviewExts(rawConfig.previewExts),
        layout: normalizeLayout(rawConfig.layout),
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
