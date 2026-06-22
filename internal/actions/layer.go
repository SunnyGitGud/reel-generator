package actions

import (
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"sort"

	"github.com/yourname/video-editor/internal/ffmpeg"
	"github.com/yourname/video-editor/internal/preset"
)

type RenderLayer struct {
	Name string

	InputIndex int

	X int
	Y int

	Visible bool
	ZIndex  int

	ImagePath string // Path to image overlay (if applicable)

	Filters []ffmpeg.Filter
}

func (r RenderLayer) GetName() string {
	return r.Name
}

func (r RenderLayer) GetInputIndex() int {
	return r.InputIndex
}

func (r RenderLayer) GetX() int {
	return r.X
}

func (r RenderLayer) GetY() int {
	return r.Y
}

func (r RenderLayer) GetFilters() []ffmpeg.Filter {
	return r.Filters
}

func (r *RenderLayer) GetWidth() int {
	for _, f := range r.Filters {
		switch filter := f.(type) {
		case ffmpeg.Scale:
			return filter.Width
		case ffmpeg.ScaleToFill:
			return filter.Width
		}
	}
	return 0
}

func (r *RenderLayer) GetHeight() int {
	for _, f := range r.Filters {
		switch filter := f.(type) {
		case ffmpeg.Scale:
			return filter.Height
		case ffmpeg.ScaleToFill:
			return filter.Height
		}
	}
	return 0
}

func (r *RenderLayer) SetSize(w, h int) {
	for i, f := range r.Filters {
		switch filter := f.(type) {
		case ffmpeg.Scale:
			filter.Width = w
			filter.Height = h
			r.Filters[i] = filter
		case ffmpeg.ScaleToFill:
			filter.Width = w
			filter.Height = h
			r.Filters[i] = filter
		case ffmpeg.DrawBox:
			filter.Width = w
			filter.Height = h
			r.Filters[i] = filter
		}
	}
}

func (r *RenderLayer) GetScaleMode() string {
	for _, f := range r.Filters {
		if sf, ok := f.(ffmpeg.Scale); ok {
			if sf.ScaleMode == "" {
				return "stretch"
			}
			return sf.ScaleMode
		}
	}
	return "stretch"
}

func (r *RenderLayer) SetScaleMode(mode string) {
	for i, f := range r.Filters {
		if sf, ok := f.(ffmpeg.Scale); ok {
			sf.ScaleMode = mode
			r.Filters[i] = sf
		}
	}
}

func (r *RenderLayer) GetCrop() (x, y, w, h float64) {
	for _, f := range r.Filters {
		if cf, ok := f.(ffmpeg.CropPercent); ok {
			return cf.XPercent, cf.YPercent, cf.WidthPercent, cf.HeightPercent
		}
	}
	return 0, 0, 0, 0
}

func (r *RenderLayer) SetCrop(x, y, w, h float64) {
	for i, f := range r.Filters {
		if cf, ok := f.(ffmpeg.CropPercent); ok {
			cf.XPercent = x
			cf.YPercent = y
			cf.WidthPercent = w
			cf.HeightPercent = h
			r.Filters[i] = cf
		}
	}
}

func GetImageDimensions(path string) (int, int) {
	if path == "" {
		return 0, 0
	}
	file, err := os.Open(path)
	if err != nil {
		return 0, 0
	}
	defer file.Close()

	config, _, err := image.DecodeConfig(file)
	if err != nil {
		return 0, 0
	}
	return config.Width, config.Height
}

// NewLayersWithDimensions creates layers that scale based on input video dimensions
func NewLayersWithDimensions(inputWidth, inputHeight int) []RenderLayer {
	return []RenderLayer{
		NewBackgroundLayer(),
		NewGameplayLayer(inputWidth, inputHeight),
		NewFacecamLayer(inputWidth, inputHeight),
	}
}

