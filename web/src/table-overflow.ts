const TABLE_CONTAINER_CLASS = 'table-scroll-container';
const TABLE_BODY_CLASS = 'table-scroll-body';
const TABLE_ENHANCED_CLASS = 'table-scroll-table';
const TABLE_TOGGLE_CLASS = 'table-scroll-toggle';
const TABLE_OVERFLOW_CLASS = 'is-overflowing';
const TABLE_EXPANDED_CLASS = 'is-expanded';

export function enhanceTablesInContent(contentRoot: HTMLElement) {
    const tables = Array.from(contentRoot.querySelectorAll('table'));

    for (const table of tables) {
        if (!(table instanceof HTMLElement)) {
            continue;
        }
        if (table.closest(`.${TABLE_CONTAINER_CLASS}`)) {
            continue;
        }

        const parent = table.parentNode;
        if (!parent) {
            continue;
        }
        table.classList.add(TABLE_ENHANCED_CLASS);

        const container = document.createElement('div');
        container.className = TABLE_CONTAINER_CLASS;

        const body = document.createElement('div');
        body.className = TABLE_BODY_CLASS;

        const toggle = document.createElement('button');
        toggle.type = 'button';
        toggle.className = TABLE_TOGGLE_CLASS;
        toggle.textContent = '︾ 展开完整表格';
        toggle.setAttribute('aria-expanded', 'false');

        toggle.addEventListener('click', () => {
            const expanded = container.classList.toggle(TABLE_EXPANDED_CLASS);
            toggle.setAttribute('aria-expanded', String(expanded));
            toggle.textContent = expanded ? '︽ 收起表格' : '︾ 展开完整表格';
            updateTableOverflow(container);
        });

        parent.insertBefore(container, table);
        body.appendChild(table);
        container.appendChild(body);
        container.appendChild(toggle);

        updateTableOverflow(container);
        window.requestAnimationFrame?.(() => updateTableOverflow(container));
        observeTableOverflow(container, body, table);
    }
}

export function updateTableOverflow(container: HTMLElement) {
    const body = container.querySelector(`.${TABLE_BODY_CLASS}`);
    const elementCtor = container.ownerDocument.defaultView?.HTMLElement;
    if (!elementCtor || !(body instanceof elementCtor)) {
        return;
    }

    if (container.classList.contains(TABLE_EXPANDED_CLASS)) {
        container.classList.remove(TABLE_OVERFLOW_CLASS);
        return;
    }

    // clientHeight 是被 max-height 截断后的可视高度；scrollHeight 更大时才显示展开区。
    const hasVerticalOverflow = body.scrollHeight > body.clientHeight + 1;
    container.classList.toggle(TABLE_OVERFLOW_CLASS, hasVerticalOverflow);
}

function observeTableOverflow(container: HTMLElement, body: HTMLElement, table: HTMLElement) {
    if (typeof ResizeObserver === 'undefined') {
        return;
    }

    const observer = new ResizeObserver(() => updateTableOverflow(container));
    observer.observe(body);
    observer.observe(table);
}
