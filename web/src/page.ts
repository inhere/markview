export interface PageSnapshot {
    title: string;
    contentHTML: string;
    fileMetaHTML: string;
    fileTreeJSON?: string;
    currentFilePathJSON: string;
}

export interface PageMountSelectors {
    contentSelector: string;
    fileMetaSelector: string;
    fileTreeDataId: string;
    currentFilePathDataId: string;
}

export function parsePageSnapshot(html: string, selectors: PageMountSelectors): PageSnapshot {
    const parser = new DOMParser();
    const nextDocument = parser.parseFromString(html, 'text/html');

    const content = nextDocument.querySelector(selectors.contentSelector);
    const fileMeta = nextDocument.querySelector(selectors.fileMetaSelector);
    const fileTreeScript = nextDocument.getElementById(selectors.fileTreeDataId);
    const currentFilePathScript = nextDocument.getElementById(selectors.currentFilePathDataId);

    if (!(content instanceof HTMLElement)) {
        throw new Error('Missing content node in fetched page');
    }
    if (!(fileMeta instanceof HTMLElement)) {
        throw new Error('Missing file meta node in fetched page');
    }
    if (!currentFilePathScript?.textContent) {
        throw new Error('Missing current file path data in fetched page');
    }

    let docTitle : string;
    // 获取第一个 h1 标题作为文档标题
    const h1 = content.querySelector('h1');
    if (h1) {
        docTitle = h1.textContent || nextDocument.title;
    } else {
        docTitle = nextDocument.title;
    }

    return {
        title: docTitle,
        contentHTML: content.innerHTML,
        fileMetaHTML: fileMeta.innerHTML,
        fileTreeJSON: fileTreeScript?.textContent,
        currentFilePathJSON: currentFilePathScript.textContent,
    };
}

export function applyPageSnapshot(snapshot: PageSnapshot, selectors: PageMountSelectors) {
    const content = document.querySelector(selectors.contentSelector);
    const fileMeta = document.querySelector(selectors.fileMetaSelector);
    const fileTreeScript = document.getElementById(selectors.fileTreeDataId);
    const currentFilePathScript = document.getElementById(selectors.currentFilePathDataId);

    if (!(content instanceof HTMLElement) || !(fileMeta instanceof HTMLElement) || !fileTreeScript || !currentFilePathScript) {
        throw new Error('Missing current page mount points');
    }

    document.title = snapshot.title;
    content.innerHTML = snapshot.contentHTML;
    fileMeta.innerHTML = snapshot.fileMetaHTML;
    if (snapshot.fileTreeJSON) {
        fileTreeScript.textContent = snapshot.fileTreeJSON;
    }
    currentFilePathScript.textContent = snapshot.currentFilePathJSON;
}
