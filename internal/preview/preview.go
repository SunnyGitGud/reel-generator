package preview

import (
	"fmt"
	"os/exec"

	"github.com/yourname/video-editor/internal/ffmpeg"
)

type PreviewService struct {
	InputVideo string
}

func NewPreviewService(inputVideo string) *PreviewService {
	return &PreviewService{
		InputVideo: inputVideo,
	}
}

func (s *PreviewService) GeneratePreview(inputArgs []string, layers []ffmpeg.RenderLayer, outputPath string) error {
	// Build filter graph without subtitles for composition layout editing
	filterGraph := ffmpeg.BuildFilterGraph(layers, "")

	// Construct final arguments
	args := append(inputArgs, "-filter_complex", filterGraph, "-vframes", "1", outputPath)

	cmd := exec.Command("ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg preview extraction failed: %w, output: %s", err, string(output))
	}
	return nil
}
