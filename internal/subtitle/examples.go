package subtitle

import (
	"fmt"
	"log"
)

// ExampleBasicSubtitleRendering demonstrates basic subtitle rendering
func ExampleBasicSubtitleRendering() {
	// Parse SRT file
	track, err := ParseSRT("output/transcript.srt")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Loaded %d subtitles\n", track.Count())

	// Generate ASS with TikTok style
	style := TikTokStyle()
	err = GenerateASSFromTrack(track, style, "output/subs.ass")
	if err != nil {
		log.Fatal(err)
	}

	// Burn subtitles into video
	err = BurnSubtitles("videos/input.mp4", "output/subs.ass", "output/result_with_subs.mp4")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("✓ Video with subtitles created: output/result_with_subs.mp4")
}

// ExampleSubtitleLayer demonstrates using the SubtitleLayer
func ExampleSubtitleLayer() {
	// Create subtitle layer with TikTok style
	subLayer := NewSubtitleLayerWithDefaults("output/transcript.srt")

	// Prepare the layer (parse SRT, generate ASS)
	err := subLayer.Prepare("output")
	if err != nil {
		log.Fatal(err)
	}

	// Apply to video
	err = subLayer.CompositeSubtitles("videos/input.mp4", "output/final_video.mp4")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("✓ Video with subtitles: output/final_video.mp4")
}

// ExampleCustomStyle demonstrates custom subtitle styling
func ExampleCustomStyle() {
	// Parse SRT
	track, err := ParseSRT("output/transcript.srt")
	if err != nil {
		log.Fatal(err)
	}

	// Create custom style
	customStyle := SubtitleStyle{
		Name:         "Custom",
		FontName:     "Verdana",
		FontSize:     72,
		PrimaryColor: ColorRed,
		OutlineColor: ColorBlack,
		BackColor:    ColorBlack,
		Bold:         true,
		Italic:       false,
		Underline:    false,
		ScaleX:       100,
		ScaleY:       100,
		Spacing:      1,
		Angle:        0,
		BorderStyle:  1,
		OutlineWidth: 3.5,
		ShadowDepth:  2.5,
		Alignment:    2,
		MarginL:      20,
		MarginR:      20,
		MarginV:      30,
		Encoding:     1,
	}

	// Generate ASS with custom style
	err = GenerateASSFromTrack(track, customStyle, "output/custom_subs.ass")
	if err != nil {
		log.Fatal(err)
	}

	// Burn to video
	err = BurnSubtitles("videos/input.mp4", "output/custom_subs.ass", "output/custom_video.mp4")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("✓ Custom styled video: output/custom_video.mp4")
}

// ExampleMultipleStyles demonstrates different subtitle styles
func ExampleMultipleStyles() {
	styles := map[string]SubtitleStyle{
		"tiktok":    TikTokStyle(),
		"youtube":   YouTubeShortsStyle(),
		"instagram": InstagramReelsStyle(),
		"classic":   ClassicStyle(),
	}

	track, _ := ParseSRT("output/transcript.srt")

	for name, style := range styles {
		assFile := fmt.Sprintf("output/%s_subs.ass", name)
		outputFile := fmt.Sprintf("output/video_%s.mp4", name)

		GenerateASSFromTrack(track, style, assFile)
		BurnSubtitles("videos/input.mp4", assFile, outputFile)

		fmt.Printf("✓ Generated: %s\n", outputFile)
	}
}

// ExampleSubtitleQuery demonstrates querying subtitles
func ExampleSubtitleQuery() {
	track, err := ParseSRT("output/transcript.srt")
	if err != nil {
		log.Fatal(err)
	}

	// Get subtitle at specific time
	atTime := track.GetAtTime(2.5)
	if atTime != nil {
		fmt.Printf("At 2.5s: %s\n", atTime.Text)
	}

	// Get range of subtitles
	inRange := track.GetRange(1.0, 5.0)
	fmt.Printf("Subtitles between 1-5s: %d\n", len(inRange))

	// Total duration
	fmt.Printf("Total subtitle duration: %.2f seconds\n", track.Duration())
}

// ExampleCustomBurnOptions demonstrates custom FFmpeg options
func ExampleCustomBurnOptions() {
	track, _ := ParseSRT("output/transcript.srt")

	// Generate ASS
	style := YouTubeShortsStyle()
	GenerateASSFromTrack(track, style, "output/subs.ass")

	// Custom burn options for high quality
	opts := BurnOptions{
		VideoCodec: "libx264",
		AudioCodec: "aac",
		Preset:     "slow",
		CRF:        18,
		ExtraFFmpegArgs: []string{
			"-b:v", "5000k",
		},
	}

	err := BurnSubtitlesWithOptions(
		"videos/input.mp4",
		"output/subs.ass",
		"output/hq_video.mp4",
		opts,
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("✓ High quality video with subtitles: output/hq_video.mp4")
}
