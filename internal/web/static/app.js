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
    redoStack: [],
    currentTime: 0.0,
    videoDuration: 0.0,
    
    // Audio / Cuts timeline additions
    timelineZoom: 1.0,
    audioData: null,
    cuts: [],
    activeCutDrag: null
};

// Subtitle State
let subtitleState = {
    srtPath: '',
    style: {
        font_name: 'Montserrat',
        font_size: 32,
        primary_color: '#FFFF00',
        outline_color: '#000000',
        margin_v: 50,
        alignment: 2
    },
    subtitles: [],
    selectedSubIndex: -1
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

// Subtitles DOM elements
const subFont = document.getElementById('sub-font');
const subSize = document.getElementById('sub-size');
const subSizeVal = document.getElementById('sub-size-val');
const subPrimaryColor = document.getElementById('sub-primary-color');
const subPrimaryColorText = document.getElementById('sub-primary-color-text');
const subOutlineColor = document.getElementById('sub-outline-color');
const subOutlineColorText = document.getElementById('sub-outline-color-text');
const subMarginV = document.getElementById('sub-margin-v');
const subMarginVVal = document.getElementById('sub-margin-v-val');
const subtitlesContainer = document.getElementById('subtitles-container');
const canvasSubtitlePreview = document.getElementById('canvas-subtitle-preview');
const captionCount = document.getElementById('caption-count');
const btnAddCaption = document.getElementById('btn-add-caption');

// Tabs DOM elements
const tabBtnStyle = document.getElementById('tab-btn-style');
const tabBtnList = document.getElementById('tab-btn-list');
const tabContentStyle = document.getElementById('tab-content-style');
const tabContentList = document.getElementById('tab-content-list');

// Timeline DOM Elements
const timelineTrack = document.getElementById('timeline-track');
const timelineThumbnails = document.getElementById('timeline-thumbnails');
const timelinePlayhead = document.getElementById('timeline-playhead');
const timelineInfo = document.getElementById('timeline-info');

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
const scaleInput = document.getElementById('layer-scale');
const scaleVal = document.getElementById('layer-scale-val');
const cropLeft = document.getElementById('crop-left');
const cropLeftVal = document.getElementById('crop-left-val');
const cropRight = document.getElementById('crop-right');
const cropRightVal = document.getElementById('crop-right-val');
const cropTop = document.getElementById('crop-top');
const cropTopVal = document.getElementById('crop-top-val');
const cropBottom = document.getElementById('crop-bottom');
const cropBottomVal = document.getElementById('crop-bottom-val');

// Buttons
const btnRefreshPreview = document.getElementById('btn-refresh-preview');
const btnSaveLayout = document.getElementById('btn-save-layout');
const btnExportLayout = document.getElementById('btn-export-layout');
const btnZoomIn = document.getElementById('btn-zoom-in');
const btnZoomOut = document.getElementById('btn-zoom-out');
const btnZoomFit = document.getElementById('btn-zoom-fit');

// Initialize editor on load
window.addEventListener('DOMContentLoaded', async () => {
    setupEventListeners();
    setupSubtitleTabs();
    setupAlignmentGrid();
    setupSubtitleStyleListeners();
    if (btnAddCaption) {
        btnAddCaption.addEventListener('click', addNewSubtitleItem);
    }

    // Bind front page elements
    const btnCreateProject = document.getElementById('btn-create-project');
    if (btnCreateProject) {
        btnCreateProject.addEventListener('click', handleCreateProject);
    }
    const btnRenderVideo = document.getElementById('btn-render-video');
    if (btnRenderVideo) {
        btnRenderVideo.addEventListener('click', handleRenderVideo);
    }
    const selectPresetLayout = document.getElementById('select-preset-layout');
    if (selectPresetLayout) {
        selectPresetLayout.addEventListener('change', handleLoadPresetLayout);
    }
    const btnNewVideo = document.getElementById('btn-new-video');
    if (btnNewVideo) {
        btnNewVideo.addEventListener('click', handleNewVideo);
    }

    // Bind view mode switcher buttons
    document.querySelectorAll('.view-mode-btn').forEach(btn => {
        btn.addEventListener('click', () => switchViewMode(btn.dataset.mode));
    });
    setupVideoPlayer();
    setupSilenceDetectionControls();

    // Check project status on load
    try {
        const response = await fetch('/api/project/status');
        const data = await response.json();
        if (data.active) {
            document.getElementById('front-page').style.display = 'none';
            document.getElementById('editor-container').style.display = 'flex';
            fetchLayout();
            fetchSubtitles();
            fetchTimeline();
            fetchPresetLayouts();
            refreshVideoSources();
        } else {
            document.getElementById('front-page').style.display = 'flex';
            document.getElementById('editor-container').style.display = 'none';
            // Prefill URL if available
            if (data.cli_url) {
                document.getElementById('yt-url-input').value = data.cli_url;
            }
        }
    } catch (err) {
        console.error('Error checking project status:', err);
        document.getElementById('front-page').style.display = 'flex';
        document.getElementById('editor-container').style.display = 'none';
    }
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

            // Calculate Left, Right, Top, Bottom crop percents from X, Y, Width, Height crop percents
            const left = layer.crop_x_percent || 0.0;
            const top = layer.crop_y_percent || 0.0;
            const right = Math.max(0.0, 1.0 - left - (layer.crop_width_percent || 1.0));
            const bottom = Math.max(0.0, 1.0 - top - (layer.crop_height_percent || 1.0));

            layer.crop_left = left;
            layer.crop_right = right;
            layer.crop_top = top;
            layer.crop_bottom = bottom;

            // Get original input dimensions from API
            layer.input_width = layer.input_width || 1920;
            layer.input_height = layer.input_height || 1080;

            // Calculate scale factor: display width / (input_width * crop_width_percent)
            const cwPct = 1.0 - left - right;
            if (cwPct > 0.001) {
                layer.scale = layer.width / (layer.input_width * cwPct);
            } else {
                layer.scale = 1.0;
            }

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

        // Locked icon
        const lockBtn = document.createElement('button');
        lockBtn.className = `layer-lock-toggle ${layer.locked ? 'locked' : ''}`;
        lockBtn.innerHTML = layer.locked ? '🔒' : '🔓';
        lockBtn.title = layer.locked ? 'Unlock Layer' : 'Lock Layer';
        lockBtn.addEventListener('click', (e) => {
            e.stopPropagation();
            layer.locked = !layer.locked;
            lockBtn.innerHTML = layer.locked ? '🔒' : '🔓';
            lockBtn.classList.toggle('locked', layer.locked);
            if (state.selectedLayerName === layer.name) {
                const chk = document.getElementById('layer-locked-chk');
                if (chk) chk.checked = layer.locked;
            }
            saveLayerPosition(layer);
        });

        const details = document.createElement('div');
        details.className = 'layer-details';

        const titleRow = document.createElement('div');
        titleRow.style.display = 'flex';
        titleRow.style.justifyContent = 'space-between';

        const title = document.createElement('span');
        title.className = 'layer-title-text';
        title.textContent = layer.label || getLayerDisplayName(layer.name);

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
        item.appendChild(lockBtn);
        item.appendChild(details);

        item.addEventListener('click', () => {
            selectLayer(layer.name);
        });

        layerList.appendChild(item);
    });
}

