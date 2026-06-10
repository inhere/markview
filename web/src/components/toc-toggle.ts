import type { AppLayout } from '../app-config';

interface TocToggleOptions {
    documentRef?: Document;
}

function readLayoutMode(documentRef: Document): AppLayout | null {
    const layout = documentRef.documentElement.dataset.layout;
    if (layout === 'compact' || layout === 'toc-middle' || layout === 'toc-right') {
        return layout;
    }
    return null;
}

function setExpandedState(documentRef: Document, expanded: boolean) {
    documentRef.body.classList.toggle('toc-floating-open', expanded);

    const button = documentRef.querySelector('.toc-section-toggle');
    if (button) {
        button.setAttribute('aria-expanded', String(expanded));
    }
}

function isPreviewActive(documentRef: Document) {
    return documentRef.body.classList.contains('preview-active');
}

function syncTocToggleDefaultState(documentRef: Document) {
    const layout = readLayoutMode(documentRef);
    if (!layout) {
        setExpandedState(documentRef, false);
        return;
    }

    if (isPreviewActive(documentRef)) {
        setExpandedState(documentRef, false);
        return;
    }

    setExpandedState(documentRef, true);
}

export function setupTocToggle({ documentRef = document }: TocToggleOptions) {
    const button = documentRef.querySelector('.toc-section-toggle');
    if (!button) {
        return;
    }

    let activeLayout = readLayoutMode(documentRef);
    let previewActive = isPreviewActive(documentRef);

    syncTocToggleDefaultState(documentRef);

    button.addEventListener('click', () => {
        if (!readLayoutMode(documentRef)) {
            setExpandedState(documentRef, false);
            return;
        }

        const expanded = !documentRef.body.classList.contains('toc-floating-open');
        setExpandedState(documentRef, expanded);
    });

    documentRef.addEventListener('markview:layout-mode-changed', () => {
        const nextLayout = readLayoutMode(documentRef);
        if (nextLayout !== activeLayout) {
            activeLayout = nextLayout;
            syncTocToggleDefaultState(documentRef);
        }
    });

    documentRef.addEventListener('markview:preview-state-changed', () => {
        const nextPreviewActive = isPreviewActive(documentRef);
        const enteredPreview = !previewActive && nextPreviewActive;
        previewActive = nextPreviewActive;
        if (enteredPreview) {
            syncTocToggleDefaultState(documentRef);
        }
    });
}
