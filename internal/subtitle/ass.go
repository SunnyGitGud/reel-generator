package subtitle

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// GenerateASS generates an Advanced SubStation Alpha (.ass) file for FFmpeg
// The ASS format is used by FFmpeg's ass filter for subtitle rendering
func GenerateASS(srtFile string, style SubtitleStyle, outputPath string) error {
	// Parse SRT
	track, err := ParseSRT(srtFile)
	if err != nil {
		return fmt.Errorf("failed to parse SRT: %w", err)
	}

	// Generate ASS content
	content := generateASSContent(track, style)

	// Write to file
	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write ASS file: %w", err)
	}

	return nil
}

// GenerateASSFromTrack generates an ASS file from a SubtitleTrack
func GenerateASSFromTrack(track *SubtitleTrack, style SubtitleStyle, outputPath string) error {
	content := generateASSContent(track, style)
	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write ASS file: %w", err)
	}
	return nil
}

// generateASSContent creates the full ASS file content
func generateASSContent(track *SubtitleTrack, style SubtitleStyle) string {
	var sb strings.Builder

	// ASS file header
	sb.WriteString("[Script Info]\n")
	sb.WriteString("Title: Video Subtitles\n")
	sb.WriteString("Original Script: videdit\n")
	sb.WriteString(fmt.Sprintf("ScriptType: v4.00+\n"))
	sb.WriteString(fmt.Sprintf("Collisions: Normal\n"))
	sb.WriteString(fmt.Sprintf("PlayDepth: 0\n"))
	sb.WriteString(fmt.Sprintf("Timer: 100.0000\n"))
	sb.WriteString(fmt.Sprintf("WrapStyle: 0\n\n"))

	// V4+ Styles section
	sb.WriteString("[V4+ Styles]\n")
	sb.WriteString("Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding\n")

	// Write the style
	styleLine := generateASSStyleLine(style)
	sb.WriteString(styleLine)
	sb.WriteString("\n\n")

	// Events section
	sb.WriteString("[Events]\n")
	sb.WriteString("Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text\n")

	// Write all events
	for _, sub := range track.Items {
		eventLine := generateASSEventLine(sub, style.Name)
		sb.WriteString(eventLine)
		sb.WriteString("\n")
	}

	return sb.String()
}

// generateASSStyleLine creates an ASS style line
func generateASSStyleLine(style SubtitleStyle) string {
	bold := 0
	if style.Bold {
		bold = -1
	}
	italic := 0
	if style.Italic {
		italic = -1
	}
	underline := 0
	if style.Underline {
		underline = -1
	}
	strikeout := 0
	if style.StrikeOut {
		strikeout = -1
	}

	return fmt.Sprintf(
		"Style: %s,%s,%d,%s,&H00FFFFFF,%s,&H00000000,%d,%d,%d,%d,%.0f,%.0f,%.0f,%.0f,%d,%.1f,%.1f,%d,%d,%d,%d,1",
		style.Name,
		style.FontName,
		style.FontSize,
		style.PrimaryColor.ToHex(),
		style.OutlineColor.ToHex(),
		bold,
		italic,
		underline,
		strikeout,
		style.ScaleX,
		style.ScaleY,
		style.Spacing,
		style.Angle,
		style.BorderStyle,
		style.OutlineWidth,
		style.ShadowDepth,
		style.Alignment,
		style.MarginL,
		style.MarginR,
		style.MarginV,
	)
}

// generateASSEventLine creates an ASS event (subtitle) line
func generateASSEventLine(sub Subtitle, styleName string) string {
	startTime := formatASSTimestamp(sub.Start)
	endTime := formatASSTimestamp(sub.End)

	// Escape special characters in text
	text := escapeASSText(sub.Text)

	return fmt.Sprintf(
		"Dialogue: 0,%s,%s,%s,,0,0,0,,{%%c&H00B4E7FF&}%s",
		startTime,
		endTime,
		styleName,
		text,
	)
}

// formatASSTimestamp converts seconds to ASS timestamp format (H:MM:SS.CC)
func formatASSTimestamp(seconds float64) string {
	hours := int(seconds) / 3600
	minutes := (int(seconds) % 3600) / 60
	secs := int(seconds) % 60
	centisecs := int((seconds - float64(int(seconds))) * 100)

	return fmt.Sprintf("%d:%02d:%02d.%02d", hours, minutes, secs, centisecs)
}

// escapeASSText escapes special characters in ASS format
func escapeASSText(text string) string {
	// Replace line breaks
	text = strings.ReplaceAll(text, "\n", "\\N")
	// Escape some special characters if needed
	return text
}

// BurnSubtitles burns subtitles into a video using FFmpeg
func BurnSubtitles(input string, assFile string, output string) error {
	// Check if files exist
	if _, err := os.Stat(input); err != nil {
		return fmt.Errorf("input file not found: %w", err)
	}
	if _, err := os.Stat(assFile); err != nil {
		return fmt.Errorf("ASS file not found: %w", err)
	}

	// Get absolute path for ASS file (FFmpeg needs it sometimes)
	absAssPath, err := filepath.Abs(assFile)
	if err != nil {
		absAssPath = assFile
	}

	// Build FFmpeg command
	// The ass filter applies ASS subtitles
	cmd := exec.Command(
		"ffmpeg",
		"-i", input,
		"-vf", fmt.Sprintf("ass=%s", absAssPath),
		"-c:a", "aac",
		"-c:v", "libx264",
		"-preset", "fast",
		"-y", // Overwrite output
		output,
	)

	fmt.Printf("Running FFmpeg: %s\n", strings.Join(cmd.Args, " "))

	// Run command
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("FFmpeg failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// BurnSubtitlesWithOptions burns subtitles with custom FFmpeg options
type BurnOptions struct {
	VideoCodec      string // e.g., "libx264"
	AudioCodec      string // e.g., "aac"
	Preset          string // e.g., "fast", "medium", "slow"
	CRF             int    // Quality (0-51, default 23)
	Timeout         time.Duration
	ExtraFFmpegArgs []string
}

// DefaultBurnOptions returns sensible defaults
func DefaultBurnOptions() BurnOptions {
	return BurnOptions{
		VideoCodec: "libx264",
		AudioCodec: "aac",
		Preset:     "fast",
		CRF:        23,
		Timeout:    10 * time.Minute,
	}
}

// BurnSubtitlesWithOptions burns subtitles with custom options
func BurnSubtitlesWithOptions(input string, assFile string, output string, opts BurnOptions) error {
	if _, err := os.Stat(input); err != nil {
		return fmt.Errorf("input file not found: %w", err)
	}
	if _, err := os.Stat(assFile); err != nil {
		return fmt.Errorf("ASS file not found: %w", err)
	}

	absAssPath, err := filepath.Abs(assFile)
	if err != nil {
		absAssPath = assFile
	}

	args := []string{
		"-i", input,
		"-vf", fmt.Sprintf("ass=%s", absAssPath),
		"-c:a", opts.AudioCodec,
		"-c:v", opts.VideoCodec,
		"-preset", opts.Preset,
		"-crf", fmt.Sprintf("%d", opts.CRF),
	}

	args = append(args, opts.ExtraFFmpegArgs...)
	args = append(args, "-y", output)

	cmd := exec.Command("ffmpeg", args...)

	fmt.Printf("Running FFmpeg: %s\n", strings.Join(cmd.Args, " "))

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("FFmpeg failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}
