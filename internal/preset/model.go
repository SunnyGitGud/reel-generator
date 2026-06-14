package preset

type Crop struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type Overlay struct {
	Image string `json:"image"`
	X     int    `json:"x"`
	Y     int    `json:"y"`
}

type LayerConfig struct {
	X                 int     `json:"x"`
	Y                 int     `json:"y"`
	Width             int     `json:"width"`
	Height            int     `json:"height"`
	CropXPercent      float64 `json:"crop_x_percent"`
	CropYPercent      float64 `json:"crop_y_percent"`
	CropWidthPercent  float64 `json:"crop_width_percent"`
	CropHeightPercent float64 `json:"crop_height_percent"`
}

type Preset struct {
	Crop     Crop        `json:"crop"`
	Overlay  Overlay     `json:"overlay"`
	Facecam  LayerConfig `json:"facecam"`
	Gameplay LayerConfig `json:"gameplay"`
}

type Layer struct {
	ID string `json:"id"`

	Type string `json:"type"`

	X int `json:"x"`
	Y int `json:"y"`

	Width  int `json:"width"`
	Height int `json:"height"`

	Blur int `json:"blur"`

	Crop *Crop `json:"crop,omitempty"`
}

type Composition struct {
	Width  int `json:"width"`
	Height int `json:"height"`

	Layers []Layer `json:"layers"`
}
