package ffmpeg

import (
	"fmt"
	"strings"
)

type Filter interface {
	Build() string
}

type Scale struct {
	Width     int
	Height    int
	ScaleMode string // "stretch", "fit", "fill"
}

func (s Scale) Build() string {
	if s.ScaleMode == "fit" {
		return fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease", s.Width, s.Height)
	}
	if s.ScaleMode == "fill" {
		return fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=increase,crop=%d:%d", s.Width, s.Height, s.Width, s.Height)
	}
	return fmt.Sprintf("scale=%d:%d", s.Width, s.Height)
}

type Crop struct {
	X      int
	Y      int
	Width  int
	Height int
}

func (c Crop) Build() string {
	return fmt.Sprintf(
		"crop=%d:%d:%d:%d",
		c.Width,
		c.Height,
		c.X,
		c.Y,
	)
}

type Blur struct {
	Sigma int
}

func (b Blur) Build() string {
	return fmt.Sprintf("gblur=sigma=%d", b.Sigma)
}

type FPS struct {
	Value int
}

func (f FPS) Build() string {
	return fmt.Sprintf("fps=%d", f.Value)
}

type ScaleToFill struct {
	Width  int
	Height int
}

func (s ScaleToFill) Build() string {
	return fmt.Sprintf("scale=-1:%d,crop=%d:%d", s.Height, s.Width, s.Height)
}

type CropPercent struct {
	XPercent      float64 // 0.0 to 1.0
	YPercent      float64 // 0.0 to 1.0
	WidthPercent  float64 // 0.0 to 1.0
	HeightPercent float64 // 0.0 to 1.0
	InputWidth    int
	InputHeight   int
}

func (c CropPercent) Build() string {
	X := int(float64(c.InputWidth) * c.XPercent)
	Y := int(float64(c.InputHeight) * c.YPercent)
	W := int(float64(c.InputWidth) * c.WidthPercent)
	H := int(float64(c.InputHeight) * c.HeightPercent)
	return fmt.Sprintf("crop=%d:%d:%d:%d", W, H, X, Y)
}

type DrawBox struct {
	X         int    // X coordinate
	Y         int    // Y coordinate
	Width     int    // Width of box
	Height    int    // Height of box
	Color     string // Color (e.g., "white", "0xFFFFFF")
	Thickness int    // Border thickness in pixels
}

func (d DrawBox) Build() string {
	return fmt.Sprintf("drawbox=x=%d:y=%d:w=%d:h=%d:color=%s:thickness=%d", d.X, d.Y, d.Width, d.Height, d.Color, d.Thickness)
}

func BuildFilterChain(filters []Filter) string {
	var parts []string

	for _, filter := range filters {
		parts = append(parts, filter.Build())
	}

	return strings.Join(parts, ",")
}
