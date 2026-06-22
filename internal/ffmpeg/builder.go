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

type KeepSegment struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

func BuildFilterGraph(layers []RenderLayer, assFilePath string, keepSegments []KeepSegment) string {
	if len(layers) == 0 {
		return ""
	}

	var finalVideoLabel string
	if len(keepSegments) > 0 {
		finalVideoLabel = "[comp_v]"
	}

	var sections []string

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

		if finalVideoLabel != "" {
			sections = append(sections, fmt.Sprintf("%s%s", finalChain, finalVideoLabel))
		} else {
			return finalChain
		}
	} else {
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
				} else if finalVideoLabel != "" {
					output = "comp_v"
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
			if finalVideoLabel != "" {
				sections = append(sections, fmt.Sprintf("[%s]%s%s", current, BurnSubtitlesFilter{ASSFilePath: assFilePath}.Build(), finalVideoLabel))
			} else {
				sections = append(sections, fmt.Sprintf("[%s]%s", current, BurnSubtitlesFilter{ASSFilePath: assFilePath}.Build()))
			}
		}
	}

	// Slicing and concatenation if keepSegments are provided
	if len(keepSegments) > 0 {
		n := len(keepSegments)

		// Create video splits
		vSplits := make([]string, n)
		for i := 0; i < n; i++ {
			vSplits[i] = fmt.Sprintf("[v_seg%d]", i)
		}
		sections = append(sections, fmt.Sprintf("[comp_v]split=%d%s", n, strings.Join(vSplits, "")))

		// Create audio splits
		aSplits := make([]string, n)
		for i := 0; i < n; i++ {
			aSplits[i] = fmt.Sprintf("[a_seg%d]", i)
		}
		sections = append(sections, fmt.Sprintf("[0:a]asplit=%d%s", n, strings.Join(aSplits, "")))

		var concatInputs []string
		for i, seg := range keepSegments {
			vLabel := fmt.Sprintf("v%d", i)
			aLabel := fmt.Sprintf("a%d", i)
			sections = append(sections, fmt.Sprintf("[v_seg%d]trim=start=%.3f:end=%.3f,setpts=PTS-STARTPTS[%s]", i, seg.Start, seg.End, vLabel))
			sections = append(sections, fmt.Sprintf("[a_seg%d]atrim=start=%.3f:end=%.3f,asetpts=PTS-STARTPTS[%s]", i, seg.Start, seg.End, aLabel))
			concatInputs = append(concatInputs, fmt.Sprintf("[%s][%s]", vLabel, aLabel))
		}
		sections = append(sections, fmt.Sprintf("%sconcat=n=%d:v=1:a=1", strings.Join(concatInputs, ""), n))
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
