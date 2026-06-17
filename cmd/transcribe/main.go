package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/yourname/video-editor/internal/transcription"
)

// formatTimeCode converts seconds to SRT timecode format (HH:MM:SS,mmm)
func formatTimeCode(seconds float64) string {
	hours := int(seconds) / 3600
	minutes := (int(seconds) % 3600) / 60
	secs := int(seconds) % 60
	millis := int((seconds - float64(int(seconds))) * 1000)
	return fmt.Sprintf("%02d:%02d:%02d,%03d", hours, minutes, secs, millis)
}

func main() {
	// Command line flags
	audioFile := flag.String("file", "", "Path to audio/video file to transcribe")
	model := flag.String("model", "base", "Whisper model: tiny, base, small, medium, large")
	language := flag.String("lang", "en", "Language code (e.g., en, es, fr, de)")
	outputDir := flag.String("output", "output", "Output directory for results")
	task := flag.String("task", "transcribe", "Task: transcribe or translate")
	device := flag.String("device", "cuda", "Device: cpu or cuda")
	format := flag.String("format", "all", "Output format: txt, srt, or all")
	verbose := flag.Bool("v", false, "Verbose output")

	flag.Parse()

	if *audioFile == "" {
		fmt.Println("Usage: go run cmd/transcribe/main.go -file <path> [options]")
		fmt.Println("\nOptions:")
		flag.PrintDefaults()
		fmt.Println("\nExamples:")
		fmt.Println("  go run cmd/transcribe/main.go -file videos/input.mp4")
		fmt.Println("  go run cmd/transcribe/main.go -file audio.mp3 -model small -lang es -task translate")
		os.Exit(1)
	}

	// Create options
	opts := transcription.TranscriptionOptions{
		Model:     *model,
		Language:  *language,
		Device:    *device,
		TaskType:  *task,
		OutputDir: *outputDir,
	}

	if *verbose {
		fmt.Printf("Transcription Options:\n")
		fmt.Printf("  File: %s\n", *audioFile)
		fmt.Printf("  Model: %s\n", opts.Model)
		fmt.Printf("  Language: %s\n", opts.Language)
		fmt.Printf("  Task: %s\n", opts.TaskType)
		fmt.Printf("  Device: %s\n", opts.Device)
		fmt.Printf("  Output Dir: %s\n", opts.OutputDir)
		fmt.Println()
	}

	// Transcribe
	fmt.Println("Starting transcription...")
	result, err := transcription.TranscribeAudio(*audioFile, opts)
	if err != nil {
		log.Fatalf("Transcription failed: %v", err)
	}

	fmt.Printf("\nTranscription complete!\n")
	fmt.Printf("Language: %s\n", result.Language)
	fmt.Printf("Duration: %.2f seconds\n", result.Duration)
	fmt.Printf("Segments: %d\n\n", len(result.Segments))

	// Save outputs
	if *format == "txt" || *format == "all" {
		txtFile := *outputDir + "/transcript.txt"
		result.SaveToFile(txtFile)
		fmt.Printf("✓ Saved: %s\n", txtFile)
	}

	if *format == "txt-timestamps" || *format == "all" {
		data := result.GetTranscriptWithTimestamps()
		tsFile := *outputDir + "/transcript_timestamps.txt"
		os.WriteFile(tsFile, []byte(data), 0644)
		fmt.Printf("✓ Saved: %s\n", tsFile)
	}

	if *format == "srt" || *format == "all" {
		srtFile := *outputDir + "/transcript.srt"
		result.SaveSegmentsToSRT(srtFile)
		fmt.Printf("✓ Saved: %s\n", srtFile)
	}

	// Display transcript
	if *format == "srt" || *format == "all" {
		fmt.Println("\n=== TRANSCRIPT WITH TIMESTAMPS (SRT Format) ===")
		for i, seg := range result.GetSegments() {
			fmt.Printf("%d\n%s --> %s\n%s\n\n",
				i+1,
				formatTimeCode(seg.Start),
				formatTimeCode(seg.End),
				seg.Text)
		}
	} else if *format == "txt-timestamps" {
		fmt.Println("\n=== TRANSCRIPT WITH TIMESTAMPS ===")
		fmt.Println(result.GetTranscriptWithTimestamps())
	} else {
		fmt.Println("\n=== TRANSCRIPT ===")
		fmt.Println(result.GetFullTranscript())
	}

	if *verbose {
		fmt.Println("\n=== DETAILED SEGMENTS ===")
		for _, seg := range result.GetSegments() {
			fmt.Printf("[%06.2f - %06.2f] %s\n", seg.Start, seg.End, seg.Text)
		}
	}
}
