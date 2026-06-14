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

type Preset struct {
	Crop    Crop    `json:"crop"`
	Overlay Overlay `json:"overlay"`
}

type Layer struct {
	Type   string
	Source string

	X int
	Y int

	Width  int
	Height int

	Blur int

	Crop *Crop
}

type Composition struct {
	Width  int
	Height int

	Layers []Layer
}
