package subtitle

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SubtitleLayer represents a subtitle layer for video composition
type SubtitleLayer struct {
	SRTFile string
	Style   SubtitleStyle
	// Internal state
	assFile string
	track   *SubtitleTrack
}

// NewSubtitleLayer creates a new subtitle layer
func NewSubtitleLayer(srtFile string, style SubtitleStyle) *SubtitleLayer {
	return &SubtitleLayer{
		SRTFile: srtFile,
		Style:   style,
	}
}

// NewSubtitleLayerWithDefaults creates a subtitle layer with TikTok style
func NewSubtitleLayerWithDefaults(srtFile string) *SubtitleLayer {
	return NewSubtitleLayer(srtFile, TikTokStyle())
}

// GetName returns the layer name for FFmpeg filter graph
func (sl *SubtitleLayer) GetName() string {
	return "subs"
}

// Prepare initializes the subtitle layer and generates the ASS file
func (sl *SubtitleLayer) Prepare(outputDir string) error {
	// Parse SRT
	track, err := ParseSRT(sl.SRTFile)
	if err != nil {
		return fmt.Errorf("failed to parse SRT: %w", err)
	}
	sl.track = track

	// Generate ASS file
	assFileName := strings.TrimSuffix(filepath.Base(sl.SRTFile), filepath.Ext(sl.SRTFile)) + ".ass"
	sl.assFile = filepath.Join(outputDir, assFileName)

	if err := GenerateASSFromTrack(track, sl.Style, sl.assFile); err != nil {
		return fmt.Errorf("failed to generate ASS: %w", err)
	}

	return nil
}

// GetASSFile returns the generated ASS file path
func (sl *SubtitleLayer) GetASSFile() string {
	return sl.assFile
}

// GetTrack returns the parsed subtitle track
func (sl *SubtitleLayer) GetTrack() *SubtitleTrack {
	return sl.track
}

// GetFilterChain returns the FFmpeg filter chain for subtitles
// Note: This returns the ass filter that should be applied
func (sl *SubtitleLayer) GetFilterChain() string {
	if sl.assFile == "" {
		return ""
	}
	// Escape backslashes for Windows paths
	assPath := strings.ReplaceAll(sl.assFile, "\\", "\\\\")
	return fmt.Sprintf("ass='%s'", assPath)
}

// CompositeSubtitles applies subtitles to a video file
func (sl *SubtitleLayer) CompositeSubtitles(inputVideo string, outputVideo string) error {
	if sl.assFile == "" {
		return fmt.Errorf("subtitle layer not prepared - call Prepare() first")
	}

	return BurnSubtitles(inputVideo, sl.assFile, outputVideo)
}

// CompositeSubtitlesWithOptions applies subtitles with custom options
func (sl *SubtitleLayer) CompositeSubtitlesWithOptions(inputVideo string, outputVideo string, opts BurnOptions) error {
	if sl.assFile == "" {
		return fmt.Errorf("subtitle layer not prepared - call Prepare() first")
	}

	return BurnSubtitlesWithOptions(inputVideo, sl.assFile, outputVideo, opts)
}

// SetStyle updates the subtitle style
func (sl *SubtitleLayer) SetStyle(style SubtitleStyle) {
	sl.Style = style
}

// IsReady returns true if the layer has been prepared
func (sl *SubtitleLayer) IsReady() bool {
	return sl.assFile != "" && fileExists(sl.assFile)
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
