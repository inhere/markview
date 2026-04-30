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
import plaintext from 'highlight.js/lib/languages/plaintext';
import powershell from 'highlight.js/lib/languages/powershell';

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
    hljs.registerLanguage('sql', sql);
    hljs.registerLanguage('python', python);
    hljs.registerLanguage('rust', rust);
    hljs.registerLanguage('plaintext', plaintext);
    hljs.registerLanguage('powershell', powershell);

    highlightReady = true;
}

export { hljs };