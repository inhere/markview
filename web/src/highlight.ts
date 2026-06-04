import hljs from 'highlight.js/lib/core';
import javascript from 'highlight.js/lib/languages/javascript';
import typescript from 'highlight.js/lib/languages/typescript';
import xml from 'highlight.js/lib/languages/xml';
import css from 'highlight.js/lib/languages/css';
import json from 'highlight.js/lib/languages/json';
import bash from 'highlight.js/lib/languages/bash';
import go from 'highlight.js/lib/languages/go';
import markdown from 'highlight.js/lib/languages/markdown';
import yaml from 'highlight.js/lib/languages/yaml';
import sql from 'highlight.js/lib/languages/sql';
import python from 'highlight.js/lib/languages/python';
import rust from 'highlight.js/lib/languages/rust';
import dart from 'highlight.js/lib/languages/dart';
import plaintext from 'highlight.js/lib/languages/plaintext';
import powershell from 'highlight.js/lib/languages/powershell';
import ini from 'highlight.js/lib/languages/ini';

let highlightReady = false;

export function ensureHighlightLanguages() {
    if (highlightReady) {
        return;
    }

    hljs.registerLanguage('javascript', javascript);
    hljs.registerLanguage('typescript', typescript);
    hljs.registerLanguage('xml', xml);
    hljs.registerLanguage('css', css);
    hljs.registerLanguage('json', json);
    hljs.registerLanguage('bash', bash);
    hljs.registerLanguage('go', go);
    hljs.registerLanguage('markdown', markdown);
    hljs.registerLanguage('yaml', yaml);
    hljs.registerLanguage('toml', ini);
    hljs.registerLanguage('sql', sql);
    hljs.registerLanguage('python', python);
    hljs.registerLanguage('rust', rust);
    hljs.registerLanguage('dart', dart);
    hljs.registerLanguage('plaintext', plaintext);
    hljs.registerLanguage('powershell', powershell);
    hljs.registerAliases(['js', 'jsx', 'mjs', 'cjs'], { languageName: 'javascript' });
    hljs.registerAliases(['ts', 'tsx'], { languageName: 'typescript' });
    hljs.registerAliases(['html', 'vue'], { languageName: 'xml' });
    hljs.registerAliases(['sh', 'shell', 'zsh'], { languageName: 'bash' });
    hljs.registerAliases(['yml'], { languageName: 'yaml' });
    hljs.registerAliases(['text', 'txt'], { languageName: 'plaintext' });
    hljs.registerAliases(['ps1'], { languageName: 'powershell' });
    hljs.registerAliases(['conf', 'cfg'], { languageName: 'ini' });

    highlightReady = true;
}

export function safeHighlightElement(block: HTMLElement) {
    ensureHighlightLanguages();

    const languageClass = Array.from(block.classList)
        .find(className => className.startsWith('language-'));
    const languageName = languageClass?.replace('language-', '');

    if (languageName && !hljs.getLanguage(languageName)) {
        block.classList.remove(languageClass);
        block.classList.add('language-plaintext');
    }

    hljs.highlightElement(block);
}

export { hljs };
