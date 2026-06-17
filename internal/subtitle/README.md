# Subtitle Rendering Engine - TikTok/YouTube Shorts Style

A modern subtitle rendering engine for creating professional social media captions with support for multiple styles including TikTok, YouTube Shorts, Instagram Reels, and custom styles.

## Features

✨ **Key Features**
- Parse SRT subtitle files
- Generate modern social media caption styles
- Burn subtitles into videos using FFmpeg
- Support for TikTok, YouTube Shorts, Instagram Reels styles
- Customizable fonts, colors, and styling
- ASS/SSA subtitle format generation
- Timed subtitle queries

## Quick Start

### 1. Parse SRT File
```go
import "github.com/yourname/video-editor/internal/subtitle"

track, err := subtitle.ParseSRT("transcript.srt")
if err != nil {
	log.Fatal(err)
}
fmt.Printf("Loaded %d subtitles\n", track.Count())
```

### 2. Generate and Burn Subtitles
```go
// Use TikTok style
style := subtitle.TikTokStyle()

// Generate ASS file for FFmpeg
err := subtitle.GenerateASSFromTrack(track, style, "output/subs.ass")

// Burn into video
err = subtitle.BurnSubtitles("input.mp4", "output/subs.ass", "output.mp4")
```

### 3. Using SubtitleLayer
```go
// Create layer with default TikTok style
layer := subtitle.NewSubtitleLayerWithDefaults("transcript.srt")

// Prepare (parse SRT, generate ASS)
layer.Prepare("output")

// Apply to video
layer.CompositeSubtitles("input.mp4", "output.mp4")
```

## Subtitle Styles

### Built-in Styles

**TikTok Style** (Default)
```go
style := subtitle.TikTokStyle()
// Features: Yellow text, black outline, 60pt Montserrat Bold
```

**YouTube Shorts Style**
```go
style := subtitle.YouTubeShortsStyle()
// Features: White text, thick black outline, 64pt Roboto Bold
```

**Instagram Reels Style**
```go
style := subtitle.InstagramReelsStyle()
// Features: White text, subtle shadow, 56pt Avenir Next
```

**Classic Style**
```go
style := subtitle.ClassicStyle()
// Features: White text, black outline, 48pt Arial
```

### Custom Style
```go
customStyle := subtitle.SubtitleStyle{
	Name:          "MyStyle",
	FontName:      "Roboto",
	FontSize:      70,
	PrimaryColor:  subtitle.ColorCyan,
	OutlineColor:  subtitle.ColorBlack,
	BackColor:     subtitle.ColorBlack,
	Bold:          true,
	Italic:        false,
	Underline:     false,
	StrikeOut:     false,
	ScaleX:        100,
	ScaleY:        100,
	Spacing:       0,
	Angle:         0,
	BorderStyle:   1,      // Outline + shadow
	OutlineWidth:  3.5,
	ShadowDepth:   2,
	Alignment:     2,      // Bottom center
	MarginL:       10,
	MarginR:       10,
	MarginV:       20,
	Encoding:      1,      // Unicode
}

subtitle.GenerateASSFromTrack(track, customStyle, "output/subs.ass")
```

## Available Colors

```go
// Predefined colors
subtitle.ColorYellow
subtitle.ColorBlack
subtitle.ColorWhite
subtitle.ColorRed
subtitle.ColorGreen
subtitle.ColorBlue
subtitle.ColorCyan
subtitle.ColorMagenta

// Custom color
customColor := subtitle.NewColor(200, 150, 100)
```

## API Reference

### Types

#### Subtitle
```go
type Subtitle struct {
	Start float64 // Start time in seconds
	End   float64 // End time in seconds
	Text  string  // Caption text
	Index int     // Sequential index
}
```

#### SubtitleTrack
```go
type SubtitleTrack struct {
	Items []Subtitle
}

// Methods
track.Add(subtitle)                    // Add subtitle
track.GetAtTime(seconds)              // Get subtitle at time
track.GetRange(start, end)            // Get subtitles in range
track.Duration()                      // Total duration
track.Count()                         // Number of subtitles
```

#### SubtitleStyle
```go
type SubtitleStyle struct {
	Name             string  // Style name
	FontName         string  // Font family
	FontSize         int     // Font size in pixels
	PrimaryColor     Color   // Text color
	OutlineColor     Color   // Outline color
	BackColor        Color   // Shadow color
	Bold             bool
	Italic           bool
	Underline        bool
	StrikeOut        bool
	ScaleX           float64 // Horizontal scale
	ScaleY           float64 // Vertical scale
	Spacing          float64 // Letter spacing
	Angle            float64 // Rotation angle
	BorderStyle      int     // 1=outline+shadow, 3=opaque box
	OutlineWidth     float64 // Outline thickness
	ShadowDepth      float64 // Shadow depth
	Alignment        int     // 1-9 (numpad layout)
	MarginL          int     // Left margin
	MarginR          int     // Right margin
	MarginV          int     // Vertical margin
	Encoding         int     // Character encoding
}
```

#### SubtitleLayer
```go
type SubtitleLayer struct {
	SRTFile string
	Style   SubtitleStyle
}

// Methods
layer := NewSubtitleLayer(srtFile, style)
layer := NewSubtitleLayerWithDefaults(srtFile)
layer.Prepare(outputDir)              // Generate ASS file
layer.CompositeSubtitles(input, output)  // Burn to video
layer.GetTrack()                      // Get parsed subtitles
layer.IsReady()                       // Check if prepared
```

### Functions

#### Parsing
```go
ParseSRT(path string) (*SubtitleTrack, error)
ParseSRTContent(content string) (*SubtitleTrack, error)
```

