import { describe, expect, test } from 'bun:test';
import { projectURL } from './project-url';

describe('projectURL', () => {
    const basePath = '/p/aaaaaaaaaaaa';

    test('prefixes project paths once and preserves query and hash', () => {
        expect(projectURL('/docs/a.md?q=main#heading', basePath))
            .toBe('/p/aaaaaaaaaaaa/docs/a.md?q=main#heading');
        expect(projectURL('/p/aaaaaaaaaaaa/docs/a.md', basePath))
            .toBe('/p/aaaaaaaaaaaa/docs/a.md');
        expect(projectURL('/', basePath)).toBe('/p/aaaaaaaaaaaa/');
    });

    test('supports relative paths and empty single-project base', () => {
        expect(projectURL('docs/a.md', basePath)).toBe('/p/aaaaaaaaaaaa/docs/a.md');
        expect(projectURL('/docs/a.md', '')).toBe('/docs/a.md');
    });

    test('leaves global and external URLs unchanged', () => {
        expect(projectURL('/static/app.js', basePath)).toBe('/static/app.js');
        expect(projectURL('/favicon.ico', basePath)).toBe('/favicon.ico');
        expect(projectURL('https://example.com/a', basePath)).toBe('https://example.com/a');
        expect(projectURL('//cdn.example.com/a', basePath)).toBe('//cdn.example.com/a');
        expect(projectURL('mailto:docs@example.com', basePath)).toBe('mailto:docs@example.com');
        expect(projectURL('#section', basePath)).toBe('#section');
    });
});
