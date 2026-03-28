import { describe, expect, test } from 'bun:test';
import {
    buildMermaidContainerId,
    parseMermaidContainerIndex,
} from './mermaid';

describe('frontend mermaid helpers', () => {
    test('buildMermaidContainerId uses stable prefix', () => {
        expect(buildMermaidContainerId(12)).toBe('mermaid-12');
    });

    test('parseMermaidContainerIndex extracts numeric suffix', () => {
        expect(parseMermaidContainerIndex('mermaid-7')).toBe(7);
        expect(parseMermaidContainerIndex('mermaid-invalid')).toBeNull();
    });
});