#### Generation
```go
GenerateASS(srtFile, style, outputPath) error
GenerateASSFromTrack(track, style, outputPath) error
```

#### Burning (FFmpeg Integration)
```go
BurnSubtitles(input, assFile, output) error
BurnSubtitlesWithOptions(input, assFile, output, opts) error
```

## Examples

### Example 1: Basic Subtitle Burning
```go
// Parse, generate, burn in one workflow
track, _ := subtitle.ParseSRT("output/transcript.srt")
style := subtitle.TikTokStyle()
subtitle.GenerateASSFromTrack(track, style, "output/subs.ass")
subtitle.BurnSubtitles("input.mp4", "output/subs.ass", "output.mp4")
```

### Example 2: Multiple Styles
```go
track, _ := subtitle.ParseSRT("output/transcript.srt")

styles := map[string]subtitle.SubtitleStyle{
	"tiktok":   subtitle.TikTokStyle(),
	"youtube":  subtitle.YouTubeShortsStyle(),
	"instagram": subtitle.InstagramReelsStyle(),
}

for name, style := range styles {
	assFile := fmt.Sprintf("output/%s.ass", name)
	outputFile := fmt.Sprintf("output/video_%s.mp4", name)
	
	subtitle.GenerateASSFromTrack(track, style, assFile)
	subtitle.BurnSubtitles("input.mp4", assFile, outputFile)
}
```

### Example 3: Query Subtitles by Time
```go
track, _ := subtitle.ParseSRT("output/transcript.srt")

// Get subtitle at 2.5 seconds
sub := track.GetAtTime(2.5)
if sub != nil {
	fmt.Println(sub.Text)
}

// Get all subtitles between 1-5 seconds
range := track.GetRange(1.0, 5.0)
```

### Example 4: High Quality Output
```go
opts := subtitle.BurnOptions{
	VideoCodec: "libx264",
	AudioCodec: "aac",
	Preset:     "slow",    // Higher quality
	CRF:        18,        // Better quality (lower is better)
}

subtitle.BurnSubtitlesWithOptions(
	"input.mp4",
	"subs.ass",
	"output_hq.mp4",
	opts,
)
```

## Subtitle Alignment Numbers

ASS uses numpad-style alignment (1-9):

```
7 (top-left)    8 (top-center)    9 (top-right)
4 (middle-left) 5 (middle)        6 (middle-right)
1 (bottom-left) 2 (bottom-center) 3 (bottom-right)
```

Bottom-center alignment (typical for social media): **2**

## Supported Fonts

The subtitle engine uses system fonts. Common options:
- **Montserrat** (TikTok style - recommended)
- **Roboto** (YouTube style)
- **Arial** (Standard)
- **Verdana** (Web-safe)
- **Impact** (Bold headers)
- **Courier New** (Monospace)

Font must be installed on the system for FFmpeg to use it.

## Workflow Integration

### With Video Rendering Pipeline
```go
// Step 1: Render video with effects
graph := ffmpeg.BuildFilterGraph(layers)
renderer.Render("input.mp4", "temp.mp4", graph)

// Step 2: Add subtitles
subLayer := subtitle.NewSubtitleLayerWithDefaults("transcript.srt")
subLayer.Prepare("output")
subLayer.CompositeSubtitles("temp.mp4", "output/final.mp4")
```

### With Whisper Transcription
```go
// Step 1: Transcribe audio
opts := transcription.DefaultOptions()
result, _ := transcription.TranscribeAudio("input.mp4", opts)
result.SaveSegmentsToSRT("output/transcript.srt")

// Step 2: Render with subtitles
track, _ := subtitle.ParseSRT("output/transcript.srt")
subtitle.GenerateASSFromTrack(track, subtitle.TikTokStyle(), "output/subs.ass")
subtitle.BurnSubtitles("input.mp4", "output/subs.ass", "output_with_subs.mp4")
```

## Troubleshooting

### "Font not found"
- Solution: Install font on system (or use commonly available fonts)
- Try using "Arial" as fallback

### Subtitles not appearing
- Check ASS file path is absolute path
- Verify SRT file is valid with `ParseSRT()` first
- Check FFmpeg is installed: `ffmpeg -version`

### Character encoding issues
- Set `Encoding: 1` for Unicode (default)
- Ensure SRT file is UTF-8 encoded

### FFmpeg command failed
- Verify input video exists
- Check ASS file was generated: `file output/subs.ass`
- Ensure FFmpeg path is in system PATH

## Performance Notes

- **Parsing**: SRT files parse nearly instantly
- **ASS Generation**: Minimal overhead
- **FFmpeg Burning**: Depends on video resolution and preset
  - Fast preset: ~10-30 seconds (1080p)
  - Medium preset: ~30-60 seconds (1080p)
  - Slow preset: ~2-5 minutes (1080p)

Use Fast preset for preview, Medium/Slow for final output.

## File Formats

### Input: SRT Format
```
1
00:00:01,000 --> 00:00:03,000
First subtitle

2
00:00:04,000 --> 00:00:06,000
Second subtitle
```

### Internal: ASS Format (Advanced SubStation Alpha)
- Binary subtitle format used by FFmpeg
- Contains styles, timing, and effects
- Generated automatically from SRT

### Output: MP4
- Standard video container
- H.264 video codec
- AAC audio codec

## References

- ASS Format Spec: http://www.aegisub.org/
- FFmpeg ass filter: https://ffmpeg.org/ffmpeg-filters.html#ass-ssa-subtitles
- SRT Format: https://en.wikipedia.org/wiki/SubRip