// Translate system names to friendly display names
function getLayerDisplayName(name) {
    if (name && name.startsWith('extra_')) {
        const num = name.substring(6);
        return `Extra Layer ${num}`;
    }
    switch (name) {
        case 'bg': return 'Background';
        case 'game': return 'Gameplay';
        case 'cam': return 'Facecam';
        case 'overlay': return 'Overlay';
        default: return name ? (name.charAt(0).toUpperCase() + name.slice(1)) : '';
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
        selectedLayerNameDisplay.textContent = layer.label || getLayerDisplayName(layer.name);
        const customNameInput = document.getElementById('layer-custom-name');
        if (customNameInput) {
            customNameInput.value = layer.label || getLayerDisplayName(layer.name);
        }
        const lockedChk = document.getElementById('layer-locked-chk');
        if (lockedChk) {
            lockedChk.checked = !!layer.locked;
        }
        inputX.value = layer.x;
        inputY.value = layer.y;
        inputW.value = layer.width;
        inputH.value = layer.height;
        inputZ.value = layer.z_index;

        // Populate Aspect Ratio / Scale Mode and Crop Sliders
        inputScaleMode.value = layer.scale_mode || 'stretch';
        inputLockAR.checked = !!layer.lock_ar;

        scaleInput.value = Math.round((layer.scale || 1.0) * 100);
        scaleVal.textContent = `${scaleInput.value}%`;

        cropLeft.value = Math.round((layer.crop_left || 0.0) * 100);
        cropRight.value = Math.round((layer.crop_right || 0.0) * 100);
        cropTop.value = Math.round((layer.crop_top || 0.0) * 100);
        cropBottom.value = Math.round((layer.crop_bottom || 0.0) * 100);

        cropLeftVal.textContent = `${cropLeft.value}%`;
        cropRightVal.textContent = `${cropRight.value}%`;
        cropTopVal.textContent = `${cropTop.value}%`;
        cropBottomVal.textContent = `${cropBottom.value}%`;

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
        label.textContent = layer.label || getLayerDisplayName(layer.name);
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

    updateCanvasSubtitlePreview();
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
        if (layer.locked) return;

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

        if (state.isResizing) {
            const cwPct = 1.0 - (layer.crop_left || 0.0) - (layer.crop_right || 0.0);
            if (cwPct > 0.001) {
                layer.scale = layer.width / (layer.input_width * cwPct);
            } else {
                layer.scale = 1.0;
            }
            scaleInput.value = Math.round(layer.scale * 100);
            scaleVal.textContent = `${scaleInput.value}%`;
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

        // Update scale factor from new width
        const cwPct = 1.0 - (layer.crop_left || 0.0) - (layer.crop_right || 0.0);
        if (cwPct > 0.001) {
            layer.scale = layer.width / (layer.input_width * cwPct);
        } else {
            layer.scale = 1.0;
        }
        scaleInput.value = Math.round(layer.scale * 100);
        scaleVal.textContent = `${scaleInput.value}%`;

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

    // Crop & Scale sliders updates
    const triggerCropUpdate = (e) => {
        const layer = state.layers.find(l => l.name === state.selectedLayerName);
        if (!layer) return;

        let left = parseFloat(cropLeft.value) || 0;
        let right = parseFloat(cropRight.value) || 0;
        let top = parseFloat(cropTop.value) || 0;
        let bottom = parseFloat(cropBottom.value) || 0;
        let scale = parseFloat(scaleInput.value) || 100;

        const target = e ? e.target : null;

        // Enforce boundary constraints: Left + Right <= 99% and Top + Bottom <= 99%
        if (target === cropLeft) {
            if (left + right > 99) {
                left = 99 - right;
                cropLeft.value = left;
            }
        } else if (target === cropRight) {
            if (left + right > 99) {
                right = 99 - left;
                cropRight.value = right;
            }
        } else if (target === cropTop) {
            if (top + bottom > 99) {
                top = 99 - bottom;
                cropTop.value = top;
            }
        } else if (target === cropBottom) {
            if (top + bottom > 99) {
                bottom = 99 - top;
                cropBottom.value = bottom;
            }
        }

        layer.crop_left = left / 100;
        layer.crop_right = right / 100;
        layer.crop_top = top / 100;
        layer.crop_bottom = bottom / 100;
        layer.scale = scale / 100;

        // Convert Left/Right/Top/Bottom to original X/Y/Width/Height crop percentages for backend compatibility
        layer.crop_x_percent = layer.crop_left;
        layer.crop_y_percent = layer.crop_top;
        layer.crop_width_percent = Math.max(0.01, 1.0 - layer.crop_left - layer.crop_right);
        layer.crop_height_percent = Math.max(0.01, 1.0 - layer.crop_top - layer.crop_bottom);

        // Automatically adjust the layer's displayed size based on scale and crop
        layer.width = Math.round(layer.input_width * layer.crop_width_percent * layer.scale);
        layer.height = Math.round(layer.input_height * layer.crop_height_percent * layer.scale);

        cropLeftVal.textContent = `${left}%`;
        cropRightVal.textContent = `${right}%`;
        cropTopVal.textContent = `${top}%`;
        cropBottomVal.textContent = `${bottom}%`;
        scaleVal.textContent = `${scale}%`;

        // Update Width/Height inputs in transform panel
        inputW.value = layer.width;
        inputH.value = layer.height;

        const frame = document.querySelector(`.layer-frame[data-name="${layer.name}"]`);
        if (frame) {
            updateFramePosition(frame, layer);
        }

        renderLayerList();
        saveLayerPosition(layer);
        saveLayerSize(layer);
        setPreviewMode(false); // Switch to real-time interactive mode
    };

    [cropLeft, cropRight, cropTop, cropBottom, scaleInput].forEach(input => {
        input.addEventListener('input', triggerCropUpdate);
    });

    // Custom Name and Locked Layer Inputs
    const customNameInput = document.getElementById('layer-custom-name');
    if (customNameInput) {
        customNameInput.addEventListener('input', (e) => {
            const layer = state.layers.find(l => l.name === state.selectedLayerName);
            if (!layer) return;
            layer.label = e.target.value;
            const frame = document.querySelector(`.layer-frame[data-name="${layer.name}"]`);
            if (frame) {
                const labelDiv = frame.querySelector('.layer-label');
                if (labelDiv) labelDiv.textContent = layer.label;
            }
            renderLayerList();
            saveLayerPosition(layer);
        });
    }

    const lockedChk = document.getElementById('layer-locked-chk');
    if (lockedChk) {
        lockedChk.addEventListener('change', (e) => {
            const layer = state.layers.find(l => l.name === state.selectedLayerName);
            if (!layer) return;
            layer.locked = e.target.checked;
            renderLayerList();
            saveLayerPosition(layer);
        });
    }

    // Header View Switch Buttons
    const btnViewLayers = document.getElementById('btn-view-layers');
    const btnViewAudio = document.getElementById('btn-view-audio');
    if (btnViewLayers) {
        btnViewLayers.addEventListener('click', () => switchEditorView('layers'));
    }
    if (btnViewAudio) {
        btnViewAudio.addEventListener('click', () => switchEditorView('audio'));
    }

    // Keyboard arrow keys nudge
    window.addEventListener('keydown', (e) => {
        if (!state.selectedLayerName) return;

        // Skip if typing in an input
        if (document.activeElement.tagName === 'INPUT') return;

        const layer = state.layers.find(l => l.name === state.selectedLayerName);
        if (!layer) return;
        if (layer.locked) return;

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
    btnRefreshPreview.addEventListener('click', () => {
        triggerPreviewRefresh(false);
    });

    const btnAddLayer = document.getElementById('btn-add-layer');
    if (btnAddLayer) {
        btnAddLayer.addEventListener('click', async () => {
            try {
                btnAddLayer.disabled = true;
                const response = await fetch('/api/layer/add', {
                    method: 'POST'
                });
                if (!response.ok) {
                    throw new Error('Server returned ' + response.status);
                }
                const newLayer = await response.json();
                
                // Map fields to match UI state layer structure
                newLayer.lock_ar = false;
                newLayer.crop_left = newLayer.crop_x_percent || 0.0;
                newLayer.crop_top = newLayer.crop_y_percent || 0.0;
                newLayer.crop_right = Math.max(0.0, 1.0 - newLayer.crop_left - (newLayer.crop_width_percent || 1.0));
                newLayer.crop_bottom = Math.max(0.0, 1.0 - newLayer.crop_top - (newLayer.crop_height_percent || 1.0));
                newLayer.input_width = newLayer.input_width || 1920;
                newLayer.input_height = newLayer.input_height || 1080;
                
                const cwPct = 1.0 - newLayer.crop_left - newLayer.crop_right;
                if (cwPct > 0.001) {
                    newLayer.scale = newLayer.width / (newLayer.input_width * cwPct);
                } else {
                    newLayer.scale = 1.0;
                }

                pushStateToHistory();
                state.layers.push(newLayer);
                
                renderLayerList();
                renderLayerFrames();
                selectLayer(newLayer.name);
                
                showToast(`Added ${getLayerDisplayName(newLayer.name)}`);
                await triggerPreviewRefresh(true);
            } catch (err) {
                console.error('Failed to add layer:', err);
                alert('Failed to add layer: ' + err.message);
            } finally {
                btnAddLayer.disabled = false;
            }
        });
    }

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

    btnExportLayout.addEventListener('click', async () => {
        const name = prompt("Enter layout name to export (e.g. splitscreen):");
        if (!name) return;

        try {
            btnExportLayout.disabled = true;
            statusBadge.textContent = 'Exporting...';
            statusBadge.classList.add('saving');

            const response = await fetch('/api/layout/export', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ name: name })
            });
            const data = await response.json();

            if (data.status === 'success') {
                showToast(`Layout exported as ${name}.json!`);
            } else {
                alert('Failed to export layout: ' + data.error);
            }
        } catch (err) {
            console.error('Error exporting layout:', err);
            alert('Failed to export. Server error.');
        } finally {
            btnExportLayout.disabled = false;
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
    [cropLeft, cropRight, cropTop, cropBottom, scaleInput].forEach(input => {
        input.addEventListener('focus', pushStateToHistory);
        input.addEventListener('mousedown', pushStateToHistory);
        input.addEventListener('touchstart', pushStateToHistory);
    });

    // Keyboard Ctrl+Z, Ctrl+Y Undo/Redo, and 'S' Split listeners
    window.addEventListener('keydown', (e) => {
        if (document.activeElement.tagName === 'INPUT' && document.activeElement.type !== 'range') return;

        if (e.ctrlKey && e.key.toLowerCase() === 'z') {
            e.preventDefault();
            undo();
        } else if ((e.ctrlKey && e.key.toLowerCase() === 'y') || (e.ctrlKey && e.shiftKey && e.key.toLowerCase() === 'z')) {
            e.preventDefault();
            redo();
        } else if (e.key.toLowerCase() === 's' && !e.ctrlKey && !e.metaKey && !e.altKey) {
            e.preventDefault();
            splitSegmentAtPlayhead();
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
                crop_height_percent: layer.crop_height_percent || 0,
                label: layer.label || '',
                locked: !!layer.locked
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

// Subtitles Features
function setupSubtitleTabs() {
    if (!tabBtnStyle || !tabBtnList) return;
    tabBtnStyle.addEventListener('click', () => {
        tabBtnStyle.classList.add('active');
        tabBtnList.classList.remove('active');
        tabContentStyle.classList.add('active');
        tabContentStyle.style.display = 'flex';
        tabContentList.classList.remove('active');
        tabContentList.style.display = 'none';
    });
    tabBtnList.addEventListener('click', () => {
        tabBtnList.classList.add('active');
        tabBtnStyle.classList.remove('active');
        tabContentList.classList.add('active');
        tabContentList.style.display = 'flex';
        tabContentStyle.classList.remove('active');
        tabContentStyle.style.display = 'none';
    });
}

function setupAlignmentGrid() {
    document.querySelectorAll('.alignment-grid .align-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            document.querySelectorAll('.alignment-grid .align-btn').forEach(b => b.classList.remove('active'));
            btn.classList.add('active');
            subtitleState.style.alignment = parseInt(btn.dataset.align);
            updateCanvasSubtitlePreview();
            saveSubtitleStyleDebounced();
        });
    });
}

function updateAlignmentGridUI(alignValue) {
    document.querySelectorAll('.alignment-grid .align-btn').forEach(btn => {
        if (parseInt(btn.dataset.align) === alignValue) {
            btn.classList.add('active');
        } else {
            btn.classList.remove('active');
        }
    });
}

function updateCanvasSubtitlePreview() {
    if (!canvasSubtitlePreview) return;

    // Set text content to active subtitle
    if (subtitleState.subtitles.length > 0 && subtitleState.selectedSubIndex >= 0 && subtitleState.selectedSubIndex < subtitleState.subtitles.length) {
        canvasSubtitlePreview.textContent = subtitleState.subtitles[subtitleState.selectedSubIndex].text;
        canvasSubtitlePreview.style.display = 'block';
    } else if (subtitleState.subtitles.length > 0) {
        canvasSubtitlePreview.textContent = subtitleState.subtitles[0].text;
        canvasSubtitlePreview.style.display = 'block';
    } else {
        canvasSubtitlePreview.textContent = 'Subtitle Preview';
        canvasSubtitlePreview.style.display = 'block';
    }

    const style = subtitleState.style;
    const fontName = style.font_name || 'Montserrat';
    const fontSize = (style.font_size || 32) * state.zoom;
    const marginV = (style.margin_v || 50) * state.zoom;
    const primaryColor = style.primary_color || '#FFFF00';
    const outlineColor = style.outline_color || '#000000';
    const alignment = style.alignment || 2;

    canvasSubtitlePreview.style.fontFamily = `"${fontName}", sans-serif`;
    canvasSubtitlePreview.style.fontSize = `${fontSize}px`;
    canvasSubtitlePreview.style.color = primaryColor;

    const outlineWidth = Math.max(1, Math.round(3 * state.zoom));
    canvasSubtitlePreview.style.webkitTextStroke = `${outlineWidth}px ${outlineColor}`;
    canvasSubtitlePreview.style.textShadow = `0px 0px 4px ${outlineColor}`;

    canvasSubtitlePreview.style.left = 'auto';
    canvasSubtitlePreview.style.right = 'auto';
    canvasSubtitlePreview.style.top = 'auto';
    canvasSubtitlePreview.style.bottom = 'auto';
    canvasSubtitlePreview.style.transform = 'none';

    const marginLR = 20 * state.zoom;

    switch (alignment) {
        case 1:
            canvasSubtitlePreview.style.left = `${marginLR}px`;
            canvasSubtitlePreview.style.bottom = `${marginV}px`;
            canvasSubtitlePreview.style.textAlign = 'left';
            break;
        case 2:
            canvasSubtitlePreview.style.left = '50%';
            canvasSubtitlePreview.style.transform = 'translateX(-50%)';
            canvasSubtitlePreview.style.bottom = `${marginV}px`;
            canvasSubtitlePreview.style.textAlign = 'center';
            break;
        case 3:
            canvasSubtitlePreview.style.right = `${marginLR}px`;
            canvasSubtitlePreview.style.bottom = `${marginV}px`;
            canvasSubtitlePreview.style.textAlign = 'right';
            break;
        case 4:
            canvasSubtitlePreview.style.left = `${marginLR}px`;
            canvasSubtitlePreview.style.top = '50%';
            canvasSubtitlePreview.style.transform = 'translateY(-50%)';
            canvasSubtitlePreview.style.textAlign = 'left';
            break;
        case 5:
            canvasSubtitlePreview.style.left = '50%';
            canvasSubtitlePreview.style.top = '50%';
            canvasSubtitlePreview.style.transform = 'translate(-50%, -50%)';
            canvasSubtitlePreview.style.textAlign = 'center';
            break;
        case 6:
            canvasSubtitlePreview.style.right = `${marginLR}px`;
            canvasSubtitlePreview.style.top = '50%';
            canvasSubtitlePreview.style.transform = 'translateY(-50%)';
            canvasSubtitlePreview.style.textAlign = 'right';
            break;
        case 7:
            canvasSubtitlePreview.style.left = `${marginLR}px`;
            canvasSubtitlePreview.style.top = `${marginV}px`;
            canvasSubtitlePreview.style.textAlign = 'left';
            break;
        case 8:
            canvasSubtitlePreview.style.left = '50%';
            canvasSubtitlePreview.style.transform = 'translateX(-50%)';
            canvasSubtitlePreview.style.top = `${marginV}px`;
            canvasSubtitlePreview.style.textAlign = 'center';
            break;
        case 9:
            canvasSubtitlePreview.style.right = `${marginLR}px`;
            canvasSubtitlePreview.style.top = `${marginV}px`;
            canvasSubtitlePreview.style.textAlign = 'right';
            break;
    }
}

let subStyleTimeout;
function saveSubtitleStyleDebounced() {
    clearTimeout(subStyleTimeout);
    subStyleTimeout = setTimeout(async () => {
        try {
            await fetch('/api/subtitles/style', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(subtitleState.style)
            });
        } catch (err) {
            console.error('Failed to save subtitle style:', err);
        }
    }, 500);
}

let subEditTimeout;
function saveSubtitlesDebounced() {
    clearTimeout(subEditTimeout);
    subEditTimeout = setTimeout(async () => {
        try {
            await fetch('/api/subtitles/edit', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ subtitles: subtitleState.subtitles })
            });
        } catch (err) {
            console.error('Failed to save subtitles:', err);
        }
    }, 600);
}

async function fetchSubtitles() {
    try {
        const response = await fetch('/api/subtitles');
        const data = await response.json();

        subtitleState.srtPath = data.srt_path;
        subtitleState.style = data.style || {
            font_name: 'Montserrat',
            font_size: 32,
            primary_color: '#FFFF00',
            outline_color: '#000000',
            margin_v: 50,
            alignment: 2
        };
        subtitleState.subtitles = data.subtitles || [];

        if (subFont) subFont.value = subtitleState.style.font_name;
        if (subSize) {
            subSize.value = subtitleState.style.font_size;
            subSizeVal.textContent = `${subtitleState.style.font_size}px`;
        }
        if (subPrimaryColor) {
            subPrimaryColor.value = subtitleState.style.primary_color;
            subPrimaryColorText.value = subtitleState.style.primary_color;
        }
        if (subOutlineColor) {
            subOutlineColor.value = subtitleState.style.outline_color;
            subOutlineColorText.value = subtitleState.style.outline_color;
        }
        if (subMarginV) {
            subMarginV.value = subtitleState.style.margin_v;
            subMarginVVal.textContent = `${subtitleState.style.margin_v}px`;
        }

        updateAlignmentGridUI(subtitleState.style.alignment);
        renderSubtitleList();
        updateCanvasSubtitlePreview();
    } catch (err) {
        console.error('Error fetching subtitles:', err);
    }
}

function renderSubtitleList() {
    if (!subtitlesContainer) return;
    subtitlesContainer.innerHTML = '';

    captionCount.textContent = `${subtitleState.subtitles.length} captions`;

    subtitleState.subtitles.forEach((sub, index) => {
        const card = document.createElement('div');
        card.className = `subtitle-card ${subtitleState.selectedSubIndex === index ? 'active' : ''}`;
        card.dataset.index = index;

        card.addEventListener('mousedown', () => {
            selectSubtitleCard(index);
        });

        const header = document.createElement('div');
        header.className = 'subtitle-card-header';

        const idxLabel = document.createElement('span');
        idxLabel.className = 'subtitle-index';
        idxLabel.textContent = `INDEX #${sub.index}`;

        const timeDiv = document.createElement('div');
        timeDiv.className = 'subtitle-time-inputs';

        const startInput = document.createElement('input');
        startInput.type = 'number';
        startInput.step = '0.05';
        startInput.min = '0';
        startInput.value = sub.start.toFixed(2);
        startInput.addEventListener('change', (e) => {
            const val = parseFloat(e.target.value) || 0;
            sub.start = val;
            saveSubtitlesDebounced();
        });

        const sep = document.createElement('span');
        sep.className = 'subtitle-time-separator';
        sep.textContent = '→';

        const endInput = document.createElement('input');
        endInput.type = 'number';
        endInput.step = '0.05';
        endInput.min = '0';
        endInput.value = sub.end.toFixed(2);
        endInput.addEventListener('change', (e) => {
            const val = parseFloat(e.target.value) || 0;
            sub.end = val;
            saveSubtitlesDebounced();
        });

        timeDiv.appendChild(startInput);
        timeDiv.appendChild(sep);
        timeDiv.appendChild(endInput);

        header.appendChild(idxLabel);
        header.appendChild(timeDiv);

        const body = document.createElement('div');
        body.className = 'subtitle-card-body';

        const txt = document.createElement('textarea');
        txt.className = 'subtitle-text-area';
        txt.value = sub.text;
        txt.addEventListener('input', (e) => {
            sub.text = e.target.value;
            updateCanvasSubtitlePreview();
            saveSubtitlesDebounced();
        });

        const delBtn = document.createElement('button');
        delBtn.className = 'btn-delete-subtitle';
        delBtn.innerHTML = '🗑️';
        delBtn.title = 'Delete Caption';
        delBtn.addEventListener('click', (e) => {
            e.stopPropagation();
            deleteSubtitleItem(index);
        });

        body.appendChild(txt);
        body.appendChild(delBtn);

        card.appendChild(header);
        card.appendChild(body);

        subtitlesContainer.appendChild(card);
    });
}

function deleteSubtitleItem(index) {
    if (confirm("Are you sure you want to delete this caption?")) {
        subtitleState.subtitles.splice(index, 1);
        subtitleState.subtitles.forEach((sub, i) => {
            sub.index = i + 1;
        });

        if (subtitleState.selectedSubIndex >= subtitleState.subtitles.length) {
            subtitleState.selectedSubIndex = subtitleState.subtitles.length - 1;
        }

        renderSubtitleList();
        updateCanvasSubtitlePreview();
        saveSubtitlesDebounced();
    }
}

function selectSubtitleCard(index) {
    subtitleState.selectedSubIndex = index;
    document.querySelectorAll('.subtitle-card').forEach((card, idx) => {
        if (idx === index) {
            card.classList.add('active');
        } else {
            card.classList.remove('active');
        }
    });
    updateCanvasSubtitlePreview();

    // Jump to the subtitle's start frame on selection
    if (index >= 0 && index < subtitleState.subtitles.length) {
        const sub = subtitleState.subtitles[index];
        state.currentTime = sub.start;
        updateTimelineUI();

        // Update canvas layer frames with the frame at seeked position
        document.querySelectorAll('.layer-frame img').forEach(img => {
            if (img.src.includes('/api/raw-frame.png')) {
                img.src = `/api/raw-frame.png?time=${state.currentTime}&t=${Date.now()}`;
            }
        });
        setPreviewMode(false);

        // Automatically refresh the preview frame at this timestamp
        triggerPreviewRefresh(true);
    }
}

function addNewSubtitleItem() {
    let newStart = 0.0;
    let newEnd = 2.0;

    if (subtitleState.subtitles.length > 0) {
        const lastSub = subtitleState.subtitles[subtitleState.subtitles.length - 1];
        newStart = lastSub.end + 0.1;
        newEnd = newStart + 2.0;
    }

    const newIndex = subtitleState.subtitles.length + 1;
    const newSub = {
        index: newIndex,
        start: newStart,
        end: newEnd,
        text: 'New subtitle text'
    };

    subtitleState.subtitles.push(newSub);
    subtitleState.selectedSubIndex = subtitleState.subtitles.length - 1;

    renderSubtitleList();
    updateCanvasSubtitlePreview();

    setTimeout(() => {
        if (subtitlesContainer) {
            subtitlesContainer.scrollTop = subtitlesContainer.scrollHeight;
        }
    }, 50);

    saveSubtitlesDebounced();
}

function setupSubtitleStyleListeners() {
    if (!subFont) return;

    subFont.addEventListener('change', () => {
        subtitleState.style.font_name = subFont.value;
        updateCanvasSubtitlePreview();
        saveSubtitleStyleDebounced();
    });

    subSize.addEventListener('input', () => {
        subtitleState.style.font_size = parseInt(subSize.value);
        subSizeVal.textContent = `${subSize.value}px`;
        updateCanvasSubtitlePreview();
        saveSubtitleStyleDebounced();
    });

    subMarginV.addEventListener('input', () => {
        subtitleState.style.margin_v = parseInt(subMarginV.value);
        subMarginVVal.textContent = `${subMarginV.value}px`;
        updateCanvasSubtitlePreview();
        saveSubtitleStyleDebounced();
    });

    subPrimaryColor.addEventListener('input', () => {
        subPrimaryColorText.value = subPrimaryColor.value;
        subtitleState.style.primary_color = subPrimaryColor.value;
        updateCanvasSubtitlePreview();
        saveSubtitleStyleDebounced();
    });

    subPrimaryColorText.addEventListener('change', () => {
        let val = subPrimaryColorText.value.trim();
        if (!val.startsWith('#')) val = '#' + val;
        if (/^#[0-9A-F]{6}$/i.test(val)) {
            subPrimaryColor.value = val;
            subtitleState.style.primary_color = val;
            updateCanvasSubtitlePreview();
            saveSubtitleStyleDebounced();
        }
    });

    subOutlineColor.addEventListener('input', () => {
        subOutlineColorText.value = subOutlineColor.value;
        subtitleState.style.outline_color = subOutlineColor.value;
        updateCanvasSubtitlePreview();
        saveSubtitleStyleDebounced();
    });

    subOutlineColorText.addEventListener('change', () => {
        let val = subOutlineColorText.value.trim();
        if (!val.startsWith('#')) val = '#' + val;
        if (/^#[0-9A-F]{6}$/i.test(val)) {
            subOutlineColor.value = val;
            subtitleState.style.outline_color = val;
            updateCanvasSubtitlePreview();
            saveSubtitleStyleDebounced();
        }
    });
}

async function triggerPreviewRefresh(silent = false) {
    try {
        if (btnRefreshPreview) {
            btnRefreshPreview.disabled = true;
            btnRefreshPreview.textContent = 'Generating...';
        }

        const response = await fetch('/api/layout/preview?time=' + state.currentTime, {
            method: 'POST'
        });
        const data = await response.json();

        if (data.status === 'success') {
            refreshPreviewSrc();
            setPreviewMode(true);
        } else {
            if (!silent) alert('Failed to regenerate preview: ' + data.error);
            else console.error('Failed to regenerate preview:', data.error);
        }
    } catch (err) {
        console.error('Error generating preview:', err);
        if (!silent) alert('Failed to generate preview. Server error.');
    } finally {
        if (btnRefreshPreview) {
            btnRefreshPreview.disabled = false;
            btnRefreshPreview.innerHTML = '<span class="btn-icon"></span> Refresh Preview';
        }
    }
}

// Timeline logic
async function fetchTimeline() {
    if (!timelineTrack) return;
    try {
        const response = await fetch('/api/timeline');
        const data = await response.json();

        state.videoDuration = data.duration;
        state.currentTime = 0.0;

        // Initialize timeline cuts as segments
        state.cuts = [{
            id: Math.random(),
            start: 0.0,
            end: state.videoDuration,
            disabled: false
        }];

        updateTimelineUI();
        renderCuts();
        renderSidebarSegments();

        // Render thumbnails
        if (timelineThumbnails) {
            timelineThumbnails.innerHTML = '';
            const bust = Date.now();
            for (let i = 0; i < data.count; i++) {
                const img = document.createElement('img');
                img.src = `/api/timeline/thumbnail?index=${i}&t=${bust}`;
                img.alt = `Thumbnail ${i}`;
                timelineThumbnails.appendChild(img);
            }
        }

        setupTimelineScrubbing();

        // Fetch audio analysis for waveform and silences
        try {
            const selectAggression = document.getElementById('select-aggression');
            if (selectAggression) {
                selectAggression.value = 'normal'; // Reset to default on video load
                const preset = aggressionPresets['normal'];
                document.getElementById('input-threshold').value = preset.threshold;
                document.getElementById('input-min-duration').value = preset.minDur;
                document.getElementById('input-padding-before').value = preset.padBefore;
                document.getElementById('input-padding-after').value = preset.padAfter;
                
                // Trigger update labels
                const labelThreshold = document.getElementById('label-threshold');
                const labelMinDur = document.getElementById('label-min-duration');
                const labelPadBefore = document.getElementById('label-padding-before');
                const labelPadAfter = document.getElementById('label-padding-after');
                if (labelThreshold) labelThreshold.textContent = `${preset.threshold} dB`;
                if (labelMinDur) labelMinDur.textContent = `${preset.minDur}ms`;
                if (labelPadBefore) labelPadBefore.textContent = `${preset.padBefore}ms`;
                if (labelPadAfter) labelPadAfter.textContent = `${preset.padAfter}ms`;
            }

            const audioResponse = await fetch('/api/audio/analysis');
            if (audioResponse.ok) {
                state.audioData = await audioResponse.json();
                
                // Calculate cuts
                recalculateAutoCuts();
                
                // Draw timeline elements
                applyTimelineZoom();
            }
        } catch (audioErr) {
            console.error('Failed to load audio analysis data:', audioErr);
        }
    } catch (err) {
        console.error('Error fetching timeline data:', err);
    }
}

function updateTimelineUI() {
    if (!timelineInfo || !timelinePlayhead) return;

    const formatTime = (seconds) => {
        const m = Math.floor(seconds / 60);
        const s = Math.floor(seconds % 60);
        const c = Math.floor((seconds - Math.floor(seconds)) * 100);
        return `${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}.${c.toString().padStart(2, '0')}`;
    };

    timelineInfo.textContent = `${formatTime(state.currentTime)} / ${formatTime(state.videoDuration)}`;

    const pct = state.videoDuration > 0 ? (state.currentTime / state.videoDuration) * 100 : 0;
    timelinePlayhead.style.left = `${pct}%`;
}

function setupTimelineScrubbing() {
    if (!timelineTrack) return;

    let isScrubbing = false;

    const handleScrub = (clientX) => {
        const rect = timelineTrack.getBoundingClientRect();
        let pct = (clientX - rect.left) / rect.width;
        pct = Math.max(0.0, Math.min(1.0, pct));

        state.currentTime = pct * state.videoDuration;
        updateTimelineUI();

        // Update canvas overlay reference frames
        document.querySelectorAll('.layer-frame img').forEach(img => {
            if (img.src.includes('/api/raw-frame.png')) {
                img.src = `/api/raw-frame.png?time=${state.currentTime}&t=${Date.now()}`;
            }
        });

        setPreviewMode(false);
    };

    timelineTrack.addEventListener('mousedown', (e) => {
        if (e.target.closest('.timeline-cut')) {
            return;
        }
        isScrubbing = true;
        handleScrub(e.clientX);
    });

    window.addEventListener('mousemove', (e) => {
        if (state.activeCutDrag) {
            handleDraggingCutHandle(e.clientX);
            return;
        }
        if (!isScrubbing) return;
        handleScrub(e.clientX);
    });

    window.addEventListener('mouseup', () => {
        if (state.activeCutDrag) {
            stopDraggingCutHandle();
        }
        isScrubbing = false;
    });
}

async function handleCreateProject() {
    const input = document.getElementById('yt-url-input');
    const url = input.value.trim();
    if (!url) {
        alert('Please paste a YouTube URL.');
        return;
    }

    const chkSubtitles = document.getElementById('chk-render-subtitles');
    const includeSubtitles = chkSubtitles ? chkSubtitles.checked : true;

    const btn = document.getElementById('btn-create-project');
    const overlay = document.getElementById('loading-overlay');
    const status = document.getElementById('loading-status');

    try {
        btn.disabled = true;
        input.disabled = true;
        if (chkSubtitles) chkSubtitles.disabled = true;
        overlay.style.display = 'flex';
        status.textContent = 'Downloading video from YouTube (this might take a moment)...';

        const response = await fetch('/api/project/create', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                url: url,
                skip_subtitles: !includeSubtitles
            })
        });
        const data = await response.json();

        if (data.status === 'success') {
            status.textContent = 'Project initialized! Loading editor...';
            setTimeout(() => {
                overlay.style.display = 'none';
                document.getElementById('front-page').style.display = 'none';
                document.getElementById('editor-container').style.display = 'flex';

                fetchLayout();
                fetchSubtitles();
                fetchTimeline();
                fetchPresetLayouts();
                refreshVideoSources();

                btn.disabled = false;
                input.disabled = false;
                if (chkSubtitles) {
                    chkSubtitles.disabled = false;
                    chkSubtitles.checked = true;
                }
                input.value = '';
            }, 1000);
        } else {
            alert('Failed to create project: ' + data.error);
            overlay.style.display = 'none';
            btn.disabled = false;
            input.disabled = false;
            if (chkSubtitles) chkSubtitles.disabled = false;
        }
    } catch (err) {
        console.error('Error creating project:', err);
        alert('An error occurred during project setup.');
        overlay.style.display = 'none';
        btn.disabled = false;
        input.disabled = false;
        if (chkSubtitles) chkSubtitles.disabled = false;
    }
}

