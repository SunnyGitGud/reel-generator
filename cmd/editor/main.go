package main

import (
	"log"

	"github.com/yourname/video-editor/internal/actions"
	"github.com/yourname/video-editor/internal/ffmpeg"
	"github.com/yourname/video-editor/internal/renderer"
)

func main() {

	concreteLayeres := []actions.RenderLayer{
		actions.NewBackgroundLayer(),
		actions.NewGameplayLayer(),
		actions.NewFacecamLayer(),
	}

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
