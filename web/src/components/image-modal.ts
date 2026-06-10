declare global {
    interface Window {
        openImageModal: (src: string, alt?: string) => void;
        closeImageModal: () => void;
    }
}

let modalSetupCompleted = false;
let currentZoom = 1.0;

const minZoom = 0.3;
const maxZoom = 3.0;

export function enhanceImagesInContent(contentRoot: HTMLElement) {
    const images = contentRoot.querySelectorAll('img');
    
    for (const img of images) {
        if (!(img instanceof HTMLImageElement)) continue;
        if (img.closest('.image-container')) continue;
        
        const container = document.createElement('div');
        container.className = 'image-container';
        
        const actions = document.createElement('div');
        actions.className = 'image-actions';
        
        const fullscreenBtn = document.createElement('button');
        fullscreenBtn.className = 'image-fullscreen-btn';
        fullscreenBtn.title = '全屏查看';
        fullscreenBtn.innerHTML = `
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M15 3h6v6M9 21H3v-6M21 3l-7 7M3 21l7-7"/>
            </svg>
        `;
        
        fullscreenBtn.onclick = () => {
            openImageModal(img.src, img.alt);
        };
        
        actions.appendChild(fullscreenBtn);
        
        img.parentNode?.insertBefore(container, img);
        container.appendChild(img);
        container.appendChild(actions);
    }
}

export function setupImageModal() {
    if (modalSetupCompleted) return;
    
    window.openImageModal = openImageModal;
    window.closeImageModal = closeImageModal;
    
    const modal = document.getElementById('image-modal');
    if (modal) {
        modal.addEventListener('click', (e) => {
            if (e.target === modal || e.target === modal.querySelector('.image-modal-content')) {
                closeImageModal();
            }
        });
    }
    
    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') {
            closeImageModal();
        }
    });
    
    modalSetupCompleted = true;
}

function openImageModal(src: string, alt?: string) {
    const modal = document.getElementById('image-modal');
    const content = document.getElementById('image-modal-content');
    const closeBtn = document.getElementById('image-modal-close');
    
    if (!modal || !content) return;
    
    currentZoom = 1.0;
    
    content.innerHTML = '';
    
    const img = document.createElement('img');
    img.src = src;
    if (alt) img.alt = alt;
    img.style.maxWidth = '100%';
    img.style.maxHeight = '90vh';
    img.style.transform = 'scale(1)';
    img.style.transition = 'transform 0.15s ease-out';
    
    content.appendChild(img);
    
    let controls = document.getElementById('image-modal-controls');
    if (controls) controls.remove();
    
    controls = document.createElement('div');
    controls.id = 'image-modal-controls';
    controls.className = 'image-modal-controls';
    controls.innerHTML = `
        <button class="image-control-btn" id="image-zoom-out" title="缩小">−</button>
        <span class="image-zoom-level" id="image-zoom-level">100%</span>
        <button class="image-control-btn" id="image-zoom-in" title="放大">+</button>
        <span class="image-ctrl-divider"></span>
        <button class="image-control-btn" data-zoom="0.5" title="50%">50%</button>
        <button class="image-control-btn active" data-zoom="1" title="100%">100%</button>
        <button class="image-control-btn" data-zoom="1.5" title="150%">150%</button>
        <button class="image-control-btn" data-zoom="2" title="200%">200%</button>
    `;
    
    modal.appendChild(controls);
    
    controls.querySelectorAll('[data-zoom]').forEach(btn => {
        btn.addEventListener('click', (e) => {
            e.stopPropagation();
            const zoom = parseFloat((btn as HTMLElement).dataset.zoom || '1');
            currentZoom = zoom;
            updateImageZoom(img, currentZoom);
            controls!.querySelectorAll('[data-zoom]').forEach(node => node.classList.remove('active'));
            btn.classList.add('active');
        });
    });
    
    const zoomInBtn = document.getElementById('image-zoom-in');
    const zoomOutBtn = document.getElementById('image-zoom-out');
    
    zoomInBtn?.addEventListener('click', (e) => {
        e.stopPropagation();
        if (currentZoom < maxZoom) {
            currentZoom = Math.min(maxZoom, Math.round((currentZoom + 0.1) * 10) / 10);
            updateImageZoom(img, currentZoom);
            syncZoomPresetButtons(controls!);
        }
    });
    
    zoomOutBtn?.addEventListener('click', (e) => {
        e.stopPropagation();
        if (currentZoom > minZoom) {
            currentZoom = Math.max(minZoom, Math.round((currentZoom - 0.1) * 10) / 10);
            updateImageZoom(img, currentZoom);
            syncZoomPresetButtons(controls!);
        }
    });
    
    closeBtn?.addEventListener('click', (e) => {
        e.stopPropagation();
        closeImageModal();
    });
    
    modal.classList.add('active');
    document.body.style.overflow = 'hidden';
}

function closeImageModal() {
    const modal = document.getElementById('image-modal');
    if (modal) {
        modal.classList.remove('active');
        document.body.style.overflow = '';
        
        const controls = document.getElementById('image-modal-controls');
        if (controls) controls.remove();
    }
}

function updateImageZoom(img: HTMLImageElement, zoom: number) {
    const label = document.getElementById('image-zoom-level');
    if (label) label.textContent = `${Math.round(zoom * 100)}%`;
    
    img.style.transform = `scale(${zoom})`;
    
    if (zoom > 1) {
        img.style.maxWidth = `${95 / zoom}vw`;
        img.style.maxHeight = `${90 / zoom}vh`;
    } else {
        img.style.maxWidth = '100%';
        img.style.maxHeight = '90vh';
    }
}

function syncZoomPresetButtons(controls: HTMLElement) {
    controls.querySelectorAll('[data-zoom]').forEach(btn => {
        const zoom = parseFloat((btn as HTMLElement).dataset.zoom || '1');
        btn.classList.toggle('active', Math.abs(zoom - currentZoom) < 0.05);
    });
}