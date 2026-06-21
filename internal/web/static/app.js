// State variables
let state = {
    resolution: { width: 1080, height: 1920 },
    layers: [],
    selectedLayerName: null,
    zoom: 0.35,
    isDragging: false,
    isResizing: false,
    activeHandle: null,
    dragStart: { x: 0, y: 0 },
    layerStartCoords: { x: 0, y: 0, w: 0, h: 0 },
    undoStack: [],
    redoStack: []
};

// DOM Elements
const canvasContainer = document.getElementById('canvas-container');
const layerOverlayContainer = document.getElementById('layer-overlay-container');
const previewImage = document.getElementById('preview-image');
const layerList = document.getElementById('layer-list');
const transformPanel = document.getElementById('transform-panel');
const selectedLayerNameDisplay = document.getElementById('selected-layer-name');
const resolutionDisplay = document.getElementById('canvas-resolution');
const statusBadge = document.getElementById('status-badge');
const zoomLabel = document.getElementById('zoom-label');
const toast = document.getElementById('toast');
const btnUndo = document.getElementById('btn-undo');
const btnRedo = document.getElementById('btn-redo');

// Preview mode toggle helper
function setPreviewMode(showPreview) {
    if (showPreview) {
        previewImage.style.opacity = '1';
        document.querySelectorAll('.layer-frame').forEach(frame => {
            const container = frame.querySelector('.layer-image-container');
            if (container) {
                container.style.opacity = '0.0';
            }
        });
    } else {
        previewImage.style.opacity = '0';
        document.querySelectorAll('.layer-frame').forEach(frame => {
            const container = frame.querySelector('.layer-image-container');
            if (container) {
                container.style.opacity = '1.0';
            }
        });
    }
}

// Inputs
const inputX = document.getElementById('pos-x');
const inputY = document.getElementById('pos-y');
const inputW = document.getElementById('dim-w');
const inputH = document.getElementById('dim-h');
const inputZ = document.getElementById('layer-zindex');
const inputScaleMode = document.getElementById('layer-scalemode');
const inputLockAR = document.getElementById('layer-lock-ar');
const cropX = document.getElementById('crop-x');
const cropY = document.getElementById('crop-y');
const cropW = document.getElementById('crop-w');
const cropH = document.getElementById('crop-h');
const cropXVal = document.getElementById('crop-x-val');
const cropYVal = document.getElementById('crop-y-val');
const cropWVal = document.getElementById('crop-w-val');
const cropHVal = document.getElementById('crop-h-val');

// Buttons
const btnRefreshPreview = document.getElementById('btn-refresh-preview');
const btnSaveLayout = document.getElementById('btn-save-layout');
const btnZoomIn = document.getElementById('btn-zoom-in');
const btnZoomOut = document.getElementById('btn-zoom-out');
const btnZoomFit = document.getElementById('btn-zoom-fit');

// Initialize editor on load
window.addEventListener('DOMContentLoaded', () => {
    fetchLayout();
    setupEventListeners();
});

// Fetch layout info from backend
async function fetchLayout() {
    try {
        statusBadge.textContent = 'Loading...';
        const response = await fetch('/api/layout');
        const data = await response.json();
        
        state.resolution = { width: data.width, height: data.height };
        state.layers = data.layers.map(layer => {
            // Default lock_ar to true for gameplay ('game') and facecam ('cam') layers
            layer.lock_ar = (layer.name === 'game' || layer.name === 'cam');
            return layer;
        });
        
        resolutionDisplay.textContent = `${data.width} × ${data.height}`;
        
        // Auto-fit canvas to viewport size
        fitCanvasToViewport();
        
        // Set initial preview src
        refreshPreviewSrc();
        
        // Render layers
        renderLayerList();
        renderLayerFrames();
        updateUndoRedoButtonsState();
        
        statusBadge.textContent = 'Connected';
        statusBadge.className = 'status-badge';
    } catch (err) {
        console.error('Error fetching layout:', err);
        statusBadge.textContent = 'Connection Error';
        statusBadge.style.color = '#ef4444';
    }
}

