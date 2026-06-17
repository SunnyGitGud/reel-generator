package main

import (
	"log"
	"os"
	"path/filepath"

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
	videoInputPath := "videos/input.mp4"
	finalOutputPath := "output/result.mp4"

	// Ensure output directory exists for intermediate files
	if err := os.MkdirAll("output", 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// --- Transcription Step ---
	_ = filepath.Join("output", "extracted_audio.mp3")
	transcriptSRTPath := filepath.Join("output", "transcript.srt")

	// Ideally, extract audio first if the input is video
	// For simplicity, directly using videoInputPath for transcription if Whisper can handle it
	opts := transcription.DefaultOptions()
	opts.Model = "small" // Using small model for better quality
	opts.OutputDir = "output"

	log.Println("Starting audio transcription...")
	transcriptResult, err := transcription.TranscribeAudio(videoInputPath, opts)
	if err != nil {
		log.Fatalf("Transcription failed: %v", err)
	}

	// Save transcript to SRT for subtitles
	err = transcriptResult.SaveSegmentsToSRT(transcriptSRTPath)
	if err != nil {
		log.Fatalf("Failed to save SRT: %v", err)
	}
	log.Printf("Transcript saved to %s\n", transcriptSRTPath)

	// --- Subtitle Layer Setup ---
	// Create a subtitle layer with TikTok style
	subtitleLayer := subtitle.NewSubtitleLayerWithDefaults(transcriptSRTPath)

	// Prepare the subtitle layer (generates .ass file)
	err = subtitleLayer.Prepare("output") // Output ASS to the output directory
	if err != nil {
		log.Fatalf("Failed to prepare subtitle layer: %v", err)
	}
	log.Printf("Generated ASS file: %s\n", subtitleLayer.GetASSFile())

	// --- Video Rendering with Subtitles ---
	// Create existing layers with dimension-aware cropping
	concreteLayers := actions.NewLayersWithDimensions(inputWidth, inputHeight)

	// Convert to interface slice for FFmpeg filter graph builder
	layers := make([]ffmpeg.RenderLayer, len(concreteLayers))
	for i, layer := range concreteLayers {
		layers[i] = layer
	}

	// Build the FFmpeg filter graph, now including the ASS file for burning
	graph := ffmpeg.BuildFilterGraph(layers, subtitleLayer.GetASSFile())

	log.Println("Starting video rendering...")
	err = renderer.Render(
		videoInputPath,
		finalOutputPath,
		graph,
	)

	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Video rendered successfully to %s\n", finalOutputPath)
}
