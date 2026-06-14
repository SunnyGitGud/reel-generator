package actions

import "github.com/yourname/video-editor/internal/ffmpeg"

type RenderLayer struct {
	Name string

	InputIndex int

	X int
	Y int

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

// NewLayersWithDimensions creates layers that scale based on input video dimensions
func NewLayersWithDimensions(inputWidth, inputHeight int) []RenderLayer {
	return []RenderLayer{
		NewBackgroundLayer(),
		NewGameplayLayer(inputWidth, inputHeight),
		NewFacecamLayer(inputWidth, inputHeight),
	}
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
				XPercent:      0.697, // ~70% from left (1340/1920)
				YPercent:      0.6,   // 60% from top (648/1080)
				WidthPercent:  0.29,  // ~29% of width (557/1920)
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