// Render the list of layers in the sidebar
function renderLayerList() {
    layerList.innerHTML = '';
    
    // Sort display list by Z-Index descending (top layers first)
    const sortedLayers = [...state.layers].sort((a, b) => b.z_index - a.z_index);
    
    sortedLayers.forEach(layer => {
        const item = document.createElement('div');
        item.className = `layer-item ${state.selectedLayerName === layer.name ? 'active' : ''}`;
        item.dataset.name = layer.name;
        
        // Visible icon
        const eyeBtn = document.createElement('button');
        eyeBtn.className = `layer-visibility-toggle ${layer.visible ? '' : 'muted'}`;
        eyeBtn.textContent = layer.visible ? '👁️' : '👁️‍🗨️';
        eyeBtn.addEventListener('click', (e) => {
            e.stopPropagation();
            toggleLayerVisibility(layer.name);
        });
        
        const details = document.createElement('div');
        details.className = 'layer-details';
        
        const titleRow = document.createElement('div');
        titleRow.style.display = 'flex';
        titleRow.style.justifyContent = 'space-between';
        
        const title = document.createElement('span');
        title.className = 'layer-title-text';
        title.textContent = getLayerDisplayName(layer.name);
        
        const badge = document.createElement('span');
        badge.className = 'layer-badge';
        badge.textContent = `z:${layer.z_index}`;
        
        titleRow.appendChild(title);
        titleRow.appendChild(badge);
        
        const coords = document.createElement('span');
        coords.className = 'layer-coords-text';
        coords.textContent = `X:${layer.x} Y:${layer.y} W:${layer.width} H:${layer.height}`;
        
        details.appendChild(titleRow);
        details.appendChild(coords);
        
        item.appendChild(eyeBtn);
        item.appendChild(details);
        
        item.addEventListener('click', () => {
            selectLayer(layer.name);
        });
        
        layerList.appendChild(item);
    });
}

// Translate system names to friendly display names
function getLayerDisplayName(name) {
    switch (name) {
        case 'bg': return 'Background';
        case 'game': return 'Gameplay';
        case 'cam': return 'Facecam';
        case 'overlay': return 'Overlay Overlay';
        default: return name.charAt(0).toUpperCase() + name.slice(1);
    }
}

// Select a layer and open transform controls
function selectLayer(name) {
    state.selectedLayerName = name;
    
    // Highlight sidebar active item
    document.querySelectorAll('.layer-item').forEach(item => {
        if (item.dataset.name === name) {
            item.classList.add('active');
        } else {
            item.classList.remove('active');
        }
    });
    
    // Highlight canvas active frame
    document.querySelectorAll('.layer-frame').forEach(frame => {
        if (frame.dataset.name === name) {
            frame.classList.add('active');
        } else {
            frame.classList.remove('active');
        }
    });
    
    const layer = state.layers.find(l => l.name === name);
    if (layer) {
        selectedLayerNameDisplay.textContent = getLayerDisplayName(layer.name);
        inputX.value = layer.x;
        inputY.value = layer.y;
        inputW.value = layer.width;
        inputH.value = layer.height;
        inputZ.value = layer.z_index;
        
        // Populate Aspect Ratio / Scale Mode and Crop Sliders
        inputScaleMode.value = layer.scale_mode || 'stretch';
        inputLockAR.checked = !!layer.lock_ar;
        
        cropX.value = Math.round((layer.crop_x_percent || 0) * 100);
        cropY.value = Math.round((layer.crop_y_percent || 0) * 100);
        cropW.value = Math.round((layer.crop_width_percent || 1) * 100);
        cropH.value = Math.round((layer.crop_height_percent || 1) * 100);
        
        cropXVal.textContent = `${cropX.value}%`;
        cropYVal.textContent = `${cropY.value}%`;
        cropWVal.textContent = `${cropW.value}%`;
        cropHVal.textContent = `${cropH.value}%`;
        
        transformPanel.style.display = 'block';
    } else {
        transformPanel.style.display = 'none';
    }
}

