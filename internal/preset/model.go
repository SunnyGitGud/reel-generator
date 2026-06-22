package preset

type Crop struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type Overlay struct {
	Image             string  `json:"image"`
	X                 int     `json:"x"`
	Y                 int     `json:"y"`
	Width             int     `json:"width,omitempty"`
	Height            int     `json:"height,omitempty"`
	Visible           *bool   `json:"visible,omitempty"`
	ZIndex            int     `json:"z_index,omitempty"`
	ScaleMode         string  `json:"scale_mode,omitempty"`
	CropXPercent      float64 `json:"crop_x_percent,omitempty"`
	CropYPercent      float64 `json:"crop_y_percent,omitempty"`
	CropWidthPercent  float64 `json:"crop_width_percent,omitempty"`
	CropHeightPercent float64 `json:"crop_height_percent,omitempty"`
}

type LayerConfig struct {
	Name              string  `json:"name,omitempty"`
	X                 int     `json:"x"`
	Y                 int     `json:"y"`
	Width             int     `json:"width"`
	Height            int     `json:"height"`
	CropXPercent      float64 `json:"crop_x_percent"`
	CropYPercent      float64 `json:"crop_y_percent"`
	CropWidthPercent  float64 `json:"crop_width_percent"`
	CropHeightPercent float64 `json:"crop_height_percent"`
	Visible           *bool   `json:"visible,omitempty"`
	ZIndex            int     `json:"z_index,omitempty"`
	ScaleMode         string  `json:"scale_mode,omitempty"`
}

type SubtitlePresetConfig struct {
	FontName     string `json:"font_name,omitempty"`
	FontSize     int    `json:"font_size,omitempty"`
	PrimaryColor string `json:"primary_color,omitempty"`
	OutlineColor string `json:"outline_color,omitempty"`
	MarginV      int    `json:"margin_v,omitempty"`
	Alignment    int    `json:"alignment,omitempty"`
}

type Preset struct {
	Crop        Crop                  `json:"crop"`
	Overlay     Overlay               `json:"overlay"`
	Facecam     LayerConfig           `json:"facecam"`
	Gameplay    LayerConfig           `json:"gameplay"`
	ExtraLayers []LayerConfig         `json:"extra_layers,omitempty"`
	Subtitles   *SubtitlePresetConfig `json:"subtitles,omitempty"`
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

	Subtitle *SubtitleConfig `json:"subtitle,omitempty"` // Optional subtitle configuration
}

// SubtitleConfig defines subtitle layer configuration
type SubtitleConfig struct {
	SRTFile   string `json:"srt_file"`
	StyleName string `json:"style_name"` // e.g., "tiktok", "youtube", "custom"
}
