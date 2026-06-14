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

// ----------------------------------------------------
// BACKGROUND
// ----------------------------------------------------

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

// ----------------------------------------------------
// FACECAM
// ----------------------------------------------------

func NewFacecamLayer() RenderLayer {
	return RenderLayer{
		Name:       "cam",
		InputIndex: 0,

		X: 170,
		Y: 60,

		Filters: []ffmpeg.Filter{
			ffmpeg.Crop{
				X:      1340,
				Y:      648,
				Width:  557,
				Height: 460,
			},

			ffmpeg.Scale{
				Width:  740,
				Height: 574, // preserve aspect ratio
			},
		},
	}
}

// ----------------------------------------------------
// GAMEPLAY
// ----------------------------------------------------

func NewGameplayLayer() RenderLayer {
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