// Generate layer frame DOM elements over the canvas preview
function renderLayerFrames() {
    layerOverlayContainer.innerHTML = '';
    
    state.layers.forEach(layer => {
        const frame = document.createElement('div');
        frame.className = `layer-frame ${state.selectedLayerName === layer.name ? 'active' : ''} ${layer.visible ? '' : 'hidden'}`;
        frame.dataset.name = layer.name;
        
        // Create image container and image for real-time WYSIWYG
        const imgContainer = document.createElement('div');
        imgContainer.className = 'layer-image-container';
        
        const img = document.createElement('img');
        if (layer.name === 'overlay') {
            img.src = '/api/overlay-image';
        } else {
            img.src = '/api/raw-frame.png';
        }
        imgContainer.appendChild(img);
        frame.appendChild(imgContainer);
        
        // Position layer based on zoom and coordinates
        updateFramePosition(frame, layer);
        
        // Label
        const label = document.createElement('div');
        label.className = 'layer-label';
        label.textContent = getLayerDisplayName(layer.name);
        frame.appendChild(label);
        
        // Resize Handles (NW, NE, SW, SE)
        const handles = ['nw', 'ne', 'sw', 'se'];
        handles.forEach(h => {
            const handle = document.createElement('div');
            handle.className = `resize-handle ${h}`;
            handle.dataset.handle = h;
            frame.appendChild(handle);
        });
        
        // Select on click
        frame.addEventListener('mousedown', (e) => {
            e.stopPropagation();
            selectLayer(layer.name);
            setPreviewMode(false); // Switch to real-time mode
            
            pushStateToHistory();
            
            // Check if clicking handle or frame itself
            if (e.target.classList.contains('resize-handle')) {
                state.isResizing = true;
                state.activeHandle = e.target.dataset.handle;
            } else {
                state.isDragging = true;
            }
            
            state.dragStart = { x: e.clientX, y: e.clientY };
            state.layerStartCoords = { x: layer.x, y: layer.y, w: layer.width, h: layer.height };
        });
        
        layerOverlayContainer.appendChild(frame);
    });
}

// Update DOM element positions/sizing for a layer
function updateFramePosition(frame, layer) {
    frame.style.left = `${layer.x * state.zoom}px`;
    frame.style.top = `${layer.y * state.zoom}px`;
    frame.style.width = `${layer.width * state.zoom}px`;
    frame.style.height = `${layer.height * state.zoom}px`;
    frame.style.zIndex = layer.z_index;
    
    // Update inner image crop and scale mode in real-time
    const img = frame.querySelector('.layer-image-container img');
    if (img) {
        let wPct = layer.crop_width_percent || 1.0;
        let hPct = layer.crop_height_percent || 1.0;
        let xPct = layer.crop_x_percent || 0.0;
        let yPct = layer.crop_y_percent || 0.0;
        
        // Clamp crop coordinates to mimic FFmpeg's internal boundary clamping
        wPct = Math.max(0.01, Math.min(1.0, wPct));
        hPct = Math.max(0.01, Math.min(1.0, hPct));
        xPct = Math.max(0.0, Math.min(1.0 - wPct, xPct));
        yPct = Math.max(0.0, Math.min(1.0 - hPct, yPct));
        
        img.style.width = `${(1.0 / wPct) * 100}%`;
        img.style.height = `${(1.0 / hPct) * 100}%`;
        img.style.left = `${(-xPct / wPct) * 100}%`;
        img.style.top = `${(-yPct / hPct) * 100}%`;
        
        if (layer.name === 'bg') {
            img.style.filter = 'blur(20px)';
            img.style.transform = 'scale(1.1)'; // avoids blur borders
        } else {
            img.style.filter = 'none';
            img.style.transform = 'none';
        }
        
        if (wPct === 1.0 && hPct === 1.0) {
            if (layer.scale_mode === 'fit') {
                img.style.objectFit = 'contain';
            } else if (layer.scale_mode === 'fill') {
                img.style.objectFit = 'cover';
            } else {
                img.style.objectFit = 'fill';
            }
            img.style.width = '100%';
            img.style.height = '100%';
            img.style.left = '0';
            img.style.top = '0';
        } else {
            img.style.objectFit = 'fill';
        }
    }
}

// Toggle layer visibility
async function toggleLayerVisibility(name) {
    const layer = state.layers.find(l => l.name === name);
    if (!layer) return;
    
    pushStateToHistory();
    
    layer.visible = !layer.visible;
    
    // Update local frame
    const frame = document.querySelector(`.layer-frame[data-name="${name}"]`);
    if (frame) {
        if (layer.visible) {
            frame.classList.remove('hidden');
        } else {
            frame.classList.add('hidden');
        }
    }
    
    renderLayerList();
    
    // Persist changes to server
    // (Note: we can send the updated visible flag. The API position/size can do it,
    // let's define visibility updates directly or let it be saved by saving the state).
    // Let's call /api/layout/position to update visibility in-memory or create a general endpoint.
    // In our plan we have specific routes. Let's make POST `/api/layout/position` also set visible and zIndex if we want,
    // or just let backend track it in-memory.
    // Let's send a layout update payload.
    saveLayerVisibilityAndZ(layer);
}