async function handleRenderVideo() {
    const btn = document.getElementById('btn-render-video');
    try {
        btn.disabled = true;
        btn.innerHTML = '<span class="btn-icon">⏳</span> Rendering...';
        statusBadge.textContent = 'Rendering...';
        statusBadge.classList.add('saving');
        showToast('Video rendering started in background...');

        const keeps = getKeepSegments();
        const response = await fetch('/api/project/render', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                keep_segments: keeps
            })
        });
        const data = await response.json();

        if (data.status === 'success') {
            showToast('Video rendered successfully! Saved to output folder.');
            alert('Video rendered successfully!\nSaved to: ' + data.output);
            refreshOutputVideoSource();
        } else {
            alert('Rendering failed: ' + data.error);
        }
    } catch (err) {
        console.error('Error rendering video:', err);
        alert('Rendering failed. Server error.');
    } finally {
        btn.disabled = false;
        btn.innerHTML = '<span class="btn-icon">🎬</span> Render Video';
        statusBadge.textContent = 'Connected';
        statusBadge.classList.remove('saving');
    }
}

async function handleNewVideo() {
    if (!confirm('Switch to a different video? The current project will be unloaded from the editor (your layout preset file is kept).')) {
        return;
    }

    try {
        const response = await fetch('/api/project/reset', { method: 'POST' });
        const data = await response.json();
        if (data.status !== 'success') {
            alert('Failed to reset project: ' + (data.error || 'unknown error'));
            return;
        }
    } catch (err) {
        console.error('Error resetting project:', err);
        alert('Failed to communicate with the server.');
        return;
    }

    // Clear editor state
    state.layers = [];
    state.selectedLayerName = null;
    state.undoStack = [];
    state.redoStack = [];
    state.currentTime = 0.0;
    state.videoDuration = 0.0;

    // Clear the URL input so the user can type a new one
    const urlInput = document.getElementById('yt-url-input');
    if (urlInput) urlInput.value = '';

    // Switch views
    document.getElementById('editor-container').style.display = 'none';
    document.getElementById('front-page').style.display = 'flex';
}

