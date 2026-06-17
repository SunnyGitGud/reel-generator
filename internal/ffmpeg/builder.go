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

func BuildFilterGraph(layers []RenderLayer, assFilePath string) string {
	if len(layers) == 0 {
		return ""
	}

	if len(layers) == 1 {
		layer := layers[0]
		input := fmt.Sprintf("[%d:v]", layer.GetInputIndex())
		filterChain := BuildFilterChain(layer.GetFilters())

		var finalChain string
		if filterChain != "" {
			finalChain = fmt.Sprintf("%s%s", input, filterChain)
		} else {
			finalChain = input
		}

		if assFilePath != "" {
			finalChain = fmt.Sprintf("%s,%s", finalChain, BurnSubtitlesFilter{ASSFilePath: assFilePath}.Build())
		}
		return finalChain
	}

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

	// For multiple layers, complex blending is done.
	current := layers[0].GetName()
	for i := 1; i < len(layers); i++ {
		next := layers[i]

		var output string
		if i == len(layers)-1 {
			if assFilePath != "" {
				output = "merged"
			} else {
				output = "" // unlabeled final output
			}
		} else {
			output = fmt.Sprintf("tmp%d", i)
		}

		var section string
		if output != "" {
			section = fmt.Sprintf("[%s][%s]overlay=%d:%d[%s]", current, next.GetName(), next.GetX(), next.GetY(), output)
			current = output
		} else {
			section = fmt.Sprintf("[%s][%s]overlay=%d:%d", current, next.GetName(), next.GetX(), next.GetY())
		}
		sections = append(sections, section)
	}

	// Add subtitles as the very last filter on the final output stream
	if assFilePath != "" {
		sections = append(sections, fmt.Sprintf("[%s]%s", current, BurnSubtitlesFilter{ASSFilePath: assFilePath}.Build()))
	}

	return strings.Join(sections, ";")
}

// BurnSubtitlesFilter represents the FFmpeg 'ass' filter for burning subtitles.
type BurnSubtitlesFilter struct {
	ASSFilePath string
}

func (b BurnSubtitlesFilter) Build() string {
	// Escape backslashes for Windows paths in FFmpeg filter string
	// FFmpeg expects path separators to be escaped in the filtergraph
	assPath := strings.ReplaceAll(b.ASSFilePath, "\\", "\\\\")
	return fmt.Sprintf("ass='%s'", assPath)
}