// NewLayersFromPreset constructs the RenderLayers using configuration from the Preset
func NewLayersFromPreset(p *preset.Preset, inputWidth, inputHeight int) []RenderLayer {
	getVisible := func(v *bool) bool {
		if v == nil {
			return true
		}
		return *v
	}

	// 1. Background layer
	bgLayer := RenderLayer{
		Name:       "bg",
		InputIndex: 0,
		X:          0,
		Y:          0,
		Visible:    true,
		ZIndex:     0,
		Filters: []ffmpeg.Filter{
			ffmpeg.ScaleToFill{
				Width:  p.Crop.Width,
				Height: p.Crop.Height,
			},
			ffmpeg.Blur{
				Sigma: 25,
			},
		},
	}

	// 2. Gameplay layer
	gameVisible := getVisible(p.Gameplay.Visible)
	gameZIndex := p.Gameplay.ZIndex
	if gameZIndex == 0 {
		gameZIndex = 1
	}
	gameWidth := p.Gameplay.Width
	if gameWidth == 0 {
		gameWidth = 1800
	}
	gameHeight := p.Gameplay.Height
	if gameHeight == 0 {
		gameHeight = 1012
	}

	gameCropX := p.Gameplay.CropXPercent
	gameCropY := p.Gameplay.CropYPercent
	gameCropW := p.Gameplay.CropWidthPercent
	gameCropH := p.Gameplay.CropHeightPercent
	if gameCropW == 0 && gameCropH == 0 {
		gameCropW = 1.0
		gameCropH = 1.0
	}

	gameScaleMode := p.Gameplay.ScaleMode
	if gameScaleMode == "" {
		gameScaleMode = "stretch"
	}

	gameFilters := []ffmpeg.Filter{
		ffmpeg.CropPercent{
			XPercent:      gameCropX,
			YPercent:      gameCropY,
			WidthPercent:  gameCropW,
			HeightPercent: gameCropH,
			InputWidth:    inputWidth,
			InputHeight:   inputHeight,
		},
		ffmpeg.Scale{
			Width:     gameWidth,
			Height:    gameHeight,
			ScaleMode: gameScaleMode,
		},
	}

	gameLayer := RenderLayer{
		Name:       "game",
		InputIndex: 0,
		X:          p.Gameplay.X,
		Y:          p.Gameplay.Y,
		Visible:    gameVisible,
		ZIndex:     gameZIndex,
		Filters:    gameFilters,
	}

	// 3. Facecam layer
	camVisible := getVisible(p.Facecam.Visible)
	camZIndex := p.Facecam.ZIndex
	if camZIndex == 0 {
		camZIndex = 2
	}
	camWidth := p.Facecam.Width
	if camWidth == 0 {
		camWidth = 740
	}
	camHeight := p.Facecam.Height
	if camHeight == 0 {
		camHeight = 574
	}

	camCropX := p.Facecam.CropXPercent
	camCropY := p.Facecam.CropYPercent
	camCropW := p.Facecam.CropWidthPercent
	camCropH := p.Facecam.CropHeightPercent
	if camCropW == 0 && camCropH == 0 {
		camCropW = 1.0
		camCropH = 1.0
	}

	camScaleMode := p.Facecam.ScaleMode
	if camScaleMode == "" {
		camScaleMode = "stretch"
	}

	camFilters := []ffmpeg.Filter{
		ffmpeg.CropPercent{
			XPercent:      camCropX,
			YPercent:      camCropY,
			WidthPercent:  camCropW,
			HeightPercent: camCropH,
			InputWidth:    inputWidth,
			InputHeight:   inputHeight,
		},
		ffmpeg.Scale{
			Width:     camWidth,
			Height:    camHeight,
			ScaleMode: camScaleMode,
		},
		ffmpeg.DrawBox{
			X:         0,
			Y:         0,
			Width:     camWidth,
			Height:    camHeight,
			Color:     "white",
			Thickness: 4,
		},
	}

	camLayer := RenderLayer{
		Name:       "cam",
		InputIndex: 0,
		X:          p.Facecam.X,
		Y:          p.Facecam.Y,
		Visible:    camVisible,
		ZIndex:     camZIndex,
		Filters:    camFilters,
	}

	// 4. Overlay image layer
	overlayVisible := getVisible(p.Overlay.Visible)
	overlayZIndex := p.Overlay.ZIndex
	if overlayZIndex == 0 {
		overlayZIndex = 3
	}

	absImgPath := p.Overlay.Image
	if absImgPath != "" {
		if abs, err := filepath.Abs(absImgPath); err == nil {
			absImgPath = abs
		}
	}

	fileExists := func(path string) bool {
		if path == "" {
			return false
		}
		info, err := os.Stat(path)
		if err != nil {
			return false
		}
		return !info.IsDir()
	}

	imgW, imgH := GetImageDimensions(absImgPath)
	overlayWidth := p.Overlay.Width
	if overlayWidth == 0 {
		overlayWidth = imgW
		if overlayWidth == 0 {
			overlayWidth = 100
		}
	}
	overlayHeight := p.Overlay.Height
	if overlayHeight == 0 {
		overlayHeight = imgH
		if overlayHeight == 0 {
			overlayHeight = 100
		}
	}

	overlayCropX := p.Overlay.CropXPercent
	overlayCropY := p.Overlay.CropYPercent
	overlayCropW := p.Overlay.CropWidthPercent
	overlayCropH := p.Overlay.CropHeightPercent
	if overlayCropW == 0 && overlayCropH == 0 {
		overlayCropW = 1.0
		overlayCropH = 1.0
	}

	overlayScaleMode := p.Overlay.ScaleMode
	if overlayScaleMode == "" {
		overlayScaleMode = "stretch"
	}

	overlayFilters := []ffmpeg.Filter{
		ffmpeg.CropPercent{
			XPercent:      overlayCropX,
			YPercent:      overlayCropY,
			WidthPercent:  overlayCropW,
			HeightPercent: overlayCropH,
			InputWidth:    imgW,
			InputHeight:   imgH,
		},
		ffmpeg.Scale{
			Width:     overlayWidth,
			Height:    overlayHeight,
			ScaleMode: overlayScaleMode,
		},
	}

	overlayLayer := RenderLayer{
		Name:       "overlay",
		InputIndex: 1, // Map dynamically
		X:          p.Overlay.X,
		Y:          p.Overlay.Y,
		Visible:    overlayVisible && absImgPath != "" && fileExists(absImgPath),
		ZIndex:     overlayZIndex,
		ImagePath:  absImgPath,
		Filters:    overlayFilters,
	}

	layers := []RenderLayer{bgLayer, gameLayer, camLayer, overlayLayer}

	for _, extra := range p.ExtraLayers {
		extraVisible := getVisible(extra.Visible)
		extraZIndex := extra.ZIndex
		extraWidth := extra.Width
		extraHeight := extra.Height

		extraCropX := extra.CropXPercent
		extraCropY := extra.CropYPercent
		extraCropW := extra.CropWidthPercent
		extraCropH := extra.CropHeightPercent
		if extraCropW == 0 && extraCropH == 0 {
			extraCropW = 1.0
			extraCropH = 1.0
		}

		extraScaleMode := extra.ScaleMode
		if extraScaleMode == "" {
			extraScaleMode = "stretch"
		}

		extraFilters := []ffmpeg.Filter{
			ffmpeg.CropPercent{
				XPercent:      extraCropX,
				YPercent:      extraCropY,
				WidthPercent:  extraCropW,
				HeightPercent: extraCropH,
				InputWidth:    inputWidth,
				InputHeight:   inputHeight,
			},
			ffmpeg.Scale{
				Width:     extraWidth,
				Height:    extraHeight,
				ScaleMode: extraScaleMode,
			},
		}

		layers = append(layers, RenderLayer{
			Name:       extra.Name,
			InputIndex: 0,
			X:          extra.X,
			Y:          extra.Y,
			Visible:    extraVisible,
			ZIndex:     extraZIndex,
			Filters:    extraFilters,
		})
	}

	return layers
}

