package transcription

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Segment represents a single transcript segment from Whisper
type Segment struct {
	ID                int     `json:"id"`
	Seek              int     `json:"seek"`
	Start             float64 `json:"start"`
	End               float64 `json:"end"`
	Text              string  `json:"text"`
	Tokens            []int   `json:"tokens"`
	Temperature       float64 `json:"temperature"`
	AvgLogprob        float64 `json:"avg_logprob"`
	Compression_ratio float64 `json:"compression_ratio"`
	NoSpeechProb      float64 `json:"no_speech_prob"`
}

// TranscriptResult represents the complete transcript from Whisper
type TranscriptResult struct {
	Task     string    `json:"task"`
	Language string    `json:"language"`
	Duration float64   `json:"duration"`
	Segments []Segment `json:"segments"`
	Text     string    `json:"text"`
}

// TranscriptionOptions holds configuration for Whisper
type TranscriptionOptions struct {
	Model     string // tiny, base, small, medium, large
	Language  string // ISO-639-1 code (e.g., "en", "de")
	Device    string // "cpu" or "cuda"
	TaskType  string // "transcribe" or "translate"
	OutputDir string // Directory to save output files
	Romanize  bool   // Romanize Devanagari Hindi text to Hinglish
}

// DefaultOptions returns sensible defaults for transcription
func DefaultOptions() TranscriptionOptions {
	return TranscriptionOptions{
		Model:     "base",
		Language:  "auto",
		Device:    "cuda",
		TaskType:  "transcribe",
		OutputDir: "output",
		Romanize:  false,
	}
}

// TranscribeAudio transcribes an audio file using OpenAI's Whisper
func TranscribeAudio(audioPath string, opts TranscriptionOptions) (*TranscriptResult, error) {
	// Verify file exists
	if _, err := os.Stat(audioPath); err != nil {
		return nil, fmt.Errorf("audio file not found: %w", err)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Build command
	cmd := buildWhisperCommand(audioPath, opts)

	fmt.Printf("Running Whisper: %s\n", strings.Join(cmd.Args, " "))

	// Run command
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("whisper command failed: %w\nOutput: %s", err, string(output))
	}

	// Parse JSON output
	baseName := strings.TrimSuffix(filepath.Base(audioPath), filepath.Ext(audioPath))
	jsonPath := filepath.Join(opts.OutputDir, baseName+".json")

	// Post-process the JSON output to romanize Hindi Devanagari text if enabled
	if opts.Romanize {
		romanizeCmd := exec.Command("python", "internal/transcription/romanize.py", jsonPath)
		if romanizeOutput, err := romanizeCmd.CombinedOutput(); err != nil {
			fmt.Printf("Warning: Romanization post-processing failed: %v\nOutput: %s\n", err, string(romanizeOutput))
		}
	}

	result, err := parseWhisperJSON(jsonPath)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// buildWhisperCommand constructs the whisper CLI command
func buildWhisperCommand(audioPath string, opts TranscriptionOptions) *exec.Cmd {
	args := []string{
		audioPath,
		"--model", opts.Model,
	}

	if opts.Language != "" && opts.Language != "auto" {
		args = append(args, "--language", opts.Language)
	}

	args = append(args,
		"--device", opts.Device,
		"--task", opts.TaskType,
		"--output_format", "json",
		"--output_dir", opts.OutputDir,
	)

	cmd := exec.Command("whisper", args...)
	cmd.Env = append(os.Environ(), "PYTHONIOENCODING=utf-8")
	return cmd
}

// parseWhisperJSON reads and unmarshals the JSON output from Whisper
func parseWhisperJSON(jsonPath string) (*TranscriptResult, error) {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read whisper output: %w", err)
	}

	var result TranscriptResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse whisper JSON: %w", err)
	}

	return &result, nil
}

// GetFullTranscript returns the complete transcribed text
func (tr *TranscriptResult) GetFullTranscript() string {
	return tr.Text
}