async function fetchPresetLayouts() {
    const selectPresetLayout = document.getElementById('select-preset-layout');
    if (!selectPresetLayout) return;

    try {
        const response = await fetch('/api/layouts');
        const layouts = await response.json();

        selectPresetLayout.innerHTML = '<option value="">Load Layout...</option>';
        layouts.forEach(name => {
            const opt = document.createElement('option');
            opt.value = name;
            let displayName = name.replace('.json', '');
            displayName = displayName.charAt(0).toUpperCase() + displayName.slice(1);
            opt.textContent = displayName;
            selectPresetLayout.appendChild(opt);
        });
    } catch (err) {
        console.error('Failed to fetch layout templates:', err);
    }
}

async function handleLoadPresetLayout(e) {
    const name = e.target.value;
    if (!name) return;

    if (!confirm(`Are you sure you want to load layout "${name}"? This will overwrite your current layer placements.`)) {
        e.target.value = '';
        return;
    }

    try {
        statusBadge.textContent = 'Loading Layout...';
        statusBadge.classList.add('saving');

        const response = await fetch('/api/layout/load', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name: name })
        });
        const data = await response.json();

        if (data.status === 'success') {
            showToast(`Applied layout: ${name}`);
            await fetchLayout();

            document.querySelectorAll('.layer-frame img').forEach(img => {
                if (img.src.includes('/api/raw-frame.png')) {
                    img.src = `/api/raw-frame.png?time=${state.currentTime}&t=${Date.now()}`;
                }
            });
            setPreviewMode(false);
        } else {
            alert('Failed to load layout template: ' + data.error);
        }
    } catch (err) {
        console.error('Error loading layout template:', err);
        alert('Failed to apply layout. Server error.');
    } finally {
        e.target.value = '';
        statusBadge.textContent = 'Connected';
        statusBadge.classList.remove('saving');
    }
}

