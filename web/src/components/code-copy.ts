// web/src/code-copy.ts

export function enhanceCodeBlocks(contentRoot: HTMLElement): void {
    const codeBlocks = contentRoot.querySelectorAll('pre code');
    
    codeBlocks.forEach(block => {
        if (!(block instanceof HTMLElement)) {
            return;
        }
        
        if (block.classList.contains('language-mermaid')) {
            return;
        }
        
        const pre = block.parentElement;
        if (!pre || pre.querySelector('.code-copy-btn')) {
            return;
        }
        
        const copyBtn = createCopyButton();
        pre.appendChild(copyBtn);
        
        if (pre.style.position !== 'relative') {
            pre.style.position = 'relative';
        }
    });
}

function createCopyButton(): HTMLElement {
    const btn = document.createElement('button');
    btn.className = 'code-copy-btn';
    btn.type = 'button';
    btn.title = '复制代码';
    btn.setAttribute('aria-label', '复制代码到剪贴板');
    
    btn.innerHTML = `
        <svg class="copy-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
            <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
        </svg>
        <svg class="check-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="display: none;">
            <polyline points="20 6 9 17 4 12"></polyline>
        </svg>
    `;
    
    btn.addEventListener('click', async (e) => {
        e.preventDefault();
        e.stopPropagation();
        
        const pre = btn.parentElement;
        const code = pre?.querySelector('code');
        
        if (!code) {
            return;
        }
        
        const text = code.textContent || '';
        try {
            await copyToClipboard(text);
            showCopySuccess(btn);
        } catch (err) {
            console.error('复制失败:', err);
        }
    });
    
    return btn;
}

async function copyToClipboard(text: string): Promise<void> {
    if (navigator.clipboard && navigator.clipboard.writeText) {
        await navigator.clipboard.writeText(text);
    } else {
        fallbackCopyToClipboard(text);
    }
}

function fallbackCopyToClipboard(text: string): void {
    const textarea = document.createElement('textarea');
    textarea.value = text;
    textarea.style.position = 'fixed';
    textarea.style.opacity = '0';
    textarea.style.left = '-9999px';
    document.body.appendChild(textarea);
    textarea.select();
    document.execCommand('copy');
    document.body.removeChild(textarea);
}

function showCopySuccess(btn: HTMLElement): void {
    const copyIcon = btn.querySelector('.copy-icon') as SVGElement;
    const checkIcon = btn.querySelector('.check-icon') as SVGElement;
    
    if (!copyIcon || !checkIcon) {
        return;
    }
    
    copyIcon.style.display = 'none';
    checkIcon.style.display = 'block';
    
    btn.classList.add('copied');
    btn.title = '已复制!';
    
    setTimeout(() => {
        copyIcon.style.display = 'block';
        checkIcon.style.display = 'none';
        btn.classList.remove('copied');
        btn.title = '复制代码';
    }, 2000);
}