// GetTranscriptWithTimestamps returns transcript with inline timestamps [HH:MM:SS]
func (tr *TranscriptResult) GetTranscriptWithTimestamps() string {
	var result strings.Builder

	for _, seg := range tr.Segments {
		timestamp := formatTimestamp(seg.Start)
		result.WriteString(fmt.Sprintf("[%s] %s\n", timestamp, strings.TrimSpace(seg.Text)))
	}

	return result.String()
}

// GetSegments returns all transcript segments
func (tr *TranscriptResult) GetSegments() []Segment {
	return tr.Segments
}

// GetSegmentByTime returns a segment for a specific timestamp
func (tr *TranscriptResult) GetSegmentByTime(timeSeconds float64) *Segment {
	for i := range tr.Segments {
		if tr.Segments[i].Start <= timeSeconds && timeSeconds <= tr.Segments[i].End {
			return &tr.Segments[i]
		}
	}
	return nil
}

// SaveToFile saves the transcript to a text file
func (tr *TranscriptResult) SaveToFile(filePath string) error {
	return os.WriteFile(filePath, []byte(tr.Text), 0644)
}

// splitSegment splits a segment into sub-segments of at most maxWords
func splitSegment(seg Segment, maxWords int) []Segment {
	words := strings.Fields(seg.Text)
	if len(words) <= maxWords {
		return []Segment{seg}
	}

	var subSegs []Segment
	totalWords := len(words)
	totalDuration := seg.End - seg.Start

	var currentStart = seg.Start
	for i := 0; i < totalWords; i += maxWords {
		endIdx := i + maxWords
		if endIdx > totalWords {
			endIdx = totalWords
		}

		chunkWords := words[i:endIdx]
		chunkText := strings.Join(chunkWords, " ")
		chunkWordCount := endIdx - i

		chunkDuration := totalDuration * (float64(chunkWordCount) / float64(totalWords))
		currentEnd := currentStart + chunkDuration

		if endIdx == totalWords {
			currentEnd = seg.End
		}

		subSegs = append(subSegs, Segment{
			ID:    seg.ID,
			Start: currentStart,
			End:   currentEnd,
			Text:  chunkText,
		})

		currentStart = currentEnd
	}

	return subSegs
}

// SaveSegmentsToSRT saves segments in SRT format with timing, splitting them to show at most 3 words at a time.
func (tr *TranscriptResult) SaveSegmentsToSRT(filePath string) error {
	var srt strings.Builder

	// Split segments to have at most 3 words on screen at a time
	const maxWords = 3
	var allSegs []Segment
	for _, seg := range tr.Segments {
		allSegs = append(allSegs, splitSegment(seg, maxWords)...)
	}

	for i, seg := range allSegs {
		srt.WriteString(fmt.Sprintf("%d\n", i+1))
		srt.WriteString(fmt.Sprintf("%s --> %s\n", formatTime(seg.Start), formatTime(seg.End)))
		srt.WriteString(fmt.Sprintf("%s\n\n", strings.TrimSpace(seg.Text)))
	}

	return os.WriteFile(filePath, []byte(srt.String()), 0644)
}

// formatTime converts seconds to SRT time format (HH:MM:SS,mmm)
func formatTime(seconds float64) string {
	hours := int(seconds) / 3600
	minutes := (int(seconds) % 3600) / 60
	secs := int(seconds) % 60
	millis := int((seconds - float64(int(seconds))) * 1000)

	return fmt.Sprintf("%02d:%02d:%02d,%03d", hours, minutes, secs, millis)
}

// formatTimestamp converts seconds to simple timestamp format (HH:MM:SS)
func formatTimestamp(seconds float64) string {
	hours := int(seconds) / 3600
	minutes := (int(seconds) % 3600) / 60
	secs := int(seconds) % 60

	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, secs)
}