// Zoom / Scale calculations
function applyZoom() {
    canvasContainer.style.width = `${state.resolution.width * state.zoom}px`;
    canvasContainer.style.height = `${state.resolution.height * state.zoom}px`;
    zoomLabel.textContent = `${Math.round(state.zoom * 100)}%`;
    
    // Re-render frames with new zoom layout
    state.layers.forEach(layer => {
        const frame = document.querySelector(`.layer-frame[data-name="${layer.name}"]`);
        if (frame) {
            updateFramePosition(frame, layer);
        }
    });
}

function fitCanvasToViewport() {
    const viewport = document.getElementById('canvas-viewport');
    const viewportWidth = viewport.clientWidth - 80;
    const viewportHeight = viewport.clientHeight - 80;
    
    const scaleX = viewportWidth / state.resolution.width;
    const scaleY = viewportHeight / state.resolution.height;
    
    state.zoom = Math.min(scaleX, scaleY);
    // Keep zoom within bounds
    state.zoom = Math.max(0.1, Math.min(1.5, state.zoom));
    
    applyZoom();
}

function refreshPreviewSrc() {
    previewImage.src = `/api/preview.png?t=${Date.now()}`;
}

function pushStateToHistory() {
    const layersClone = JSON.parse(JSON.stringify(state.layers));
    if (state.undoStack.length > 0) {
        const lastState = JSON.stringify(state.undoStack[state.undoStack.length - 1]);
        const currentState = JSON.stringify(layersClone);
        if (lastState === currentState) {
            return;
        }
    }
    state.undoStack.push(layersClone);
    if (state.undoStack.length > 50) {
        state.undoStack.shift();
    }
    state.redoStack = [];
    updateUndoRedoButtonsState();
}

function undo() {
    if (state.undoStack.length === 0) return;
    const currentClone = JSON.parse(JSON.stringify(state.layers));
    state.redoStack.push(currentClone);
    state.layers = state.undoStack.pop();
    applyRestoredState('Undone change');
}

function redo() {
    if (state.redoStack.length === 0) return;
    const currentClone = JSON.parse(JSON.stringify(state.layers));
    state.undoStack.push(currentClone);
    state.layers = state.redoStack.pop();
    applyRestoredState('Redone change');
}

function applyRestoredState(msg) {
    renderLayerList();
    renderLayerFrames();
    if (state.selectedLayerName) {
        selectLayer(state.selectedLayerName);
    }
    state.layers.forEach(layer => {
        saveLayerPosition(layer);
        saveLayerSize(layer);
    });
    setPreviewMode(false);
    showToast(msg);
    updateUndoRedoButtonsState();
}

function updateUndoRedoButtonsState() {
    if (btnUndo) {
        btnUndo.disabled = state.undoStack.length === 0;
        btnUndo.style.opacity = state.undoStack.length === 0 ? '0.3' : '1';
        btnUndo.style.cursor = state.undoStack.length === 0 ? 'not-allowed' : 'pointer';
    }
    if (btnRedo) {
        btnRedo.disabled = state.redoStack.length === 0;
        btnRedo.style.opacity = state.redoStack.length === 0 ? '0.3' : '1';
        btnRedo.style.cursor = state.redoStack.length === 0 ? 'not-allowed' : 'pointer';
    }
}