// ============================================================
// VIDEO PLAYER MODULE
// ============================================================

let currentViewMode = 'editor'; // 'editor' | 'source' | 'output' | 'compare'
let vcIsSeeking = false;
let vcAnimFrame = null;

function switchViewMode(mode) {
    currentViewMode = mode;

    // Update active button
    document.querySelectorAll('.view-mode-btn').forEach(btn => {
        btn.classList.toggle('active', btn.dataset.mode === mode);
    });

    const canvasViewport = document.getElementById('canvas-viewport');
    const videoPanel = document.getElementById('video-panel');
    const timelineSection = document.getElementById('timeline-section');
    const vidSource = document.getElementById('vid-source');
    const vidOutput = document.getElementById('vid-output');
    const slotSource = document.getElementById('video-slot-source');
    const slotOutput = document.getElementById('video-slot-output');

    // Pause both videos when switching
    if (vidSource) vidSource.pause();
    if (vidOutput) vidOutput.pause();
    vcUpdatePlayBtn(false);

    if (mode === 'editor') {
        canvasViewport.style.display = '';
        videoPanel.style.display = 'none';
        timelineSection.style.display = '';
    } else {
        canvasViewport.style.display = 'none';
        videoPanel.style.display = 'flex';
        timelineSection.style.display = 'none';

        if (mode === 'source') {
            slotSource.style.display = '';
            slotOutput.style.display = 'none';
            vcSetPrimaryVideo(vidSource);
        } else if (mode === 'output') {
            slotSource.style.display = 'none';
            slotOutput.style.display = '';
            vcSetPrimaryVideo(vidOutput);
        } else if (mode === 'compare') {
            slotSource.style.display = '';
            slotOutput.style.display = '';
            vcSetPrimaryVideo(vidSource); // source drives controls; output syncs
        }

        // Load videos lazily
        if ((mode === 'source' || mode === 'compare') && vidSource && vidSource.readyState === 0) vidSource.load();
        if ((mode === 'output' || mode === 'compare') && vidOutput && vidOutput.readyState === 0) vidOutput.load();
    }

    cancelAnimationFrame(vcAnimFrame);
    if (mode !== 'editor') vcTick();
}

