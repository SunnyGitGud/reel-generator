package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/yourname/video-editor/internal/actions"
	"github.com/yourname/video-editor/internal/ffmpeg"
	"github.com/yourname/video-editor/internal/renderer"
	"github.com/yourname/video-editor/internal/subtitle"
	"github.com/yourname/video-editor/internal/transcription"
)

func main() {
	// Standard input dimensions (1920x1080 for most streamers)
	// These can be made configurable from preset later
	inputWidth := 1920
	inputHeight := 1080

	// Define command line flag for input directory
	videosDirFlag := flag.String("dir", "videos", "Directory containing input videos")
	flag.Parse()

	videosDir := *videosDirFlag

	// Read files in the directory
	files, err := os.ReadDir(videosDir)
	if err != nil {
		log.Fatalf("Failed to read videos directory %s: %v", videosDir, err)
	}

	var videoFiles []string
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(file.Name()))
		if ext == ".mp4" || ext == ".mov" || ext == ".mkv" || ext == ".avi" {
			videoFiles = append(videoFiles, filepath.Join(videosDir, file.Name()))
		}
	}

	if len(videoFiles) == 0 {
		log.Printf("No video files found in directory: %s\n", videosDir)
		return
	}

	log.Printf("Found %d video(s) to process:\n", len(videoFiles))
	for _, vf := range videoFiles {
		log.Printf("  - %s\n", vf)
	}

	// Ensure output directory exists
	if err := os.MkdirAll("output", 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Process each video
	for _, videoInputPath := range videoFiles {
		log.Printf("\n========================================")
		log.Printf("Processing video: %s", videoInputPath)
		log.Printf("========================================\n")

		baseName := filepath.Base(videoInputPath)
		inputBaseNoExt := strings.TrimSuffix(baseName, filepath.Ext(baseName))
		finalOutputPath := filepath.Join("output", baseName)

		// --- Transcription Step ---
		transcriptSRTPath := filepath.Join("output", inputBaseNoExt+".srt")

		opts := transcription.DefaultOptions()
		opts.Model = "small" // Using small model for better quality
		opts.Language = "en"
		opts.OutputDir = "output"

		log.Printf("[%s] Starting audio transcription...\n", baseName)
		transcriptResult, err := transcription.TranscribeAudio(videoInputPath, opts)
		if err != nil {
			log.Printf("[%s] Transcription failed: %v\n", baseName, err)
			continue
		}

		// Save transcript to SRT for subtitles
		err = transcriptResult.SaveSegmentsToSRT(transcriptSRTPath)
		if err != nil {
			log.Printf("[%s] Failed to save SRT: %v\n", baseName, err)
			continue
		}
		log.Printf("[%s] Transcript saved to %s\n", baseName, transcriptSRTPath)

		// --- Subtitle Layer Setup ---
		subtitleLayer := subtitle.NewSubtitleLayerWithDefaults(transcriptSRTPath)

		// Prepare the subtitle layer (generates .ass file)
		err = subtitleLayer.Prepare("output")
		if err != nil {
			log.Printf("[%s] Failed to prepare subtitle layer: %v\n", baseName, err)
			continue
		}
		log.Printf("[%s] Generated ASS file: %s\n", baseName, subtitleLayer.GetASSFile())

		// --- Video Rendering with Subtitles ---
		concreteLayers := actions.NewLayersWithDimensions(inputWidth, inputHeight)

		layers := make([]ffmpeg.RenderLayer, len(concreteLayers))
		for i, layer := range concreteLayers {
			layers[i] = layer
		}

		graph := ffmpeg.BuildFilterGraph(layers, subtitleLayer.GetASSFile())

		log.Printf("[%s] Starting video rendering...\n", baseName)
		err = renderer.Render(
			videoInputPath,
			finalOutputPath,
			graph,
		)
		if err != nil {
			log.Printf("[%s] Rendering failed: %v\n", baseName, err)
			continue
		}

		log.Printf("[%s] Video rendered successfully to %s\n", baseName, finalOutputPath)
	}
}