// Mouse dragging and resizing handlers
function setupEventListeners() {
    // Mouse move on viewport
    window.addEventListener('mousemove', (e) => {
        if (!state.isDragging && !state.isResizing) return;
        
        const dx = (e.clientX - state.dragStart.x) / state.zoom;
        const dy = (e.clientY - state.dragStart.y) / state.zoom;
        
        const layer = state.layers.find(l => l.name === state.selectedLayerName);
        if (!layer) return;
        
        if (state.isDragging) {
            // Move layer
            layer.x = Math.round(state.layerStartCoords.x + dx);
            layer.y = Math.round(state.layerStartCoords.y + dy);
        } else if (state.isResizing) {
            // Resize layer
            const handle = state.activeHandle;
            const start = state.layerStartCoords;
            const startAspect = start.w / start.h;
            const lockAR = layer.lock_ar;
            
            if (lockAR) {
                if (handle === 'se') {
                    const newWidth = Math.max(10, start.w + dx);
                    const newHeight = newWidth / startAspect;
                    layer.width = Math.round(newWidth);
                    layer.height = Math.round(newHeight);
                } else if (handle === 'sw') {
                    const newWidth = Math.max(10, start.w - dx);
                    const newHeight = newWidth / startAspect;
                    const actualDx = start.w - newWidth;
                    layer.x = Math.round(start.x + actualDx);
                    layer.width = Math.round(newWidth);
                    layer.height = Math.round(newHeight);
                } else if (handle === 'ne') {
                    const newWidth = Math.max(10, start.w + dx);
                    const newHeight = newWidth / startAspect;
                    const actualDy = start.h - newHeight;
                    layer.y = Math.round(start.y + actualDy);
                    layer.width = Math.round(newWidth);
                    layer.height = Math.round(newHeight);
                } else if (handle === 'nw') {
                    const newWidth = Math.max(10, start.w - dx);
                    const newHeight = newWidth / startAspect;
                    const actualDx = start.w - newWidth;
                    const actualDy = start.h - newHeight;
                    layer.x = Math.round(start.x + actualDx);
                    layer.y = Math.round(start.y + actualDy);
                    layer.width = Math.round(newWidth);
                    layer.height = Math.round(newHeight);
                }
            } else {
                if (handle === 'se') {
                    layer.width = Math.round(Math.max(10, start.w + dx));
                    layer.height = Math.round(Math.max(10, start.h + dy));
                } else if (handle === 'sw') {
                    const newWidth = Math.max(10, start.w - dx);
                    const actualDx = start.w - newWidth;
                    layer.x = Math.round(start.x + actualDx);
                    layer.width = Math.round(newWidth);
                    layer.height = Math.round(Math.max(10, start.h + dy));
                } else if (handle === 'ne') {
                    layer.width = Math.round(Math.max(10, start.w + dx));
                    const newHeight = Math.max(10, start.h - dy);
                    const actualDy = start.h - newHeight;
                    layer.y = Math.round(start.y + actualDy);
                    layer.height = Math.round(newHeight);
                } else if (handle === 'nw') {
                    const newWidth = Math.max(10, start.w - dx);
                    const actualDx = start.w - newWidth;
                    layer.x = Math.round(start.x + actualDx);
                    layer.width = Math.round(newWidth);
                    
                    const newHeight = Math.max(10, start.h - dy);
                    const actualDy = start.h - newHeight;
                    layer.y = Math.round(start.y + actualDy);
                    layer.height = Math.round(newHeight);
                }
            }
        }
        
        // Update DOM visuals instantly
        const frame = document.querySelector(`.layer-frame[data-name="${layer.name}"]`);
        if (frame) {
            updateFramePosition(frame, layer);
        }
        
        // Update transform values in panel
        inputX.value = layer.x;
        inputY.value = layer.y;
        inputW.value = layer.width;
        inputH.value = layer.height;
        
        // Update layer details in sidebar text
        renderLayerList();
        
        setPreviewMode(false); // Switch to real-time interactive mode
    });
    
    // Mouse up
    window.addEventListener('mouseup', () => {
        if (state.isDragging || state.isResizing) {
            const layer = state.layers.find(l => l.name === state.selectedLayerName);
            if (layer) {
                if (state.isDragging) {
                    saveLayerPosition(layer);
                } else if (state.isResizing) {
                    saveLayerSize(layer);
                }
            }
        }
        state.isDragging = false;
        state.isResizing = false;
        state.activeHandle = null;
    });
    
    // Transform input box changes
    const triggerManualUpdate = (e) => {
        const layer = state.layers.find(l => l.name === state.selectedLayerName);
        if (!layer) return;
        
        pushStateToHistory();
        
        const target = e ? e.target : null;
        
        let newX = parseInt(inputX.value) || 0;
        let newY = parseInt(inputY.value) || 0;
        let newW = parseInt(inputW.value) || 10;
        let newH = parseInt(inputH.value) || 10;
        
        if (layer.lock_ar && layer.width > 0 && layer.height > 0) {
            const aspect = layer.width / layer.height;
            if (target === inputW) {
                newH = Math.round(newW / aspect);
                inputH.value = newH;
            } else if (target === inputH) {
                newW = Math.round(newH * aspect);
                inputW.value = newW;
            }
        }
        
        layer.x = newX;
        layer.y = newY;
        layer.width = newW;
        layer.height = newH;
        layer.z_index = parseInt(inputZ.value) || 0;
        layer.scale_mode = inputScaleMode.value;
        
        const frame = document.querySelector(`.layer-frame[data-name="${layer.name}"]`);
        if (frame) {
            updateFramePosition(frame, layer);
        }
        
        renderLayerList();
        saveLayerPosition(layer);
        saveLayerSize(layer);
        setPreviewMode(false); // Switch to real-time interactive mode
    };
    
    [inputX, inputY, inputW, inputH, inputZ, inputScaleMode].forEach(input => {
        input.addEventListener('change', triggerManualUpdate);
    });
    
    inputLockAR.addEventListener('change', () => {
        const layer = state.layers.find(l => l.name === state.selectedLayerName);
        if (layer) {
            layer.lock_ar = inputLockAR.checked;
        }
    });
    
    // Crop sliders updates
    const triggerCropUpdate = (e) => {
        const layer = state.layers.find(l => l.name === state.selectedLayerName);
        if (!layer) return;
        
        let cx = parseFloat(cropX.value) || 0;
        let cy = parseFloat(cropY.value) || 0;
        let cw = parseFloat(cropW.value) || 100;
        let ch = parseFloat(cropH.value) || 100;
        
        const target = e ? e.target : null;
        
        if (target === cropX) {
            if (cx + cw > 100) {
                cx = 100 - cw;
                cropX.value = cx;
            }
        } else if (target === cropY) {
            if (cy + ch > 100) {
                cy = 100 - ch;
                cropY.value = cy;
            }
        } else if (target === cropW) {
            if (cx + cw > 100) {
                cw = 100 - cx;
                cropW.value = cw;
            }
        } else if (target === cropH) {
            if (cy + ch > 100) {
                ch = 100 - cy;
                cropH.value = ch;
            }
        }
        
        layer.crop_x_percent = cx / 100;
        layer.crop_y_percent = cy / 100;
        layer.crop_width_percent = cw / 100;
        layer.crop_height_percent = ch / 100;
        
        cropXVal.textContent = `${cx}%`;
        cropYVal.textContent = `${cy}%`;
        cropWVal.textContent = `${cw}%`;
        cropHVal.textContent = `${ch}%`;
        
        const frame = document.querySelector(`.layer-frame[data-name="${layer.name}"]`);
        if (frame) {
            updateFramePosition(frame, layer);
        }
        
        renderLayerList();
        saveLayerPosition(layer);
        setPreviewMode(false); // Switch to real-time interactive mode
    };
    
    [cropX, cropY, cropW, cropH].forEach(input => {
        input.addEventListener('input', triggerCropUpdate);
    });
    
    // Keyboard arrow keys nudge
    window.addEventListener('keydown', (e) => {
        if (!state.selectedLayerName) return;
        
        // Skip if typing in an input
        if (document.activeElement.tagName === 'INPUT') return;
        
        const layer = state.layers.find(l => l.name === state.selectedLayerName);
        if (!layer) return;
        
        const step = e.shiftKey ? 10 : 1;
        let moved = false;
        
        if (e.key === 'ArrowLeft') {
            layer.x -= step;
            moved = true;
        } else if (e.key === 'ArrowRight') {
            layer.x += step;
            moved = true;
        } else if (e.key === 'ArrowUp') {
            layer.y -= step;
            moved = true;
        } else if (e.key === 'ArrowDown') {
            layer.y += step;
            moved = true;
        }
        
        if (moved) {
            e.preventDefault();
            if (!e.repeat) {
                pushStateToHistory();
            }
            
            // Update input panel
            inputX.value = layer.x;
            inputY.value = layer.y;
            
            // Update DOM frame and sidebar
            const frame = document.querySelector(`.layer-frame[data-name="${layer.name}"]`);
            if (frame) {
                updateFramePosition(frame, layer);
            }
            renderLayerList();
            
            // Send position update
            saveLayerPosition(layer);
            setPreviewMode(false); // Switch to real-time interactive mode
        }
    });
    
    // Action buttons
    btnRefreshPreview.addEventListener('click', async () => {
        try {
            btnRefreshPreview.disabled = true;
            btnRefreshPreview.textContent = 'Generating...';
            
            const response = await fetch('/api/layout/preview', {
                method: 'POST'
            });
            const data = await response.json();
            
            if (data.status === 'success') {
                refreshPreviewSrc();
                setPreviewMode(true); // Show composite preview once loaded
            } else {
                alert('Failed to regenerate preview: ' + data.error);
            }
        } catch (err) {
            console.error('Error generating preview:', err);
            alert('Failed to generate preview. Server error.');
        } finally {
            btnRefreshPreview.disabled = false;
            btnRefreshPreview.innerHTML = '<span class="btn-icon">🔄</span> Refresh Preview';
        }
    });
    
    btnSaveLayout.addEventListener('click', async () => {
        try {
            btnSaveLayout.disabled = true;
            statusBadge.textContent = 'Saving...';
            statusBadge.classList.add('saving');
            
            const response = await fetch('/api/layout/save', {
                method: 'POST'
            });
            const data = await response.json();
            
            if (data.status === 'success') {
                showToast('Layout settings saved!');
            } else {
                alert('Failed to save layout: ' + data.error);
            }
        } catch (err) {
            console.error('Error saving layout:', err);
            alert('Failed to save. Server error.');
        } finally {
            btnSaveLayout.disabled = false;
            statusBadge.textContent = 'Connected';
            statusBadge.classList.remove('saving');
        }
    });
    
    // Zoom control click listeners
    btnZoomIn.addEventListener('click', () => {
        state.zoom = Math.min(2.0, state.zoom + 0.05);
        applyZoom();
    });
    
    btnZoomOut.addEventListener('click', () => {
        state.zoom = Math.max(0.1, state.zoom - 0.05);
        applyZoom();
    });
    
    btnZoomFit.addEventListener('click', fitCanvasToViewport);
    
    // Undo / Redo click listeners
    if (btnUndo) {
        btnUndo.addEventListener('click', undo);
    }
    if (btnRedo) {
        btnRedo.addEventListener('click', redo);
    }
    
    // Slider interaction mousedown/keydown history pushes
    [cropX, cropY, cropW, cropH].forEach(input => {
        input.addEventListener('focus', pushStateToHistory);
        input.addEventListener('mousedown', pushStateToHistory);
        input.addEventListener('touchstart', pushStateToHistory);
    });
    
    // Keyboard Ctrl+Z and Ctrl+Y/Ctrl+Shift+Z Undo/Redo listener
    window.addEventListener('keydown', (e) => {
        if (document.activeElement.tagName === 'INPUT' && document.activeElement.type !== 'range') return;
        
        if (e.ctrlKey && e.key.toLowerCase() === 'z') {
            e.preventDefault();
            undo();
        } else if ((e.ctrlKey && e.key.toLowerCase() === 'y') || (e.ctrlKey && e.shiftKey && e.key.toLowerCase() === 'z')) {
            e.preventDefault();
            redo();
        }
    });
    
    // Window resize
    window.addEventListener('resize', () => {
        // If zoom-fit or scaled down, it's nice to adapt
        fitCanvasToViewport();
    });
}

