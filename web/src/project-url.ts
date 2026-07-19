import { readAppConfig } from './app-config';

const EXTERNAL_URL = /^(?:[a-z][a-z0-9+.-]*:|#|\/\/)/i;

export function projectURL(path: string, basePath = readAppConfig().basePath): string {
    if (!basePath || EXTERNAL_URL.test(path) || path === '/favicon.ico' || path.startsWith('/static/')) {
        return path;
    }

    const queryAt = path.indexOf('?');
    const hashAt = path.indexOf('#');
    const suffixAt = [queryAt, hashAt]
        .filter(index => index >= 0)
        .reduce((smallest, index) => Math.min(smallest, index), path.length);
    const pathname = path.slice(0, suffixAt);
    if (pathname === basePath || pathname.startsWith(`${basePath}/`)) {
        return path;
    }

    const suffix = path.slice(suffixAt);
    return `${basePath}/${pathname.replace(/^\/+/, '')}${suffix}`;
}