func NewBackgroundLayer() RenderLayer {
	return RenderLayer{
		Name:       "bg",
		InputIndex: 0,

		X: 0,
		Y: 0,

		Filters: []ffmpeg.Filter{
			ffmpeg.ScaleToFill{
				Width:  1080,
				Height: 1920,
			},

			ffmpeg.Blur{
				Sigma: 25,
			},
		},
	}
}

func NewFacecamLayer(inputWidth, inputHeight int) RenderLayer {
	return RenderLayer{
		Name:       "cam",
		InputIndex: 0,

		X: 170,
		Y: 60,

		Filters: []ffmpeg.Filter{
			ffmpeg.CropPercent{
				XPercent:      0.82,  // ~70% from left (1340/1920)
				YPercent:      0.6,   // 60% from top (648/1080)
				WidthPercent:  0.31,  // ~29% of width (557/1920)
				HeightPercent: 0.426, // ~42.6% of height (460/1080)
				InputWidth:    inputWidth,
				InputHeight:   inputHeight,
			},

			ffmpeg.Scale{
				Width:  740,
				Height: 574,
			},

			ffmpeg.DrawBox{
				X:         0,
				Y:         0,
				Width:     740,
				Height:    574,
				Color:     "white",
				Thickness: 4,
			},
		},
	}
}

func NewGameplayLayer(inputWidth, inputHeight int) RenderLayer {
	return RenderLayer{
		Name:       "game",
		InputIndex: 0,

		X: -250,
		Y: 450,

		Filters: []ffmpeg.Filter{
			ffmpeg.Scale{
				Width:  1800,
				Height: 1012,
			},
		},
	}
}

// BuildRenderingGraph resolves input indices for image overlays and compiles the inputs arguments for FFmpeg
func BuildRenderingGraph(concreteLayers []RenderLayer, inputVideoPath string) ([]string, []ffmpeg.RenderLayer) {
	inputArgs := []string{"-i", inputVideoPath}
	imageToIndex := make(map[string]int)
	currentIndex := 1

	// Sort a copy of concreteLayers by ZIndex ascending so overlay ordering is correct in FFmpeg
	layersCopy := make([]RenderLayer, len(concreteLayers))
	copy(layersCopy, concreteLayers)
	sort.SliceStable(layersCopy, func(i, j int) bool {
		return layersCopy[i].ZIndex < layersCopy[j].ZIndex
	})

	var activeLayers []ffmpeg.RenderLayer
	for _, layer := range layersCopy {
		if !layer.Visible {
			continue
		}

		lCopy := layer
		if lCopy.ImagePath != "" {
			idx, exists := imageToIndex[lCopy.ImagePath]
			if !exists {
				inputArgs = append(inputArgs, "-i", lCopy.ImagePath)
				idx = currentIndex
				imageToIndex[lCopy.ImagePath] = idx
				currentIndex++
			}
			lCopy.InputIndex = idx
		} else {
			lCopy.InputIndex = 0
		}
		activeLayers = append(activeLayers, lCopy)
	}

	return inputArgs, activeLayers
}
