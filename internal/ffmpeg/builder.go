package ffmpeg

import (
	"fmt"
	"strings"
)

type RenderLayer interface {
	GetName() string
	GetInputIndex() int
	GetX() int
	GetY() int
	GetFilters() []Filter
}

func BuildFilterGraph(layers []RenderLayer) string {

	var sections []string

	for _, layer := range layers {

		input := fmt.Sprintf("[%d:v]", layer.GetInputIndex())

		filterChain := BuildFilterChain(layer.GetFilters())

		section := fmt.Sprintf(
			"%s%s[%s]",
			input,
			filterChain,
			layer.GetName(),
		)

		sections = append(sections, section)
	}

	if len(layers) < 2 {
		return strings.Join(sections, ";")
	}

	current := layers[0].GetName()

	for i := 1; i < len(layers); i++ {

		next := layers[i]

		output := "out"

		if i < len(layers)-1 {
			output = fmt.Sprintf("tmp%d", i)
		}

		overlay := fmt.Sprintf(
			"[%s][%s]overlay=%d:%d[%s]",
			current,
			next.GetName(),
			next.GetX(),
			next.GetY(),
			output,
		)

		sections = append(sections, overlay)

		current = output
	}

	return strings.Join(sections, ";")
}

func BuildCommand(inputFile string, outputFile string, layers []RenderLayer) []string {
	var args []string
	args = append(args, "-i", inputFile)

	if len(layers) > 0 {
		filterGraph := BuildFilterGraph(layers)
		args = append(args, "-filter_complex", filterGraph)
		args = append(args, "-map", "[out]")
	}

	args = append(args, "-c:v", "libx264")
	args = append(args, "-preset", "fast")
	args = append(args, outputFile)
	return args
}