let vcPrimaryVideo = null;

function vcSetPrimaryVideo(vid) {
    vcPrimaryVideo = vid;
    const seek = document.getElementById('vc-seek');
    if (seek) seek.value = 0;
    const fill = document.getElementById('vc-progress-fill');
    if (fill) fill.style.width = '0%';
    vcUpdateTime(0, vid ? vid.duration || 0 : 0);
}

function vcTick() {
    if (!vcPrimaryVideo || vcIsSeeking) {
        vcAnimFrame = requestAnimationFrame(vcTick);
        return;
    }
    const cur = vcPrimaryVideo.currentTime;
    const dur = vcPrimaryVideo.duration || 0;
    const pct = dur > 0 ? (cur / dur) * 100 : 0;

    const seek = document.getElementById('vc-seek');
    const fill = document.getElementById('vc-progress-fill');
    if (seek) seek.value = pct * 10; // max is 1000
    if (fill) fill.style.width = pct + '%';
    vcUpdateTime(cur, dur);

    // Sync output video in compare mode
    if (currentViewMode === 'compare') {
        const vidOut = document.getElementById('vid-output');
        if (vidOut && Math.abs(vidOut.currentTime - cur) > 0.3) {
            vidOut.currentTime = cur;
        }
    }

    vcAnimFrame = requestAnimationFrame(vcTick);
}

function vcUpdateTime(cur, dur) {
    const fmt = s => {
        const m = Math.floor(s / 60);
        const sec = Math.floor(s % 60).toString().padStart(2, '0');
        return `${m}:${sec}`;
    };
    const el = document.getElementById('vc-time');
    if (el) el.textContent = `${fmt(cur)} / ${fmt(dur || 0)}`;
}

function vcUpdatePlayBtn(playing) {
    const btn = document.getElementById('vc-playpause');
    if (btn) btn.textContent = playing ? '⏸' : '▶';
}

function refreshVideoSources() {
    const vidSource = document.getElementById('vid-source');
    const vidOutput = document.getElementById('vid-output');
    const bust = Date.now();
    if (vidSource) {
        vidSource.src = `/api/video/source?t=${bust}`;
        vidSource.load();
    }
    if (vidOutput) {
        vidOutput.src = `/api/video/output?t=${bust}`;
        vidOutput.load();
    }
}

function refreshOutputVideoSource() {
    const vidOutput = document.getElementById('vid-output');
    if (vidOutput) {
        vidOutput.src = `/api/video/output?t=${Date.now()}`;
        vidOutput.load();
    }
}

function setupVideoPlayer() {
    const vidSource = document.getElementById('vid-source');
    const vidOutput = document.getElementById('vid-output');
    const btnPlay = document.getElementById('vc-playpause');
    const btnRestart = document.getElementById('vc-restart');
    const seekInput = document.getElementById('vc-seek');
    const muteBtn = document.getElementById('vc-mute');
    const volInput = document.getElementById('vc-volume');
    const fsBtn = document.getElementById('vc-fullscreen');

    if (!vidSource) return;

    // Set initial volume
    [vidSource, vidOutput].forEach(v => { if (v) v.volume = 0.8; });

    // Play / Pause
    btnPlay && btnPlay.addEventListener('click', () => {
        if (!vcPrimaryVideo) return;
        if (vcPrimaryVideo.paused) {
            vcPrimaryVideo.play();
            if (currentViewMode === 'compare' && vidOutput) vidOutput.play();
            vcUpdatePlayBtn(true);
        } else {
            vcPrimaryVideo.pause();
            if (currentViewMode === 'compare' && vidOutput) vidOutput.pause();
            vcUpdatePlayBtn(false);
        }
    });

    // Restart
    btnRestart && btnRestart.addEventListener('click', () => {
        if (!vcPrimaryVideo) return;
        vcPrimaryVideo.currentTime = 0;
        if (currentViewMode === 'compare' && vidOutput) vidOutput.currentTime = 0;
    });

    // Seek — mousedown locks tick updates
    seekInput && seekInput.addEventListener('mousedown', () => { vcIsSeeking = true; });
    seekInput && seekInput.addEventListener('input', () => {
        if (!vcPrimaryVideo || !vcPrimaryVideo.duration) return;
        const pct = seekInput.value / 1000;
        const fill = document.getElementById('vc-progress-fill');
        if (fill) fill.style.width = (pct * 100) + '%';
        vcUpdateTime(vcPrimaryVideo.duration * pct, vcPrimaryVideo.duration);
    });
    seekInput && seekInput.addEventListener('change', () => {
        if (!vcPrimaryVideo || !vcPrimaryVideo.duration) return;
        const t = (seekInput.value / 1000) * vcPrimaryVideo.duration;
        vcPrimaryVideo.currentTime = t;
        if (currentViewMode === 'compare' && vidOutput) vidOutput.currentTime = t;
        vcIsSeeking = false;
    });

    // Mute toggle
    muteBtn && muteBtn.addEventListener('click', () => {
        const muted = !vcPrimaryVideo.muted;
        [vidSource, vidOutput].forEach(v => { if (v) v.muted = muted; });
        muteBtn.textContent = muted ? '🔇' : '🔊';
    });

    // Volume slider
    volInput && volInput.addEventListener('input', () => {
        const vol = volInput.value / 100;
        [vidSource, vidOutput].forEach(v => { if (v) v.volume = vol; });
    });

    // Fullscreen on the whole video panel
    fsBtn && fsBtn.addEventListener('click', () => {
        const panel = document.getElementById('video-panel');
        if (!panel) return;
        if (document.fullscreenElement) {
            document.exitFullscreen();
        } else {
            panel.requestFullscreen();
        }
    });

    // Ended events
    [vidSource, vidOutput].forEach(v => {
        if (!v) return;
        v.addEventListener('ended', () => vcUpdatePlayBtn(false));
    });
}

// Silence Detection & Waveform Timeline Logic

const aggressionPresets = {
    'very-conservative': { threshold: -45, minDur: 600, padBefore: 250, padAfter: 300 },
    'conservative': { threshold: -40, minDur: 450, padBefore: 200, padAfter: 250 },
    'normal': { threshold: -35, minDur: 300, padBefore: 100, padAfter: 150 },
    'aggressive': { threshold: -30, minDur: 200, padBefore: 50, padAfter: 100 },
    'very-aggressive': { threshold: -25, minDur: 150, padBefore: 0, padAfter: 50 }
};

function detectSilences(dbPoints, thresholdDb, minSilenceDurationMs, paddingBeforeMs, paddingAfterMs) {
    const pointsCount = dbPoints.length;
    if (pointsCount === 0) return [];

    const intervalMs = 50; // 50ms per point
    const minSilencePoints = Math.ceil(minSilenceDurationMs / intervalMs);
    const paddingBeforePoints = Math.round(paddingBeforeMs / intervalMs);
    const paddingAfterPoints = Math.round(paddingAfterMs / intervalMs);

    const cuts = [];
    let silenceStartIdx = -1;

    for (let i = 0; i < pointsCount; i++) {
        const isSilentPoint = dbPoints[i] <= thresholdDb;

        if (isSilentPoint) {
            if (silenceStartIdx === -1) {
                silenceStartIdx = i;
            }
        } else {
            if (silenceStartIdx !== -1) {
                const durationPoints = i - silenceStartIdx;
                if (durationPoints >= minSilencePoints) {
                    const cutStartIdx = silenceStartIdx + paddingAfterPoints;
                    const cutEndIdx = i - paddingBeforePoints;

                    if (cutStartIdx < cutEndIdx) {
                        cuts.push({
                            id: Math.random(),
                            start: cutStartIdx * (intervalMs / 1000.0),
                            end: cutEndIdx * (intervalMs / 1000.0),
                            manual: false
                        });
                    }
                }
                silenceStartIdx = -1;
            }
        }
    }

    if (silenceStartIdx !== -1) {
        const durationPoints = pointsCount - silenceStartIdx;
        if (durationPoints >= minSilencePoints) {
            const cutStartIdx = silenceStartIdx + paddingAfterPoints;
            const cutEndIdx = pointsCount - 1;
            if (cutStartIdx < cutEndIdx) {
                cuts.push({
                    id: Math.random(),
                    start: cutStartIdx * (intervalMs / 1000.0),
                    end: cutEndIdx * (intervalMs / 1000.0),
                    manual: false
                });
            }
        }
    }

    return cuts;
}

function recalculateAutoCuts() {
    if (!state.audioData) return;

    const threshold = parseFloat(document.getElementById('input-threshold').value);
    const minDur = parseInt(document.getElementById('input-min-duration').value);
    const padBefore = parseInt(document.getElementById('input-padding-before').value);
    const padAfter = parseInt(document.getElementById('input-padding-after').value);

    // Auto-detect cuts using db readings
    const silenceCuts = detectSilences(state.audioData.db, threshold, minDur, padBefore, padAfter);

    // Sort and merge the silent regions
    const mergedSilences = sortAndMergeCuts(silenceCuts);

    // Convert silent regions into kept segments (the inverse)
    const duration = state.videoDuration || 0.0;
    const keptSegments = [];
    let lastEnd = 0.0;

    for (const silence of mergedSilences) {
        if (silence.start > lastEnd + 0.01) {
            keptSegments.push({
                id: Math.random(),
                start: lastEnd,
                end: silence.start,
                disabled: false
            });
        }
        lastEnd = Math.max(lastEnd, silence.end);
    }

    if (duration > lastEnd + 0.01) {
        keptSegments.push({
            id: Math.random(),
            start: lastEnd,
            end: duration,
            disabled: false
        });
    }

    state.cuts = keptSegments;
    renderCuts();
    renderSidebarSegments();
}

