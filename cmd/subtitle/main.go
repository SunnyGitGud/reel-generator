package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/yourname/video-editor/internal/subtitle"
)

func main() {
	// Commands
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "parse":
		cmdParse(os.Args[2:])
	case "style":
		cmdStyle(os.Args[2:])
	case "burn":
		cmdBurn(os.Args[2:])
	case "info":
		cmdInfo(os.Args[2:])
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Print(`Subtitle CLI Tool

Usage: go run cmd/subtitle/main.go <command> [options]

Commands:
  parse    Parse and display SRT file
  style    Show available styles and generate with a specific style
  burn     Burn subtitles into video
  info     Show subtitle file information

Examples:
  go run cmd/subtitle/main.go parse -file output/transcript.srt
  go run cmd/subtitle/main.go style -file output/transcript.srt -style tiktok -output output/subs.ass
  go run cmd/subtitle/main.go burn -input videos/input.mp4 -ass output/subs.ass -output output.mp4
  go run cmd/subtitle/main.go info -file output/transcript.srt
`)
}

// cmdParse parses and displays an SRT file
func cmdParse(args []string) {
	fs := flag.NewFlagSet("parse", flag.ExitOnError)
	srtFile := fs.String("file", "", "Path to SRT file")
	verbose := fs.Bool("v", false, "Verbose output")
	fs.Parse(args)

	if *srtFile == "" {
		fmt.Println("Error: -file flag required")
		os.Exit(1)
	}

	track, err := subtitle.ParseSRT(*srtFile)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("✓ Parsed SRT file: %s\n", *srtFile)
	fmt.Printf("  Total subtitles: %d\n", track.Count())
	fmt.Printf("  Duration: %.2f seconds\n\n", track.Duration())

	if *verbose {
		fmt.Println("=== SUBTITLES ===")
		for _, sub := range track.Items {
			fmt.Printf("%d. [%.2f - %.2f] %s\n", sub.Index, sub.Start, sub.End, sub.Text)
		}
	} else {
		fmt.Println("First 5 subtitles:")
		count := 5
		if track.Count() < 5 {
			count = track.Count()
		}
		for i := 0; i < count; i++ {
			sub := track.Items[i]
			fmt.Printf("%d. [%.2f - %.2f] %s\n", sub.Index, sub.Start, sub.End, sub.Text)
		}
	}
}

// cmdStyle generates ASS file with specified style
func cmdStyle(args []string) {
	fs := flag.NewFlagSet("style", flag.ExitOnError)
	srtFile := fs.String("file", "", "Path to SRT file")
	styleName := fs.String("style", "tiktok", "Style: tiktok, youtube, instagram, classic")
	outputFile := fs.String("output", "output/subs.ass", "Output ASS file path")
	fs.Parse(args)

	if *srtFile == "" {
		fmt.Println("Error: -file flag required")
		os.Exit(1)
	}

	// Parse SRT
	track, err := subtitle.ParseSRT(*srtFile)
	if err != nil {
		log.Fatal(err)
	}

	// Select style
	var style subtitle.SubtitleStyle
	switch *styleName {
	case "tiktok":
		style = subtitle.TikTokStyle()
	case "youtube":
		style = subtitle.YouTubeShortsStyle()
	case "instagram":
		style = subtitle.InstagramReelsStyle()
	case "classic":
		style = subtitle.ClassicStyle()
	default:
		fmt.Printf("Unknown style: %s\n", *styleName)
		fmt.Println("Available: tiktok, youtube, instagram, classic")
		os.Exit(1)
	}

	// Generate ASS
	err = subtitle.GenerateASSFromTrack(track, style, *outputFile)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("✓ Generated %s style: %s\n", *styleName, *outputFile)
	fmt.Printf("  Font: %s %dpt Bold\n", style.FontName, style.FontSize)
	fmt.Printf("  Color: %s text with %s outline\n", colorName(style.PrimaryColor), colorName(style.OutlineColor))
}

// cmdBurn burns subtitles into video
func cmdBurn(args []string) {
	fs := flag.NewFlagSet("burn", flag.ExitOnError)
	input := fs.String("input", "", "Input video file")
	assFile := fs.String("ass", "", "ASS subtitle file")
	output := fs.String("output", "", "Output video file")
	preset := fs.String("preset", "fast", "FFmpeg preset: fast, medium, slow")
	fs.Parse(args)

	if *input == "" || *assFile == "" || *output == "" {
		fmt.Println("Error: -input, -ass, and -output flags required")
		os.Exit(1)
	}

	opts := subtitle.BurnOptions{
		VideoCodec: "libx264",
		AudioCodec: "aac",
		Preset:     *preset,
		CRF:        23,
	}

	fmt.Printf("Burning subtitles...\n")
	fmt.Printf("  Input: %s\n", *input)
	fmt.Printf("  Subtitles: %s\n", *assFile)
	fmt.Printf("  Preset: %s\n", *preset)

	err := subtitle.BurnSubtitlesWithOptions(*input, *assFile, *output, opts)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("✓ Output: %s\n", *output)
}

// cmdInfo displays information about an SRT file
func cmdInfo(args []string) {
	fs := flag.NewFlagSet("info", flag.ExitOnError)
	srtFile := fs.String("file", "", "Path to SRT file")
	fs.Parse(args)

	if *srtFile == "" {
		fmt.Println("Error: -file flag required")
		os.Exit(1)
	}

	track, err := subtitle.ParseSRT(*srtFile)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== SUBTITLE FILE INFORMATION ===")
	fmt.Printf("File: %s\n", *srtFile)
	fmt.Printf("Total subtitles: %d\n", track.Count())
	fmt.Printf("Duration: %.2f seconds (%.2f minutes)\n", track.Duration(), track.Duration()/60)

	if track.Count() > 0 {
		first := track.Items[0]
		last := track.Items[len(track.Items)-1]

		fmt.Printf("\nFirst subtitle:\n")
		fmt.Printf("  Time: %.2f - %.2f seconds\n", first.Start, first.End)
		fmt.Printf("  Text: %s\n", first.Text)

		fmt.Printf("\nLast subtitle:\n")
		fmt.Printf("  Time: %.2f - %.2f seconds\n", last.Start, last.End)
		fmt.Printf("  Text: %s\n", last.Text)

		// Calculate average display time
		totalDisplayTime := 0.0
		for _, sub := range track.Items {
			totalDisplayTime += (sub.End - sub.Start)
		}
		avgTime := totalDisplayTime / float64(track.Count())
		fmt.Printf("\nAverage subtitle duration: %.2f seconds\n", avgTime)
	}

	fmt.Println("\n=== AVAILABLE STYLES ===")
	styles := map[string]subtitle.SubtitleStyle{
		"TikTok":    subtitle.TikTokStyle(),
		"YouTube":   subtitle.YouTubeShortsStyle(),
		"Instagram": subtitle.InstagramReelsStyle(),
		"Classic":   subtitle.ClassicStyle(),
	}

	for name, style := range styles {
		fmt.Printf("%s: %s %dpt\n", name, style.FontName, style.FontSize)
	}
}

// colorName returns a readable color name
func colorName(c subtitle.Color) string {
	if c.R == 255 && c.G == 255 && c.B == 0 {
		return "Yellow"
	} else if c.R == 255 && c.G == 255 && c.B == 255 {
		return "White"
	} else if c.R == 0 && c.G == 0 && c.B == 0 {
		return "Black"
	} else if c.R == 255 && c.G == 0 && c.B == 0 {
		return "Red"
	}
	return fmt.Sprintf("RGB(%d,%d,%d)", c.R, c.G, c.B)
}
