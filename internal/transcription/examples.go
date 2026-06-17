package transcription

import (
	"fmt"
	"log"
)

// ExampleTranscribeBasic demonstrates basic transcription usage
func ExampleTranscribeBasic() {
	opts := DefaultOptions()
	opts.Model = "base"

	result, err := TranscribeAudio("videos/input.mp4", opts)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Full Transcript:")
	fmt.Println(result.GetFullTranscript())
}

// ExampleTranscribeWithDetails demonstrates detailed usage with segment information
func ExampleTranscribeWithDetails() {
	opts := DefaultOptions()
	opts.Model = "small"
	opts.Language = "en"
	opts.OutputDir = "output"

	result, err := TranscribeAudio("videos/input.mp4", opts)
	if err != nil {
		log.Fatal(err)
	}

	// Save outputs
	result.SaveToFile("output/transcript.txt")
	result.SaveSegmentsToSRT("output/transcript.srt")

	// Display segments
	fmt.Printf("Language: %s\nDuration: %.2f seconds\n", result.Language, result.Duration)
	fmt.Println("\nSegments:")

	for _, seg := range result.GetSegments() {
		fmt.Printf("[%06.2f - %06.2f] %s\n", seg.Start, seg.End, seg.Text)
	}
}

// ExampleTranscribeMultipleLanguages demonstrates transcription with automatic language detection
func ExampleTranscribeMultipleLanguages() {
	// Let Whisper auto-detect language (use empty string)
	opts := DefaultOptions()
	opts.Model = "base"
	opts.Language = "auto" // Note: Whisper will detect automatically

	result, err := TranscribeAudio("videos/input.mp4", opts)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Detected language: %s\n", result.Language)
	fmt.Println(result.GetFullTranscript())
}

// ExampleTranscribeWithTranslation demonstrates translation to English
func ExampleTranscribeWithTranslation() {
	opts := DefaultOptions()
	opts.Model = "base"
	opts.Language = "es"        // Spanish
	opts.TaskType = "translate" // Translate to English

	result, err := TranscribeAudio("videos/input.mp4", opts)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Translated to English:")
	fmt.Println(result.GetFullTranscript())
}

// ExampleGetTranscriptAtTimestamp demonstrates retrieving transcript at specific time
func ExampleGetTranscriptAtTimestamp() {
	opts := DefaultOptions()
	opts.Model = "base"

	result, err := TranscribeAudio("videos/input.mp4", opts)
	if err != nil {
		log.Fatal(err)
	}

	// Get transcript at specific timestamps
	timestamps := []float64{0.0, 5.5, 10.0, 15.5}

	for _, ts := range timestamps {
		seg := result.GetSegmentByTime(ts)
		if seg != nil {
			fmt.Printf("At %.1f seconds: %s\n", ts, seg.Text)
		}
	}
}
