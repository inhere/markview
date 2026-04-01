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
    if (!fileTreeScript?.textContent) {
        throw new Error('Missing file tree data in fetched page');
    }
    if (!currentFilePathScript?.textContent) {
        throw new Error('Missing current file path data in fetched page');
    }

    return {
        title: nextDocument.title,
        contentHTML: content.innerHTML,
        fileMetaHTML: fileMeta.innerHTML,
        fileTreeJSON: fileTreeScript.textContent,
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