function sortAndMergeCuts(cutsArray) {
    if (!cutsArray || cutsArray.length === 0) return [];

    const cuts = cutsArray.map(c => ({...c}));
    cuts.sort((a, b) => a.start - b.start);

    const merged = [];
    let current = cuts[0];

    for (let i = 1; i < cuts.length; i++) {
        const next = cuts[i];

        if (next.start <= current.end + 0.05) {
            current.end = Math.max(current.end, next.end);
            if (next.manual) current.manual = true;
        } else {
            merged.push(current);
            current = next;
        }
    }
    merged.push(current);
    return merged;
}

function drawWaveform() {
    const canvas = document.getElementById('waveform-canvas');
    if (!canvas || !state.audioData) return;

    canvas.width = canvas.clientWidth;
    canvas.height = canvas.clientHeight;

    const ctx = canvas.getContext('2d');
    ctx.clearRect(0, 0, canvas.width, canvas.height);

    const peaks = state.audioData.peaks;
    const count = peaks.length;
    if (count === 0) return;

    const width = canvas.width;
    const height = canvas.height;
    const midY = height / 2;

    ctx.strokeStyle = 'rgba(255, 255, 255, 0.05)';
    ctx.lineWidth = 1.0;
    ctx.beginPath();
    ctx.moveTo(0, midY);
    ctx.lineTo(width, midY);
    ctx.stroke();

    ctx.strokeStyle = 'rgba(234, 172, 139, 0.42)'; // var(--light-bronze) at ~42% opacity
    ctx.lineWidth = 1.5;
    ctx.beginPath();

    for (let x = 0; x < width; x++) {
        const pct = x / width;
        const dataIdx = Math.floor(pct * count);
        if (dataIdx >= count) break;

        const amplitude = peaks[dataIdx] || 0.0;
        const lineH = amplitude * (height * 0.70);

        ctx.moveTo(x, midY - lineH / 2);
        ctx.lineTo(x, midY + lineH / 2);
    }
    ctx.stroke();
}

function drawRuler() {
    const ruler = document.getElementById('timeline-ruler');
    if (!ruler || !state.videoDuration) return;

    ruler.innerHTML = '';

    let interval = 5.0; 
    if (state.timelineZoom > 7.0) {
        interval = 0.5;
    } else if (state.timelineZoom > 4.5) {
        interval = 1.0;
    } else if (state.timelineZoom > 2.0) {
        interval = 2.0;
    } else if (state.timelineZoom < 0.5) {
        interval = 30.0;
    }

    const duration = state.videoDuration;
    const ticksCount = Math.floor(duration / interval);

    for (let i = 0; i <= ticksCount; i++) {
        const time = i * interval;
        const pct = (time / duration) * 100;

        const tick = document.createElement('div');
        tick.style.position = 'absolute';
        tick.style.left = `${pct}%`;
        tick.style.bottom = '0';
        tick.style.width = '1px';
        tick.style.height = '6px';
        tick.style.backgroundColor = 'rgba(255, 255, 255, 0.15)';

        if (i % 2 === 0 || interval <= 1.0) {
            const label = document.createElement('div');
            label.style.position = 'absolute';
            label.style.left = `${pct}%`;
            label.style.bottom = '8px';
            label.style.transform = 'translateX(-50%)';
            label.style.fontSize = '0.65rem';
            label.style.color = 'var(--text-muted)';
            label.style.fontFamily = 'monospace';
            
            const m = Math.floor(time / 60);
            const s = Math.floor(time % 60);
            const ms = Math.floor((time % 1) * 10);
            if (interval < 1.0) {
                label.textContent = `${m}:${s.toString().padStart(2, '0')}.${ms}`;
            } else {
                label.textContent = `${m}:${s.toString().padStart(2, '0')}`;
            }
            ruler.appendChild(label);
        }

        ruler.appendChild(tick);
    }
}

function renderCuts() {
    const container = document.getElementById('cuts-container');
    if (!container || !state.videoDuration) return;

    container.innerHTML = '';
    const duration = state.videoDuration;

    state.cuts.forEach((cut, idx) => {
        const cutDiv = document.createElement('div');
        cutDiv.className = 'timeline-cut';
        if (cut.disabled) {
            cutDiv.classList.add('disabled');
        }
        if (cut.id === state.selectedSegmentId) {
            cutDiv.classList.add('selected');
        }
        cutDiv.style.left = `${(cut.start / duration) * 100}%`;
        cutDiv.style.width = `${((cut.end - cut.start) / duration) * 100}%`;
        cutDiv.dataset.id = cut.id;

        const label = document.createElement('span');
        label.className = 'cut-label';
        label.textContent = `Clip ${idx + 1}: ${(cut.end - cut.start).toFixed(1)}s`;
        cutDiv.appendChild(label);

        // Add selection listener
        cutDiv.addEventListener('mousedown', (e) => {
            if (e.target.classList.contains('cut-handle') || e.target.classList.contains('cut-remove-btn') || e.target.classList.contains('cut-disable-btn')) {
                return;
            }
            state.selectedSegmentId = cut.id;
            renderCuts();
            renderSidebarSegments();
        });

        const leftHandle = document.createElement('div');
        leftHandle.className = 'cut-handle left-handle';
        leftHandle.addEventListener('mousedown', (e) => {
            e.stopPropagation();
            e.preventDefault();
            startDraggingCutHandle(cut.id, 'left', e.clientX);
        });
        cutDiv.appendChild(leftHandle);

        const rightHandle = document.createElement('div');
        rightHandle.className = 'cut-handle right-handle';
        rightHandle.addEventListener('mousedown', (e) => {
            e.stopPropagation();
            e.preventDefault();
            startDraggingCutHandle(cut.id, 'right', e.clientX);
        });
        cutDiv.appendChild(rightHandle);

        const muteBtn = document.createElement('button');
        muteBtn.className = 'cut-disable-btn';
        muteBtn.innerHTML = cut.disabled ? '👁️' : '∅';
        muteBtn.title = cut.disabled ? 'Enable Clip' : 'Disable Clip';
        muteBtn.addEventListener('click', (e) => {
            e.stopPropagation();
            cut.disabled = !cut.disabled;
            renderCuts();
            renderSidebarSegments();
        });
        cutDiv.appendChild(muteBtn);

        const removeBtn = document.createElement('button');
        removeBtn.className = 'cut-remove-btn';
        removeBtn.innerHTML = '&times;';
        removeBtn.title = 'Delete Clip';
        removeBtn.addEventListener('click', (e) => {
            e.stopPropagation();
            removeCut(cut.id);
            renderSidebarSegments();
        });
        cutDiv.appendChild(removeBtn);

        container.appendChild(cutDiv);
    });
}

function startDraggingCutHandle(cutId, handleType, startX) {
    const cut = state.cuts.find(c => c.id === cutId);
    if (!cut) return;

    state.activeCutDrag = {
        cutId: cutId,
        handleType: handleType,
        startX: startX,
        startVal: handleType === 'left' ? cut.start : cut.end
    };

    document.body.style.cursor = 'col-resize';
}

function handleDraggingCutHandle(clientX) {
    if (!state.activeCutDrag || !state.videoDuration) return;

    const drag = state.activeCutDrag;
    const track = document.getElementById('timeline-track');
    const trackWidth = track.clientWidth;
    
    const dx = clientX - drag.startX;
    const dt = (dx / trackWidth) * state.videoDuration;

    const cut = state.cuts.find(c => c.id === drag.cutId);
    if (!cut) return;

    const idx = state.cuts.findIndex(c => c.id === cut.id);

    if (drag.handleType === 'left') {
        const minVal = (idx > 0) ? state.cuts[idx - 1].end : 0.0;
        const maxVal = cut.end - 0.05;
        const newVal = Math.max(minVal, Math.min(maxVal, drag.startVal + dt));
        cut.start = newVal;
    } else if (drag.handleType === 'right') {
        const minVal = cut.start + 0.05;
        const maxVal = (idx < state.cuts.length - 1) ? state.cuts[idx + 1].start : state.videoDuration;
        const newVal = Math.max(minVal, Math.min(maxVal, drag.startVal + dt));
        cut.end = newVal;
    }

    renderCuts();
    renderSidebarSegments();
}

function stopDraggingCutHandle() {
    if (!state.activeCutDrag) return;

    state.activeCutDrag = null;
    document.body.style.cursor = '';

    renderCuts();
    renderSidebarSegments();
}

function splitSegmentAtPlayhead() {
    if (!state.videoDuration) return;
    const t = state.currentTime;

    // Find the segment containing t
    const segIdx = state.cuts.findIndex(seg => t > seg.start && t < seg.end);
    if (segIdx === -1) {
        showToast('Playhead is not inside any active clip.');
        return;
    }

    const seg = state.cuts[segIdx];
    
    // Split into two
    const seg1 = {
        id: Math.random(),
        start: seg.start,
        end: t,
        disabled: seg.disabled
    };
    const seg2 = {
        id: Math.random(),
        start: t,
        end: seg.end,
        disabled: seg.disabled
    };

    // Replace the segment in state.cuts
    state.cuts.splice(segIdx, 1, seg1, seg2);

    renderCuts();
    renderSidebarSegments();
    showToast('Split clip at playhead');
}