// API Persistence calls
async function saveLayerPosition(layer) {
    try {
        await fetch('/api/layout/position', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                name: layer.name,
                x: layer.x,
                y: layer.y,
                visible: layer.visible,
                z_index: layer.z_index,
                crop_x_percent: layer.crop_x_percent || 0,
                crop_y_percent: layer.crop_y_percent || 0,
                crop_width_percent: layer.crop_width_percent || 0,
                crop_height_percent: layer.crop_height_percent || 0
            })
        });
    } catch (err) {
        console.error('Failed to update layer position API:', err);
    }
}

async function saveLayerSize(layer) {
    try {
        await fetch('/api/layout/size', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                name: layer.name,
                width: layer.width,
                height: layer.height,
                scale_mode: layer.scale_mode || 'stretch'
            })
        });
    } catch (err) {
        console.error('Failed to update layer size API:', err);
    }
}

async function saveLayerVisibilityAndZ(layer) {
    try {
        // We can use the position endpoint to sync visibility and zindex since we send all fields
        await saveLayerPosition(layer);
    } catch (err) {
        console.error('Failed to update layer visibility/Z:', err);
    }
}

// Display toast message
function showToast(message) {
    toast.textContent = message;
    toast.classList.add('show');
    setTimeout(() => {
        toast.classList.remove('show');
    }, 3000);
}
