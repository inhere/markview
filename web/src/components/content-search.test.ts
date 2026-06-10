import { describe, expect, test } from 'bun:test';
import { JSDOM } from 'jsdom';
import { renderResults } from './content-search';
import type { SearchResponse } from './content-search';

/**
 * content-search.ts 搜索结果文案渲染测试
 * 
 * 测试目标：
 * 1. matches=[] 时（纯 exclude 查询），match-count 应显示 "file match" 而非 "0"
 * 2. matches 非空时，match-count 应显示实际匹配数量
 * 
 * 当前实现：第88行直接渲染 ${result.matches.length}
 * 预期失败：matches=[] 会显示 "0" 而非 "file match"
 */

describe('renderResults match-count display', () => {
    // 创建最小 DOM 环境用于渲染测试
    const dom = new JSDOM('<!DOCTYPE html><div id="container"></div>');
    const container = dom.window.document.getElementById('container') as HTMLElement;

    test('空 matches 时 match-count 显示 "file match" 而非 "0"', () => {
        // 模拟纯 exclude 查询结果：有结果文件但 matches 为空
        const response: SearchResponse = {
            query: '!vendor',
            results: [
                { file: 'README.md', matches: [] },  // 空 matches - 纯 exclude 查询
            ],
            total: 1,  // 有一个结果文件
        };

        // 执行渲染
        renderResults(response, container);

        // 获取渲染后的 match-count 元素
        const matchCountEl = container.querySelector('.match-count');
        expect(matchCountEl).not.toBeNull();

        // 核心断言：空 matches 时应显示 "file match"，而非 "0"
        // 当前实现会失败，因为它直接显示 matches.length = 0
        expect(matchCountEl?.textContent).toBe('file match');
        // 验证不应显示 "0"
        expect(matchCountEl?.textContent).not.toBe('0');
    });

    test('非空 matches 时 match-count 显示实际数量', () => {
        // 模拟普通搜索结果：有匹配内容
        const response: SearchResponse = {
            query: 'markdown',
            results: [
                { 
                    file: 'README.md', 
                    matches: [
                        { line: 10, snippet: 'markdown preview' },
                        { line: 25, snippet: 'markdown syntax' },
                    ] 
                },  // 2 个 matches
            ],
            total: 2,
        };

        // 执行渲染
        renderResults(response, container);

        // 获取渲染后的 match-count 元素
        const matchCountEl = container.querySelector('.match-count');
        expect(matchCountEl).not.toBeNull();

        // 核心断言：非空 matches 时应显示实际数量
        expect(matchCountEl?.textContent).toBe('2');
    });
});