function removeCut(id) {
    state.cuts = state.cuts.filter(c => c.id !== id);
    renderCuts();
    renderSidebarSegments();
    showToast('Timeline clip deleted');
}

function getKeepSegments() {
    return state.cuts
        .filter(c => !c.disabled)
        .map(c => ({ start: c.start, end: c.end }));
}

function renderSidebarSegments() {
    const list = document.getElementById('sidebar-segments-list');
    if (!list) return;
    list.innerHTML = '';

    state.cuts.forEach((seg, idx) => {
        const item = document.createElement('div');
        item.className = `segment-item ${seg.disabled ? 'disabled' : ''} ${seg.id === state.selectedSegmentId ? 'selected' : ''}`;
        item.draggable = true;
        item.dataset.index = idx;

        // Drag and drop events
        item.addEventListener('dragstart', (e) => {
            e.dataTransfer.setData('text/plain', idx);
            item.classList.add('dragging');
        });
        item.addEventListener('dragend', () => {
            item.classList.remove('dragging');
        });
        item.addEventListener('dragover', (e) => {
            e.preventDefault();
        });
        item.addEventListener('drop', (e) => {
            e.preventDefault();
            const fromIdx = parseInt(e.dataTransfer.getData('text/plain'));
            const toIdx = idx;
            if (fromIdx !== toIdx) {
                const [moved] = state.cuts.splice(fromIdx, 1);
                state.cuts.splice(toIdx, 0, moved);
                renderCuts();
                renderSidebarSegments();
            }
        });

        // Item Header
        const header = document.createElement('div');
        header.className = 'segment-item-header';
        
        const nameSpan = document.createElement('span');
        nameSpan.textContent = `Clip ${idx + 1} (${(seg.end - seg.start).toFixed(1)}s)`;
        header.appendChild(nameSpan);

        const timeSpan = document.createElement('span');
        timeSpan.style.fontFamily = 'monospace';
        timeSpan.style.fontSize = '0.75rem';
        timeSpan.textContent = `${seg.start.toFixed(1)}s - ${seg.end.toFixed(1)}s`;
        header.appendChild(timeSpan);

        item.appendChild(header);

        // Controls Row
        const controls = document.createElement('div');
        controls.className = 'segment-item-controls';

        // Move Up
        const btnUp = document.createElement('button');
        btnUp.className = 'segment-btn';
        btnUp.textContent = '▲';
        btnUp.title = 'Move Up';
        btnUp.disabled = idx === 0;
        btnUp.addEventListener('click', (e) => {
            e.stopPropagation();
            if (idx > 0) {
                const temp = state.cuts[idx];
                state.cuts[idx] = state.cuts[idx - 1];
                state.cuts[idx - 1] = temp;
                renderCuts();
                renderSidebarSegments();
            }
        });
        controls.appendChild(btnUp);

        // Move Down
        const btnDown = document.createElement('button');
        btnDown.className = 'segment-btn';
        btnDown.textContent = '▼';
        btnDown.title = 'Move Down';
        btnDown.disabled = idx === state.cuts.length - 1;
        btnDown.addEventListener('click', (e) => {
            e.stopPropagation();
            if (idx < state.cuts.length - 1) {
                const temp = state.cuts[idx];
                state.cuts[idx] = state.cuts[idx + 1];
                state.cuts[idx + 1] = temp;
                renderCuts();
                renderSidebarSegments();
            }
        });
        controls.appendChild(btnDown);

        // Disable/Enable
        const btnMute = document.createElement('button');
        btnMute.className = `segment-btn ${seg.disabled ? 'active' : ''}`;
        btnMute.textContent = seg.disabled ? 'Enable' : 'Disable';
        btnMute.addEventListener('click', (e) => {
            e.stopPropagation();
            seg.disabled = !seg.disabled;
            renderCuts();
            renderSidebarSegments();
        });
        controls.appendChild(btnMute);

        // Delete
        const btnDel = document.createElement('button');
        btnDel.className = 'segment-btn danger';
        btnDel.textContent = 'Delete';
        btnDel.addEventListener('click', (e) => {
            e.stopPropagation();
            removeCut(seg.id);
            renderSidebarSegments();
        });
        controls.appendChild(btnDel);

        item.appendChild(controls);

        item.addEventListener('click', () => {
            state.selectedSegmentId = seg.id;
            renderCuts();
            renderSidebarSegments();
            updatePreviewFrameAtTime(seg.start);
        });

        list.appendChild(item);
    });
}

function updatePreviewFrameAtTime(t) {
    state.currentTime = t;
    updateTimelineUI();
    document.querySelectorAll('.layer-frame img').forEach(img => {
        if (img.src.includes('/api/raw-frame.png')) {
            img.src = `/api/raw-frame.png?time=${t}&t=${Date.now()}`;
        }
    });
    const vidSource = document.getElementById('vid-source');
    if (vidSource) vidSource.currentTime = t;
    const vidOutput = document.getElementById('vid-output');
    if (vidOutput) vidOutput.currentTime = t;
}

function switchEditorView(view) {
    const btnLayers = document.getElementById('btn-view-layers');
    const btnAudio = document.getElementById('btn-view-audio');
    const sidebarLayers = document.getElementById('sidebar-layers-view');
    const sidebarAudio = document.getElementById('sidebar-audio-view');
    const overlayContainer = document.getElementById('layer-overlay-container');
    const subtitlesSidebar = document.querySelector('.subtitles-sidebar');

    if (view === 'layers') {
        if (btnLayers) btnLayers.classList.add('active');
        if (btnAudio) btnAudio.classList.remove('active');
        if (sidebarLayers) sidebarLayers.style.display = 'flex';
        if (sidebarAudio) sidebarAudio.style.display = 'none';
        if (overlayContainer) overlayContainer.style.display = 'block';
        if (subtitlesSidebar) subtitlesSidebar.style.display = 'flex';
    } else if (view === 'audio') {
        if (btnLayers) btnLayers.classList.remove('active');
        if (btnAudio) btnAudio.classList.add('active');
        if (sidebarLayers) sidebarLayers.style.display = 'none';
        if (sidebarAudio) sidebarAudio.style.display = 'flex';
        if (overlayContainer) overlayContainer.style.display = 'none';
        if (subtitlesSidebar) subtitlesSidebar.style.display = 'none';
        
        renderSidebarSegments();
    }
}

function setupSilenceDetectionControls() {
    const selectAggression = document.getElementById('select-aggression');
    const inputThreshold = document.getElementById('input-threshold');
    const inputMinDur = document.getElementById('input-min-duration');
    const inputPadBefore = document.getElementById('input-padding-before');
    const inputPadAfter = document.getElementById('input-padding-after');
    const btnAddCut = document.getElementById('btn-add-cut');

    const labelThreshold = document.getElementById('label-threshold');
    const labelMinDur = document.getElementById('label-min-duration');
    const labelPadBefore = document.getElementById('label-padding-before');
    const labelPadAfter = document.getElementById('label-padding-after');

    const updateLabels = () => {
        if (labelThreshold) labelThreshold.textContent = `${inputThreshold.value} dB`;
        if (labelMinDur) labelMinDur.textContent = `${inputMinDur.value}ms`;
        if (labelPadBefore) labelPadBefore.textContent = `${inputPadBefore.value}ms`;
        if (labelPadAfter) labelPadAfter.textContent = `${inputPadAfter.value}ms`;
    };

    if (selectAggression) {
        selectAggression.addEventListener('change', () => {
            const presetName = selectAggression.value;
            if (presetName === 'custom') return;

            const preset = aggressionPresets[presetName];
            if (preset) {
                inputThreshold.value = preset.threshold;
                inputMinDur.value = preset.minDur;
                inputPadBefore.value = preset.padBefore;
                inputPadAfter.value = preset.padAfter;

                updateLabels();
                recalculateAutoCuts();
            }
        });
    }

    [inputThreshold, inputMinDur, inputPadBefore, inputPadAfter].forEach(input => {
        if (input) {
            input.addEventListener('input', () => {
                if (selectAggression) selectAggression.value = 'custom';
                updateLabels();
                recalculateAutoCuts();
            });
        }
    });

    if (btnAddCut) {
        btnAddCut.addEventListener('click', splitSegmentAtPlayhead);
    }

    const btnTimelineZoomIn = document.getElementById('btn-timeline-zoom-in');
    const btnTimelineZoomOut = document.getElementById('btn-timeline-zoom-out');

    if (btnTimelineZoomIn) {
        btnTimelineZoomIn.addEventListener('click', () => {
            state.timelineZoom = Math.min(10.0, state.timelineZoom + 0.5);
            applyTimelineZoom();
        });
    }

    if (btnTimelineZoomOut) {
        btnTimelineZoomOut.addEventListener('click', () => {
            state.timelineZoom = Math.max(1.0, state.timelineZoom - 0.5);
            applyTimelineZoom();
        });
    }
}

function applyTimelineZoom() {
    const track = document.getElementById('timeline-track');
    if (!track) return;

    track.style.width = `${state.timelineZoom * 100}%`;

    drawWaveform();
    drawRuler();
    renderCuts();
    updateTimelineUI();
}

