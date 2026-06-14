package ffmpeg

import (
	"fmt"
	"strings"
)

type Filter interface {
	Build() string
}

type Scale struct {
	Width  int
	Height int
}

func (s Scale) Build() string {
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

func BuildFilterChain(filters []Filter) string {
	var parts []string

	for _, filter := range filters {
		parts = append(parts, filter.Build())
	}

	return strings.Join(parts, ",")
}
