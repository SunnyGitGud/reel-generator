package subtitle

import "fmt"

// Subtitle represents a single subtitle/caption line
type Subtitle struct {
	Start float64 // Start time in seconds
	End   float64 // End time in seconds
	Text  string  // Caption text
	Index int     // Sequential index (1-based)
}

// SubtitleTrack represents a collection of subtitles
type SubtitleTrack struct {
	Items []Subtitle
}

// Color represents an RGB color
type Color struct {
	R int // 0-255
	G int // 0-255
	B int // 0-255
}

// NewColor creates a new Color from RGB values
func NewColor(r, g, b int) Color {
	return Color{R: r, G: g, B: b}
}

// ToHex converts color to hex format for ASS (BGR format)
func (c Color) ToHex() string {
	// ASS uses BGR (not RGB), so we need to reverse
	return fmt.Sprintf("&H%02X%02X%02X&", c.B, c.G, c.R)
}

// SubtitleStyle defines the visual appearance of subtitles
type SubtitleStyle struct {
	Name         string  // Style name
	FontName     string  // Font family (e.g., "Montserrat")
	FontSize     int     // Font size in pixels
	PrimaryColor Color   // Main text color
	OutlineColor Color   // Outline/stroke color
	BackColor    Color   // Background/shadow color
	Bold         bool    // Bold text
	Italic       bool    // Italic text
	Underline    bool    // Underlined text
	StrikeOut    bool    // Strike-through text
	ScaleX       float64 // Horizontal scale (0-100)
	ScaleY       float64 // Vertical scale (0-100)
	Spacing      float64 // Letter spacing
	Angle        float64 // Text rotation angle
	BorderStyle  int     // 1=outline + drop shadow, 3=opaque box
	OutlineWidth float64 // Outline thickness in pixels
	ShadowDepth  float64 // Shadow depth in pixels
	Alignment    int     // Subtitle alignment (numpad style: 1=bottom-left, 2=bottom-center, etc.)
	MarginL      int     // Left margin in pixels
	MarginR      int     // Right margin in pixels
	MarginV      int     // Vertical margin in pixels
	Encoding     int     // Character encoding
}

// Common predefined colors
var (
	ColorYellow  = NewColor(255, 255, 0)   // Yellow
	ColorBlack   = NewColor(0, 0, 0)       // Black
	ColorWhite   = NewColor(255, 255, 255) // White
	ColorRed     = NewColor(255, 0, 0)     // Red
	ColorGreen   = NewColor(0, 255, 0)     // Green
	ColorBlue    = NewColor(0, 0, 255)     // Blue
	ColorCyan    = NewColor(0, 255, 255)   // Cyan
	ColorMagenta = NewColor(255, 0, 255)   // Magenta
)

// TikTokStyle returns a modern TikTok-style subtitle preset
func TikTokStyle() SubtitleStyle {
	return SubtitleStyle{
		Name:         "TikTok",
		FontName:     "Montserrat",
		FontSize:     60,
		PrimaryColor: ColorYellow,
		OutlineColor: ColorBlack,
		BackColor:    ColorBlack,
		Bold:         true,
		Italic:       false,
		Underline:    false,
		StrikeOut:    false,
		ScaleX:       100,
		ScaleY:       100,
		Spacing:      0,
		Angle:        0,
		BorderStyle:  1, // Outline + drop shadow
		OutlineWidth: 3,
		ShadowDepth:  2,
		Alignment:    2, // Bottom center
		MarginL:      10,
		MarginR:      10,
		MarginV:      20, // Bottom margin
		Encoding:     1,  // Unicode
	}
}

// YouTubeShortsStyle returns a YouTube Shorts style preset
func YouTubeShortsStyle() SubtitleStyle {
	return SubtitleStyle{
		Name:         "YouTubeShorts",
		FontName:     "Roboto",
		FontSize:     64,
		PrimaryColor: ColorWhite,
		OutlineColor: ColorBlack,
		BackColor:    ColorBlack,
		Bold:         true,
		Italic:       false,
		Underline:    false,
		StrikeOut:    false,
		ScaleX:       100,
		ScaleY:       100,
		Spacing:      0,
		Angle:        0,
		BorderStyle:  1,
		OutlineWidth: 4,
		ShadowDepth:  3,
		Alignment:    2,
		MarginL:      15,
		MarginR:      15,
		MarginV:      25,
		Encoding:     1,
	}
}

// InstagramReelsStyle returns an Instagram Reels style preset
func InstagramReelsStyle() SubtitleStyle {
	return SubtitleStyle{
		Name:         "InstagramReels",
		FontName:     "Avenir Next",
		FontSize:     56,
		PrimaryColor: ColorWhite,
		OutlineColor: ColorBlack,
		BackColor:    ColorBlack,
		Bold:         true,
		Italic:       false,
		Underline:    false,
		StrikeOut:    false,
		ScaleX:       100,
		ScaleY:       100,
		Spacing:      2,
		Angle:        0,
		BorderStyle:  1,
		OutlineWidth: 2.5,
		ShadowDepth:  2,
		Alignment:    2,
		MarginL:      12,
		MarginR:      12,
		MarginV:      22,
		Encoding:     1,
	}
}

// ClassicStyle returns a classic white text on black outline style
func ClassicStyle() SubtitleStyle {
	return SubtitleStyle{
		Name:         "Classic",
		FontName:     "Arial",
		FontSize:     48,
		PrimaryColor: ColorWhite,
		OutlineColor: ColorBlack,
		BackColor:    ColorBlack,
		Bold:         true,
		Italic:       false,
		Underline:    false,
		StrikeOut:    false,
		ScaleX:       100,
		ScaleY:       100,
		Spacing:      0,
		Angle:        0,
		BorderStyle:  1,
		OutlineWidth: 2,
		ShadowDepth:  1,
		Alignment:    2,
		MarginL:      10,
		MarginR:      10,
		MarginV:      20,
		Encoding:     1,
	}
}

// NewSubtitleTrack creates an empty SubtitleTrack
func NewSubtitleTrack() *SubtitleTrack {
	return &SubtitleTrack{
		Items: []Subtitle{},
	}
}

// Add adds a subtitle to the track
func (st *SubtitleTrack) Add(sub Subtitle) {
	sub.Index = len(st.Items) + 1
	st.Items = append(st.Items, sub)
}

// GetAtTime returns the subtitle active at the given time in seconds
func (st *SubtitleTrack) GetAtTime(seconds float64) *Subtitle {
	for i := range st.Items {
		if st.Items[i].Start <= seconds && seconds < st.Items[i].End {
			return &st.Items[i]
		}
	}
	return nil
}

// GetRange returns all subtitles within a time range
func (st *SubtitleTrack) GetRange(startSeconds, endSeconds float64) []Subtitle {
	var result []Subtitle
	for _, sub := range st.Items {
		if sub.End > startSeconds && sub.Start < endSeconds {
			result = append(result, sub)
		}
	}
	return result
}

// Duration returns the total duration of the subtitle track
func (st *SubtitleTrack) Duration() float64 {
	if len(st.Items) == 0 {
		return 0
	}
	return st.Items[len(st.Items)-1].End
}

// Count returns the number of subtitles
func (st *SubtitleTrack) Count() int {
	return len(st.Items)
}
