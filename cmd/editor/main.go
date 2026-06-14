package main

import (
	"log"

	"github.com/yourname/video-editor/internal/actions"
	"github.com/yourname/video-editor/internal/ffmpeg"
	"github.com/yourname/video-editor/internal/renderer"
)

func main() {
	// Standard input dimensions (1920x1080 for most streamers)
	// These can be made configurable from preset later
	inputWidth := 1920
	inputHeight := 1080

	// Create layers with dimension-aware cropping
	concreteLayeres := actions.NewLayersWithDimensions(inputWidth, inputHeight)

	// Convert to interface slice
	layers := make([]ffmpeg.RenderLayer, len(concreteLayeres))
	for i, layer := range concreteLayeres {
		layers[i] = layer
	}

	graph := ffmpeg.BuildFilterGraph(layers)

	err := renderer.Render(
		"videos/input.mp4",
		"output/result.mp4",
		graph,
	)

	if err != nil {
		log.Fatal(err)
	}
}
